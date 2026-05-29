package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(db *sql.DB) *SQLiteStore {
	return &SQLiteStore{db: db}
}

// User represents an OAuth2 user
type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name,omitempty"`
	Email    string `json:"email,omitempty"`
}

// OAuth2Token represents an OAuth2 token
type OAuth2Token struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	AccessToken string    `json:"access_token"`
	RefreshToken string   `json:"refresh_token,omitempty"`
	TokenType   string    `json:"token_type"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	Scope       string    `json:"scope,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// APIKey represents an API key for LLM access
type APIKey struct {
	ID          string    `json:"id"`
	KeyHash     string    `json:"-"`
	Name        string    `json:"name"`
	Role        string    `json:"role"`
	Permissions string    `json:"permissions,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// APIKeyList represents a list of API keys
type APIKeyList struct {
	Keys []*APIKey `json:"keys"`
}

// ListAPIKeys returns all API keys
func (s *SQLiteStore) ListAPIKeys() (*APIKeyList, error) {
	rows, err := s.db.Query("SELECT id, name, role, permissions, created_at, updated_at FROM api_keys")
	if err != nil {
		return nil, fmt.Errorf("failed to query API keys: %w", err)
	}
	defer rows.Close()

	var keys *APIKeyList
	keys = &APIKeyList{Keys: make([]*APIKey, 0)}

	for rows.Next() {
		var id, name, role, permissions string
		var createdAt, updatedAt time.Time

		if err := rows.Scan(&id, &name, &role, &permissions, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan API key: %w", err)
		}

		key := &APIKey{
			ID:          id,
			KeyHash:     "",
			Name:        name,
			Role:        role,
			Permissions: permissions,
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
		}

		keys.Keys = append(keys.Keys, key)
	}

	return keys, nil
}

// GetAPIKey retrieves an API key by ID
func (s *SQLiteStore) GetAPIKey(id string) (*APIKey, error) {
	var key APIKey
	err := s.db.QueryRow(
		"SELECT id, name, role, permissions, created_at, updated_at FROM api_keys WHERE id = ?", id,
	).Scan(&key.ID, &key.Name, &key.Role, &key.Permissions, &key.CreatedAt, &key.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("API key not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	return &key, nil
}

// CreateAPIKey creates a new API key
func (s *SQLiteStore) CreateAPIKey(name, role, scope string) (*APIKey, error) {
	// Generate random API key
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, fmt.Errorf("failed to generate API key: %w", err)
	}

	keyHash, err := hashString(string(keyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to hash API key: %w", err)
	}

	permissions := ""
	if scope != "" {
		permissions = fmt.Sprintf("%s,chat", scope)
	}

	result, err := s.db.Exec(
		"INSERT INTO api_keys (key_hash, name, role, permissions) VALUES (?, ?, ?, ?)",
		keyHash, name, role, permissions,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create API key: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert ID: %w", err)
	}

	key := &APIKey{
		ID:       fmt.Sprintf("%d", id),
		KeyHash:  keyHash,
		Name:     name,
		Role:     role,
		Permissions: permissions,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return key, nil
}

// DeleteAPIKey deletes an API key by ID
func (s *SQLiteStore) DeleteAPIKey(id string) error {
	result, err := s.db.Exec("DELETE FROM api_keys WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete API key: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("API key not found")
	}

	return nil
}

// VerifyAPIKey verifies an API key
func (s *SQLiteStore) VerifyAPIKey(key string) (string, error) {
	keyHash, err := hashString(key)
	if err != nil {
		return "", fmt.Errorf("failed to hash API key: %w", err)
	}

	var id string
	err = s.db.QueryRow(
		"SELECT id FROM api_keys WHERE key_hash = ?", keyHash,
	).Scan(&id)

	if err == sql.ErrNoRows {
		return "", fmt.Errorf("invalid API key")
	}
	if err != nil {
		return "", fmt.Errorf("failed to verify API key: %w", err)
	}

	return id, nil
}

// ValidateToken validates an OAuth2 token
func (s *SQLiteStore) ValidateToken(token, scope string) (*OAuth2Token, error) {
	var tokenRow OAuth2Token
	now := time.Now()

	err := s.db.QueryRow(
		"SELECT id, user_id, access_token, refresh_token, token_type, expires_at, scope, created_at FROM tokens WHERE access_token = ?", token,
	).Scan(&tokenRow.ID, &tokenRow.UserID, &tokenRow.AccessToken, &tokenRow.RefreshToken, &tokenRow.TokenType, &tokenRow.ExpiresAt, &tokenRow.Scope, &tokenRow.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invalid token")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to validate token: %w", err)
	}

	// Check expiration
	if tokenRow.ExpiresAt != nil && now.After(*tokenRow.ExpiresAt) {
		return nil, fmt.Errorf("token expired")
	}

	// Check scope
	if scope != "" && tokenRow.Scope != "" && tokenRow.Scope != scope {
		return nil, fmt.Errorf("insufficient scope")
	}

	return &tokenRow, nil
}

// CreateToken creates a new OAuth2 token
func (s *SQLiteStore) CreateToken(userID string, accessToken, refreshToken string, tokenType, scope string, expiresIn int) (*OAuth2Token, error) {
	expiresAt := time.Now().Add(time.Duration(expiresIn) * time.Second)

	result, err := s.db.Exec(
		"INSERT INTO tokens (user_id, access_token, refresh_token, token_type, expires_at, scope) VALUES (?, ?, ?, ?, ?, ?)",
		userID, accessToken, refreshToken, tokenType, expiresAt, scope,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create token: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert ID: %w", err)
	}

	token := &OAuth2Token{
		ID:           fmt.Sprintf("%d", id),
		UserID:       userID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    tokenType,
		ExpiresAt:    &expiresAt,
		Scope:        scope,
		CreatedAt:    time.Now(),
	}

	return token, nil
}

// DeleteToken deletes an OAuth2 token
func (s *SQLiteStore) DeleteToken(token string) error {
	result, err := s.db.Exec("DELETE FROM tokens WHERE access_token = ?", token)
	if err != nil {
		return fmt.Errorf("failed to delete token: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("token not found")
	}

	return nil
}

// GetOrCreateUser gets or creates a user
func (s *SQLiteStore) GetOrCreateUser(username, name, email string) (*User, error) {
	var user User
	err := s.db.QueryRow(
		"SELECT id, username, name, email FROM users WHERE username = ?", username,
	).Scan(&user.ID, &user.Username, &user.Name, &user.Email)

	if err == sql.ErrNoRows {
		// Create new user
		result, err := s.db.Exec(
			"INSERT INTO users (username, name, email) VALUES (?, ?, ?)",
			username, name, email,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}

		id, err := result.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("failed to get last insert ID: %w", err)
		}

		user = User{
			ID:       fmt.Sprintf("%d", id),
			Username: username,
			Name:     name,
			Email:    email,
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

func hashString(s string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(s), bcrypt.DefaultCost)
	return string(hash), err
}

// APIKeyList represents a list of API keys
func (l APIKeyList) String() string {
	jsonData, _ := json.Marshal(l)
	return string(jsonData)
}
