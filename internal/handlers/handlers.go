package handlers

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/karthikbhandary2/chat/internal/db/sqlc"
	"github.com/redis/go-redis/v9"
)

type Handler struct {
	Store       db.Store
	RedisClient *redis.Client
}

type UserResponse struct {
	ID        uuid.UUID          `json:"id"`
	Username  string             `json:"username"`
	Email     string             `json:"email"`
	FullName  string             `json:"full_name"`
	CreatedAt pgtype.Timestamptz `json:"created_at"`
}

func toUserResponse(u db.User) UserResponse {
	return UserResponse{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		FullName:  u.FullName,
		CreatedAt: u.CreatedAt,
	}
}

func NewHandler(store db.Store, redisClient *redis.Client) *Handler {
	return &Handler{
		Store:       store,
		RedisClient: redisClient,
	}
}
