package data

import (
	"time"

	"github.com/mhdna/kashi/internal/validator"
)

type Product struct {
	ID          int64     `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Year        int32     `json:"year,omitempty"`
	Price       float64   `json:"price"`
	Cost        float64   `json:"cost"`
	Category    string    `json:"category"`
	Active      bool      `json:"active"`
	Runtime     Runtime   `json:"runtime,omitempty"`
	CreatedAt   time.Time `json:"-"`
	// TODO change below to have a table of updates log
	// UpdatedAt   time.Time `json:"updated_at"`
}

func ValidateProduct(v *validator.Validator, product *Product) {
	v.Check(product.Name != "", "name", "must be provided")
	v.Check(len(product.Name) <= 500, "name", "must not be mroe than 100 bytes long")

	v.Check(product.Year != 0, "year", "must be provided")
	v.Check(product.Year >= 2000, "year", "must be greater than 2000")
	v.Check(product.Year <= int32(time.Now().Year()), "year", "must not be in the future")

	v.Check(product.Code != "", "code", "must be provided")

	// v.Check(validator.Unique(product.Tags), "tags", "must not contain duplicate values")
}
