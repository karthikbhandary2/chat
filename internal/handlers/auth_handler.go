package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/karthikbhandary2/chat/internal/auth"
	db "github.com/karthikbhandary2/chat/internal/db/sqlc"
	"github.com/karthikbhandary2/chat/internal/middleware"
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

type LoginResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

type MeResponse struct {
	ID string `json:"id"`
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
		Username:     s.Username,
		Email:        s.Email,
		PasswordHash: string(hashPassword),
		FullName:     s.FullName,
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

	token, err := auth.GenerateToken(user.ID.String())
	if err != nil {
		log.Println("generate token error: ", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	lr := LoginResponse{
		Token: token,
		User:  toUserResponse(user),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(lr)
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	idStr, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		log.Println("invalid user id in token:", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	user, err := h.Store.GetUserByID(r.Context(), id)
	if err != nil {
		log.Println("get user by id error:", err)
		http.Error(w, "user not found", http.StatusNotFound)
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
