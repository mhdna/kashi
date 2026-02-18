package models

import "time"

type User struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Username  string    `json:"username" gorm:"unique;not null"`
	Password  string    `json:"-" gorm:"not null"` // "-" prevents JSON serialization
	Name      string    `json:"name"`
	Role      string    `json:"role"` // "admin", "manager", "cashier"
	BranchID  uint      `json:"branch_id"`
	Branch    *Branch   `json:"branch,omitempty" gorm:"foreignKey:BranchID"`
	Active    bool      `json:"active" gorm:"default:true"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
