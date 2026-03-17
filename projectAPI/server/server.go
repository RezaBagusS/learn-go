package server

import (
	"belajar-go/projectAPI/internal/api/task/handler"
	"belajar-go/projectAPI/internal/middleware"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/jmoiron/sqlx"
)

type Server struct {
	mux *http.ServeMux
	db  *sqlx.DB
}

func NewServer(db *sqlx.DB) *Server {
	s := &Server{
		mux: http.NewServeMux(),
		db:  db,
	}

	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	// Inisialisasi semua domain di sini
	taskHandler := handler.NewTaskHandler(s.mux, s.db)
	taskHandler.MapRoutes()
}

func (s *Server) Run() {
	addr := os.Getenv("APP_ADDR")
	port := os.Getenv("APP_PORT")
	listen := fmt.Sprintf(":%s", port)

	fmt.Printf("Server berjalan di %s:%s\n", addr, port)
	log.Fatal(http.ListenAndServe(listen, middleware.ErrorHandling(s.mux)))
}
