package utils

import "time"

type AIResult struct {
}
type Payment struct {
	ID       uint      `gorm:"primaryKey"`
	ChatID   int64     `gorm:"index" json:"chat_id"`
	Category string    `gorm:"not null"`
	Object   string    `gorm:"not null"`
	Price    float64   `gorm:"not null"`
	DatePaid time.Time `gorm:"not null;default:now()"`
}
