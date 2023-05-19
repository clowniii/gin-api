package models

import (
	"time"
)

type User struct {
	ID        uint      `db:"id"`
	Name      string    `db:"name" binding:"required"`
	Age       int       `db:"age"`
	Email     string    `db:"email"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}
