package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgconn"
	db "github.com/karthikbhandary2/chat/internal/db/sqlc"
	"golang.org/x/crypto/bcrypt"
)

var validate = validator.New()

type RegisterRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	FullName string `json:"full_name" validate:"required,max=50"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var s RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if err := validate.Struct(&s); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	hashPassword, err := bcrypt.GenerateFromPassword([]byte(s.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Println("bcrypt error:", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	user, err := h.Store.CreateUser(r.Context(), db.CreateUserParams{
		Username:       s.Username,
		Email:          s.Email,
		PasswordHash: string(hashPassword),
		FullName:       s.FullName,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			http.Error(w, "username or email already exists", http.StatusConflict)
			return
		}
		log.Println("create user error:", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var l LoginRequest 
	if err := json.NewDecoder(r.Body).Decode(&l); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if err := validate.Struct(&l); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	user, err := h.Store.GetUserByEmail(r.Context(), l.Email)
	if err != nil {
		log.Println("Get user by email error: ", err)
		http.Error(w, "invalid email or password", http.StatusUnauthorized)
		return
	}
	ok := checkPassword(user.PasswordHash, l.Password)
	if !ok {
		http.Error(w, "invalid email or password", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(toUserResponse(user))
}

func checkPassword(hashedPassword, password string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
    return err == nil
}