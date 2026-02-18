package models

import "time"

type Product struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	SKU         string    `json:"sku" gorm:"unique"`
	Name        string    `json:"name" gorm:"not null"`
	Description string    `json:"description"`
	Price       float64   `json:"price" gorm:"not null"`
	Cost        float64   `json:"cost"`
	Category    string    `json:"category"`
	Active      bool      `json:"active" gorm:"default:true"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
