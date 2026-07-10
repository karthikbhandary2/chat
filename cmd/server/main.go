package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	db "github.com/karthikbhandary2/chat/internal/db/sqlc"
	"github.com/karthikbhandary2/chat/internal/handlers"
	"github.com/karthikbhandary2/chat/internal/middleware"
	"github.com/karthikbhandary2/chat/internal/ws"
)

var interruptSignals = []os.Signal{
	os.Interrupt,
	syscall.SIGTERM,
	syscall.SIGINT,
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, relying on real environment variables")
	}

	ctx, stop := signal.NotifyContext(context.Background(), interruptSignals...)
	defer stop()

	connPool, err := pgxpool.New(ctx, os.Getenv("DB_URL"))
	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}
	if err := connPool.Ping(ctx); err != nil {
		log.Fatal("db ping failed: ", err)
	}

	store := db.NewStore(connPool)
	h := handlers.NewHandler(store)

	hub := ws.NewHub(store)
	go hub.Run()

	r := chi.NewRouter()
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", h.Health)
		r.Post("/register", h.Register)
		r.Post("/login", h.Login)

		r.Group(func(r chi.Router) {
			r.Use(middleware.AuthMiddleware)
			r.Get("/me", h.Me)
		})
	})
	webserver := ws.NewServer(hub)
	r.Get("/ws", webserver.HandleConnections)
	r.Handle("/*", http.FileServer(http.Dir("./web")))
	log.Println("starting server on port: 8082")
	log.Fatal(http.ListenAndServe(":8082", r))

}
