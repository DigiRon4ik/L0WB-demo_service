// Package server provides the implementation of the HTTP API server that handles
// requests related to orders. It defines the APIServer struct, which holds the
// configuration, router, and orderer for interacting with orders. The server
// exposes an HTTP endpoint to retrieve an order by its unique identifier (UID).
package server

import (
	"context"
	"demo_service/internal/config"
	"demo_service/internal/models"
	"encoding/json"
	"net/http"
	"time"
)

// Orderer defines the methods for interacting with orders,
// including retrieving an order by its UID.
type Orderer interface {
	GetOrder(ctx context.Context, orderUID string) (*models.Order, error)
}

// APIServer represents the HTTP API server with configuration, router, context,
// and orderer for handling requests.
type APIServer struct {
	config *config.HTTPServer
	router *http.ServeMux
	ctx    context.Context
	ord    Orderer
}

// New creates a new APIServer instance with the provided context,
// ordererModule, and server configuration.
func New(ctx context.Context, ord Orderer, config *config.HTTPServer) *APIServer {
	router := http.NewServeMux()

	return &APIServer{
		config: config,
		router: router,
		ctx:    ctx,
		ord:    ord,
	}
}

// Start initializes the HTTP server with specified timeout settings and router,
// then starts listening for requests.
func (s *APIServer) Start() error {
	s.configureRouter()
	server := &http.Server{
		Addr:         s.config.Address,
		Handler:      s.router,
		ReadTimeout:  30 * time.Second,  // Request read timeout
		WriteTimeout: 10 * time.Second,  // Response Record Timeout
		IdleTimeout:  120 * time.Second, // Keep-alive connections timeout
	}

	return server.ListenAndServe()
}

func (s *APIServer) getOrder(w http.ResponseWriter, r *http.Request) {
	uid := r.URL.Path[len("/order/"):]
	if order, err := s.ord.GetOrder(s.ctx, uid); err != nil {
		http.Error(w, "Order not found", http.StatusNotFound)
	} else {
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(order)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func (s *APIServer) configureRouter() {
	s.router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "internal/templates/index.html")
	})
	s.router.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("internal/templates/static/"))))
	s.router.HandleFunc("/order/", s.getOrder)
}
