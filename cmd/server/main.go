package main

import (
	"context"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/karthikbhandary2/chat/internal/db"
	"github.com/karthikbhandary2/chat/internal/handlers"
)

func main() {
	ctx := context.Background()

	pool, err := db.NewPool(ctx)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	defer pool.Close()

	r := chi.NewRouter()
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", handlers.Health)
	})
	log.Println("starting server on port: 8082")
	log.Fatal(http.ListenAndServe(":8082", r))

}
