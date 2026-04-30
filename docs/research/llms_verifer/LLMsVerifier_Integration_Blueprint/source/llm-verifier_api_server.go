// Package api contains HTTP API handlers
// Temporarily commented out due to dependency issues
package api

import (
	"net/http"

	"digital.vasic.llmsverifier/config"
	"digital.vasic.llmsverifier/database"
)

// Server represents the REST API server
type Server struct {
	config   *config.Config
	database *database.Database
	server   *http.Server
}

// NewServer creates a new API server
func NewServer(cfg *config.Config, db *database.Database) *Server {
	return &Server{
		config:   cfg,
		database: db,
	}
}

// Router returns the HTTP router for testing purposes
func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()

	// Register API endpoints
	mux.HandleFunc("/api/health", s.HealthHandler)
	mux.HandleFunc("/api/models", s.ListModelsHandler)
	mux.HandleFunc("/api/models/", s.GetModelHandler)
	mux.HandleFunc("/api/models/{id}/verify", s.VerifyModelHandler)
	mux.HandleFunc("/api/providers", s.ProvidersHandler)

	return mux
}

// Start starts the HTTP server on the specified port
func (s *Server) Start(port string) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/health", s.HealthHandler)
	mux.HandleFunc("/api/models", s.ListModelsHandler)
	mux.HandleFunc("/api/models/", s.GetModelHandler)
	mux.HandleFunc("/api/models/{id}/verify", s.VerifyModelHandler)
	mux.HandleFunc("/api/providers", s.ProvidersHandler)

	if port == "" {
		port = "8080"
	}

	s.server = &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	return s.server.ListenAndServe()
}

// Stop stops the HTTP server
func (s *Server) Stop() error {
	if s.server != nil {
		return s.server.Close()
	}
	return nil
}
