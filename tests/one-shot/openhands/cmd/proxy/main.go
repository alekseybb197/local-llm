// OAuth2 Proxy for Local LLM with OpenAI-compatible API
// This proxy provides authentication and routes requests to an upstream LLM service.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/openhands/oauth2-proxy/pkg/config"
	"github.com/openhands/oauth2-proxy/pkg/proxy"
)

func main() {
	// Parse command line flags
	readTimeout := flag.Duration("read-timeout", 30*time.Second, "Read timeout")
	writeTimeout := flag.Duration("write-timeout", 30*time.Second, "Write timeout")
	idleTimeout := flag.Duration("idle-timeout", 60*time.Second, "Idle timeout")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Override with command line flags if provided
	if *readTimeout != 0 {
		cfg.ServerConfig.ReadTimeout = *readTimeout
	}
	if *writeTimeout != 0 {
		cfg.ServerConfig.WriteTimeout = *writeTimeout
	}
	if *idleTimeout != 0 {
		cfg.ServerConfig.IdleTimeout = *idleTimeout
	}

	// Create proxy
	p, err := proxy.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create proxy: %v", err)
	}

	// Create OpenAI proxy
	openaiProxy := proxy.NewOpenAIProxy(p)

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.ServerConfig.Host, cfg.ServerConfig.Port),
		ReadTimeout:  cfg.ServerConfig.ReadTimeout,
		WriteTimeout: cfg.ServerConfig.WriteTimeout,
		IdleTimeout:  cfg.ServerConfig.IdleTimeout,
	}

	// Create mux
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// OAuth2 endpoints
	mux.HandleFunc("/oauth2/auth", func(w http.ResponseWriter, r *http.Request) {
		p.AuthHandler(w, r)
	})
	mux.HandleFunc("/oauth2/callback", func(w http.ResponseWriter, r *http.Request) {
		p.CallbackHandler(w, r)
	})

	// OpenAI-compatible endpoints
	mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		openaiProxy.ChatCompletionHandler(w, r)
	})
	mux.HandleFunc("/v1/completions", func(w http.ResponseWriter, r *http.Request) {
		openaiProxy.CompletionsHandler(w, r)
	})
	mux.HandleFunc("/v1/models", func(w http.ResponseWriter, r *http.Request) {
		openaiProxy.ModelsHandler(w, r)
	})

	// Protected endpoints
	mux.HandleFunc("/api/protected", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		w.Write([]byte("Protected endpoint accessed"))
	})

	// Public endpoint
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>OAuth2 Proxy for Local LLM</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            color: white;
        }
        .container {
            max-width: 600px;
            padding: 2rem;
        }
        h1 {
            font-size: 2.5rem;
            margin-bottom: 0.5rem;
            text-align: center;
        }
        .subtitle {
            text-align: center;
            margin-bottom: 2rem;
            opacity: 0.9;
        }
        .card {
            background: rgba(255, 255, 255, 0.1);
            backdrop-filter: blur(10px);
            border-radius: 16px;
            padding: 2rem;
            margin-bottom: 1rem;
            border: 1px solid rgba(255, 255, 255, 0.2);
        }
        .btn {
            display: inline-block;
            padding: 0.875rem 1.75rem;
            background: white;
            color: #667eea;
            text-decoration: none;
            border-radius: 8px;
            font-weight: 600;
            transition: all 0.3s ease;
            cursor: pointer;
            border: none;
            font-size: 1rem;
        }
        .btn:hover {
            transform: translateY(-2px);
            box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
        }
        .info {
            margin-top: 1rem;
            padding: 1rem;
            background: rgba(255, 255, 255, 0.1);
            border-radius: 8px;
        }
        .info h3 {
            margin-bottom: 0.5rem;
        }
        .info p {
            opacity: 0.8;
            line-height: 1.5;
            margin-bottom: 0.5rem;
        }
        code {
            background: rgba(0, 0, 0, 0.3);
            padding: 0.25rem 0.5rem;
            border-radius: 4px;
            font-family: 'Monaco', 'Consolas', monospace;
            font-size: 0.9rem;
        }
        .features {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 1rem;
            margin-top: 1rem;
        }
        .feature {
            background: rgba(255, 255, 255, 0.1);
            padding: 1rem;
            border-radius: 8px;
            text-align: center;
        }
        .feature h4 {
            margin-bottom: 0.5rem;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>🔐 OAuth2 Proxy</h1>
        <p class="subtitle">Secure access to your local LLM</p>
        
        <div class="card">
            <h2>Get Started</h2>
            <p style="margin-top: 1rem; margin-bottom: 1.5rem; opacity: 0.9;">
                Click the button below to log in and access the AI API.
            </p>
            <a href="/oauth2/auth" class="btn">Log In with OAuth2</a>
        </div>

        <div class="card">
            <h3>📖 How to Use</h3>
            <div class="info">
                <p><strong>API Endpoints:</strong></p>
                <p>Make requests to <code>/v1/chat/completions</code> with authentication:</p>
                <pre style="margin: 0.5rem 0; overflow-x: auto; background: rgba(0, 0, 0, 0.3); padding: 0.5rem; border-radius: 4px;"><code>curl -X POST "http://localhost:8080/v1/chat/completions" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama3",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'</code></pre>
            </div>
        </div>

        <div class="card">
            <h3>🚀 Features</h3>
            <div class="features">
                <div class="feature">
                    <h4>🔒 OAuth2 Auth</h4>
                    <p>Secure authorization code flow</p>
                </div>
                <div class="feature">
                    <h4>⚡ OpenAI API</h4>
                    <p>Compatible API format</p>
                </div>
                <div class="feature">
                    <h4>🎯 Session Mgmt</h4>
                    <p>Token-based sessions</p>
                </div>
            </div>
        </div>

        <div class="info">
            <p><strong>Health Check:</strong> <a href="/health" style="color: white; text-decoration: underline;">/health</a></p>
            <p><strong>API Docs:</strong> <a href="/v1/chat/completions" style="color: white; text-decoration: underline;">/v1/chat/completions</a></p>
        </div>
    </div>
</body>
</html>
`))
	})

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Fatalf("Server shutdown error: %v", err)
		}
	}()

	log.Printf("Server starting on %s:%d", cfg.ServerConfig.Host, cfg.ServerConfig.Port)
	log.Printf("OAuth2 Callback: %s", cfg.OAuth2Config.RedirectURI)
	log.Printf("Upstream LLM: %s", cfg.ProxyConfig.LLMEndpoint)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}
