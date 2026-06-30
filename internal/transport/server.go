package transport

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/websocket"
	"yga.my.id/cscom/internal/metrics"
)

// Server wraps the chi router and the metrics collector.
type Server struct {
	router    chi.Router
	collector *metrics.Collector
	authKey   string
	sessions  *sessionStore
	staticDir string
}

// New creates a Server that serves REST + WebSocket endpoints and static files
// from staticDir. If authKey is non-empty, all endpoints require authentication.
func New(collector *metrics.Collector, staticDir string, authKey string) *Server {
	s := &Server{
		collector: collector,
		authKey:   authKey,
		sessions:  newSessionStore(),
		staticDir: staticDir,
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Unprotected: login page and login API
	r.Get("/login", s.serveLogin)
	r.Post("/api/login", s.handleLogin)

	// Protected routes
	r.Group(func(r chi.Router) {
		if authKey != "" {
			r.Use(s.authMiddleware)
		}
		r.Get("/api/metrics", s.handleMetrics)
		r.Get("/ws", s.handleWS)
		r.Post("/api/logout", s.handleLogout)
		r.Handle("/*", http.FileServer(http.Dir(staticDir)))
	})

	s.router = r
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	snap := s.collector.Latest()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(snap); err != nil {
		log.Printf("metrics encode: %v", err)
	}
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true // non-browser client; auth middleware already gated it
			}
			u, err := url.Parse(origin)
			return err == nil && u.Host == r.Host
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade: %v", err)
		return
	}
	defer conn.Close()

	// read loop: detects client disconnect so the write loop can exit
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			data, err := json.Marshal(s.collector.Latest())
			if err != nil {
				return
			}
			conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}
		}
	}
}
