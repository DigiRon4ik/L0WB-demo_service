package server

import (
	"context"
	"demo_service/internal/config"
	"demo_service/internal/models"
	"encoding/json"
	"net/http"
	"time"
)

type APIServer struct {
	config *config.HTTPServer
	router *http.ServeMux
	ctx    context.Context
	ord    Orderer
}

type Orderer interface {
	GetOrder(ctx context.Context, orderUID string) (*models.Order, error)
}

func New(ctx context.Context, ord Orderer, config *config.HTTPServer) *APIServer {
	router := http.NewServeMux()

	return &APIServer{
		config: config,
		router: router,
		ctx:    ctx,
		ord:    ord,
	}
}

func (s *APIServer) Start() error {
	s.configureRouter()
	server := &http.Server{
		Addr:         s.config.Address,
		Handler:      s.router,
		ReadTimeout:  10 * time.Second,  // Таймаут чтения запроса
		WriteTimeout: 10 * time.Second,  // Таймаут записи ответа
		IdleTimeout:  120 * time.Second, // Таймаут ожидания keep-alive соединений
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
	s.router.HandleFunc("GET /order/{uid}", s.getOrder)
}
