package models

import "time"

// Expense представляет запись о расходе
type Expense struct {
	ID        uint  `gorm:"primaryKey"`
	UserID    int64 `gorm:"index"`
	Amount    float64
	Category  string
	CreatedAt time.Time
}
