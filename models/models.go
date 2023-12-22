package models

import (
	"github.com/google/uuid"
	"time"
)

type User struct {
	Id        uuid.UUID `json:"id" gorm:"default:uuid_generate_v4()"`
	Phone     string    `json:"phone"`
	Password  string    `json:"-"`
	Roles     string    `json:"roles" gorm:"default:user"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Surname   string    `json:"surname"`
	Thread    string    `json:"thread"`
	CreatedAt time.Time `json:"createdAt" gorm:"default:now()"`
}

type Message struct {
	Role string `json:"role,omitempty"`
	Text string `json:"text"`
}
