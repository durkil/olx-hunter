package database

import (
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type User struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	TelegramID int64     `json:"telegram_id" gorm:"uniqueIndex;not null"`
	Username   string    `json:"username" gorm:"size:50"`
	FirstName  string    `json:"first_name" gorm:"size:100"`
	CreatedAt  time.Time `json:"created_at"`

	Filters []UserFilter `json:"filters" gorm:"foreignKey:UserID"`
}

type UserFilter struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"not null"`
	Name      string    `json:"name" gorm:"size:100;not null"`
	Query     string    `json:"query" gorm:"size:100;not null"`
	MinPrice  int       `json:"min_price" gorm:"default:0"`
	MaxPrice  int       `json:"max_price" gorm:"default:0"`
	City      string    `json:"city" gorm:"size:50"`
	IsActive  bool      `json:"is_active" gorm:"default:true"`
	CreatedAt time.Time `json:"created_at"`
}

type DB struct {
	*gorm.DB
}

func Connect(dsn string) (*DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	return &DB{db}, nil
}
