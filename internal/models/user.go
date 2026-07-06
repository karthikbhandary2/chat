package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	Id           uuid.UUID	`json:"id" db:"id"`
	UserName     string	`json:"username" db:"username" validate:"required,min=3,max=50"`
	Email        string	`json:"email" db:"email" validate:"required,email,max=255"`
	PasswordHash string `json:"-" db:"password_hash"` // json:"-" skips the password_hash from unmarshalling.
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}
