package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/oauth2-proxy/local-llm-proxy/models"
)

type Database struct {
	db *sql.DB
}

func NewDatabase(path string) (*Database, error) {
	dir := "./data"
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create data directory: %w", err)
		}
	}

	db, err := sql.Open("sqlite3", path+"?_foreign_keys=on&_journal_wal=1")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	_, err = db.Exec("PRAGMA journal_mode=WAL;")
	if err != nil {
		log.Printf("Warning: failed to enable WAL mode: %v", err)
	}

	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return &Database{db: db}, nil
}

func (d *Database) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

func (d *Database) WithTx() (*sql.Tx, error) {
	return d.db.Begin()
}

func runMigrations(db *sql.DB) error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS clients (
			id TEXT PRIMARY KEY,
			client_id TEXT UNIQUE NOT NULL,
			client_secret TEXT NOT NULL,
			redirect_uri TEXT NOT NULL,
			scopes TEXT DEFAULT '["openid", "profile", "email"]',
			grant_types TEXT DEFAULT '["authorization_code", "refresh_token"]',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS tokens (
			id TEXT PRIMARY KEY,
			access_token TEXT NOT NULL,
			refresh_token TEXT,
			expires_at DATETIME NOT NULL,
			token_type TEXT DEFAULT 'Bearer',
			scopes TEXT NOT NULL,
			subject TEXT NOT NULL,
			client_id TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS codes (
			id TEXT PRIMARY KEY,
			code TEXT NOT NULL,
			client_id TEXT NOT NULL,
			redirect_uri TEXT NOT NULL,
			expires_at DATETIME NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			scopes TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS pkce_verifiers (
			id TEXT PRIMARY KEY,
			code_verifier TEXT NOT NULL,
			code_challenge TEXT NOT NULL,
			code_challenge_method TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE INDEX IF NOT EXISTS idx_tokens_expires ON tokens(expires_at);`,
		`CREATE INDEX IF NOT EXISTS idx_codes_expires ON codes(expires_at);`,
		`CREATE INDEX IF NOT EXISTS idx_pkce_verifiers_challenge ON pkce_verifiers(code_challenge);`,
	}

	for _, migration := range migrations {
		_, err := db.Exec(migration)
		if err != nil {
			return fmt.Errorf("failed to execute migration: %w", err)
		}
	}
	return nil
}

func (d *Database) CreateClient(client *models.Client) error {
	b, _ := json.Marshal(client.Scopes)
	scopesJSON := string(b)
	b, _ = json.Marshal(client.GrantTypes)
	grantTypesJSON := string(b)

	_, err := d.db.Exec(`
		INSERT INTO clients (id, client_id, client_secret, redirect_uri, scopes, grant_types, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, client.ID, client.ClientID, client.ClientSecret, client.RedirectURI, scopesJSON, grantTypesJSON, client.CreatedAt)
	return err
}

func (d *Database) GetClient(clientID string) (*models.Client, error) {
	var client models.Client
	var scopesJSON, grantTypesJSON string

	err := d.db.QueryRow(`
		SELECT id, client_id, client_secret, redirect_uri, scopes, grant_types, created_at
		FROM clients WHERE client_id = ?
	`, clientID).Scan(&client.ID, &client.ClientID, &client.ClientSecret, &client.RedirectURI, &scopesJSON, &grantTypesJSON, &client.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("client not found")
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(scopesJSON), &client.Scopes); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(grantTypesJSON), &client.GrantTypes); err != nil {
		return nil, err
	}
	return &client, nil
}

func (d *Database) DeleteClient(clientID string) error {
	_, err := d.db.Exec(`DELETE FROM clients WHERE client_id = ?`, clientID)
	return err
}

func (d *Database) CreateToken(token *models.Token) error {
	_, err := d.db.Exec(`
		INSERT INTO tokens (id, access_token, refresh_token, expires_at, token_type, scopes, subject, client_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, token.ID, token.AccessToken, token.RefreshToken, token.ExpiresAt, token.TokenType, token.Scopes, token.Subject, token.ClientID, token.CreatedAt)
	return err
}

func (d *Database) GetToken(tokenID string) (*models.Token, error) {
	var token models.Token
	var scopesJSON string

	err := d.db.QueryRow(`
		SELECT id, access_token, refresh_token, expires_at, token_type, scopes, subject, client_id, created_at
		FROM tokens WHERE id = ?
	`, tokenID).Scan(&token.ID, &token.AccessToken, &token.RefreshToken, &token.ExpiresAt, &token.TokenType, &scopesJSON, &token.Scopes, &token.ClientID, &token.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("token not found")
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(scopesJSON), &token.Scopes); err != nil {
		return nil, err
	}
	return &token, nil
}

func (d *Database) DeleteToken(tokenID string) error {
	_, err := d.db.Exec(`DELETE FROM tokens WHERE id = ?`, tokenID)
	return err
}

func (d *Database) DeleteExpiredTokens() (int64, error) {
	result, err := d.db.Exec(`DELETE FROM tokens WHERE expires_at < CURRENT_TIMESTAMP`)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (d *Database) IsTokenValid(tokenID string) (bool, error) {
	var exists bool
	var expiresAt time.Time

	err := d.db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM tokens WHERE id = ?), expires_at
	`, tokenID).Scan(&exists, &expiresAt)

	if err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}
	return time.Now().Before(expiresAt), nil
}

func (d *Database) GetTokenByAccessToken(accessToken string) (*models.Token, error) {
	var token models.Token
	var scopesJSON string

	err := d.db.QueryRow(`
		SELECT id, access_token, refresh_token, expires_at, token_type, scopes, subject, client_id, created_at
		FROM tokens WHERE access_token = ?
	`, accessToken).Scan(&token.ID, &token.AccessToken, &token.RefreshToken, &token.ExpiresAt, &token.TokenType, &scopesJSON, &token.Scopes, &token.ClientID, &token.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("token not found")
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(scopesJSON), &token.Scopes); err != nil {
		return nil, err
	}
	return &token, nil
}

func (d *Database) CreateCode(code *models.OAuthCode) error {
	_, err := d.db.Exec(`
		INSERT INTO codes (id, code, client_id, redirect_uri, expires_at, created_at, scopes)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, code.ID, code.Code, code.ClientID, code.RedirectURI, code.ExpiresAt, code.CreatedAt, code.Scopes)
	return err
}

func (d *Database) GetCodeByCode(code string) (*models.OAuthCode, error) {
	var codeObj models.OAuthCode
	var scopesJSON string

	err := d.db.QueryRow(`
		SELECT id, code, client_id, redirect_uri, expires_at, created_at, scopes
		FROM codes WHERE code = ?
	`, code).Scan(&codeObj.ID, &codeObj.Code, &codeObj.ClientID, &codeObj.RedirectURI, &codeObj.ExpiresAt, &codeObj.CreatedAt, &scopesJSON)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("code not found")
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(scopesJSON), &codeObj.Scopes); err != nil {
		return nil, err
	}
	return &codeObj, nil
}

func (d *Database) DeleteCodeByCode(code string) error {
	_, err := d.db.Exec(`DELETE FROM codes WHERE code = ?`, code)
	return err
}

func (d *Database) IsCodeExpiredByCode(code string) (bool, error) {
	var expiresAt time.Time
	err := d.db.QueryRow(`SELECT expires_at FROM codes WHERE code = ?`, code).Scan(&expiresAt)
	if err != nil {
		return false, err
	}
	return time.Now().After(expiresAt), nil
}

func (d *Database) CreatePKCECodeVerifier(verifier *models.PKCECodeVerifier) error {
	_, err := d.db.Exec(`
		INSERT INTO pkce_verifiers (id, code_verifier, code_challenge, code_challenge_method, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, verifier.ID, verifier.CodeVerifier, verifier.CodeChallenge, verifier.CodeChallengeMethod, verifier.CreatedAt)
	return err
}

func (d *Database) GetPKCECodeVerifier(codeChallenge string) (*models.PKCECodeVerifier, error) {
	var verifier models.PKCECodeVerifier
	var createdAt string

	err := d.db.QueryRow(`
		SELECT id, code_verifier, code_challenge, code_challenge_method, created_at
		FROM pkce_verifiers WHERE code_challenge = ?
	`, codeChallenge).Scan(&verifier.ID, &verifier.CodeVerifier, &verifier.CodeChallenge, &verifier.CodeChallengeMethod, &createdAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("verifier not found")
	}
	if err != nil {
		return nil, err
	}

	verifier.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return &verifier, nil
}

func (d *Database) DeletePKCECodeVerifier(verifierID string) error {
	_, err := d.db.Exec(`DELETE FROM pkce_verifiers WHERE id = ?`, verifierID)
	return err
}

func (d *Database) DeleteExpiredCodes() (int64, error) {
	result, err := d.db.Exec(`DELETE FROM codes WHERE expires_at < CURRENT_TIMESTAMP`)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
