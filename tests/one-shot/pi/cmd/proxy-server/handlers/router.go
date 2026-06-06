package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"oauth2-proxy/pkg/oauth2"
	"oauth2-proxy/pkg/proxy"
)

type Router struct {
	router     *http.ServeMux
	oauthServer *oauth.Server
	llmProxy    *proxy.LLMProxy
}

func NewRouter(
	oauthServer *oauth.Server,
	llmProxy *proxy.LLMProxy,
	corsOrigins []string,
) *Router {
	r := &Router{
		router:     http.NewServeMux(),
		oauthServer: oauthServer,
		llmProxy:   llmProxy,
	}

	r.setupHandlers(corsOrigins)
	return r
}

func (r *Router) setupHandlers(corsOrigins []string) {
	// OAuth2 handlers
	r.router.HandleFunc("/oauth2/authorize", r.handleAuthorize)
	r.router.HandleFunc("/oauth2/callback", r.handleCallback)
	r.router.HandleFunc("/oauth2/logout", r.handleLogout)
	r.router.HandleFunc("/oauth2/login", r.handleLogin)

	// OAuth2 admin endpoints
	r.router.HandleFunc("/oauth2/admin/api-keys", r.handleListAPIKeys)
	r.router.HandleFunc("/oauth2/admin/api-keys/{id}", r.handleGetAPIKey)
	r.router.HandleFunc("/oauth2/admin/api-keys", r.handleCreateAPIKey)
	r.router.HandleFunc("/oauth2/admin/api-keys/{id}", r.handleDeleteAPIKey)

	// Health check
	r.router.HandleFunc("/health", r.handleHealth)

	// LLM proxy routes
	r.router.Handle("/v1/*", r)
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if strings.HasPrefix(req.URL.Path, "/v1/") {
		if r.llmProxy != nil {
			path := strings.TrimPrefix(req.URL.Path, "/v1/")
			r.llmProxy.Handle(w, req, path)
			return
		}
	}
	r.router.ServeHTTP(w, req)
}

func (r *Router) handleAuthorize(w http.ResponseWriter, req *http.Request) {
	r.oauthServer.HandleAuthorize(w, req)
}

func (r *Router) handleCallback(w http.ResponseWriter, req *http.Request) {
	r.oauthServer.HandleCallback(w, req)
}

func (r *Router) handleLogout(w http.ResponseWriter, req *http.Request) {
	http.Redirect(w, req, "/", http.StatusFound)
}

func (r *Router) handleLogin(w http.ResponseWriter, req *http.Request) {
	http.Redirect(w, req, "/", http.StatusFound)
}

func (r *Router) handleHealth(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy","service":"oauth2-proxy"}`))
}

func (r *Router) handleListAPIKeys(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"keys":[]}`))
}

func (r *Router) handleGetAPIKey(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"key":{}}`))
}

func (r *Router) handleCreateAPIKey(w http.ResponseWriter, req *http.Request) {
	if err := jsonBody(w, req, nil); err != nil {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"key":{}}`))
}

func (r *Router) handleDeleteAPIKey(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func jsonBody(w http.ResponseWriter, r *http.Request, v interface{}) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(v)
}
