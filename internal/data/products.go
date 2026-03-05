package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/mhdna/kashi/internal/validator"
)

type Product struct {
	ID          int64   `json:"id"`
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Year        int32   `json:"year,omitempty"`
	Kind        string  `json:"kind"`
	Type        string  `json:"type"`
	Unit        string  `json:"unit"`
	Season      string  `json:"season"`
	Price       float64 `json:"price"`
	Cost        float64 `json:"cost"`
	Category    string  `json:"category"`
	IsActive    bool    `json:"is_active"`
	Version     int     `json:"version"`
	// Runtime     Runtime   `json:"runtime,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	// TODO change below to have a table of updates log
	// UpdatedAt   time.Time `json:"updated_at"`
}

type ProductModel struct {
	DB *sql.DB
}

func (m ProductModel) Insert(product *Product) error {
	// TODO fix missing things
	query := `
		INSERT INTO products (code, name, description, kind, year, price, is_active, season, unit, type)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, version
	`

	args := []any{product.Code, product.Name, product.Description, product.Kind, product.Year, product.Price, product.IsActive, product.Season, product.Unit, product.Type}

	return m.DB.QueryRow(query, args...).Scan(&product.ID, &product.CreatedAt, &product.Version)
}

func (m ProductModel) Get(id int64) (*Product, error) {
	if id < 1 {
		return nil, ErrRecordNotFound
	}

	query := `
		SELECT id, code, name, description, kind, year, price, is_active, season, unit, type, version, created_at
		FROM products
		WHERE id = $1`

	var product Product

	err := m.DB.QueryRow(query, id).Scan(
		&product.ID,
		&product.Code,
		&product.Name,
		&product.Description,
		&product.Kind,
		&product.Year,
		&product.Price,
		&product.IsActive,
		&product.Season,
		&product.Unit,
		&product.Type,
		&product.Version,
		&product.CreatedAt,
	)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound

		default:
			return nil, err
		}
	}

	return &product, nil
}

// FINISH THE CRUDS and Race conditions
// TODO:

func (m ProductModel) Update(product *Product) error {
	query := `
	UPDATE products
	SET code = $1, name = $2, description = $3, kind = $4, year = $5, price = $6, is_active = $7, season = $8, unit = $9, type = $10, version = version + 1
	WHERE id = $11 AND version = $12
	RETURNING version`

	args := []any{
		product.Code,
		product.Name,
		product.Description,
		product.Kind,
		product.Year,
		product.Price,
		product.IsActive,
		product.Season,
		product.Unit,
		product.Type,
		product.ID,
		product.Version,
	}

	err := m.DB.QueryRow(query, args...).Scan(&product.Version)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConflict
		default:
			return err
		}
	}

	return nil
}

func (m ProductModel) Delete(id int64) error {
	if id < 1 {
		return ErrRecordNotFound
	}

	query := `
		DELETE FROM products
		WHERE id = $1
	`

	result, err := m.DB.Exec(query, id)

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil
	}
	if rowsAffected == 0 {
		return ErrRecordNotFound
	}
	return nil
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

func (m ProductModel) GetAll(code string, name string, filters Filters) ([]*Product, Metadata, error) {
	query := fmt.Sprintf(`
	SELECT count(*) OVER(), id, created_at, code, name, year, version
	FROM products
	WHERE (LOWER(code) = LOWER($1) OR $1 = '')
	AND (to_tsvector('simple', name) @@ plainto_tsquery('simple', $2) OR $2 = '')
	ORDER BY %s %s, id DESC
	LIMIT $3 OFFSET $4`, filters.sortColumn(), filters.sortDirection())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)

	defer cancel()

	args := []any{code, name, filters.limit(), filters.offset()}
	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}

	defer rows.Close()

	totalRecords := 0
	products := []*Product{}

	for rows.Next() {
		var product Product
		err := rows.Scan(
			&totalRecords,
			&product.ID,
			&product.CreatedAt,
			&product.Code,
			&product.Name,
			&product.Year,
			&product.Version,
		)
		if err != nil {
			return nil, Metadata{}, err
		}

		products = append(products, &product)
	}

	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}

	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)
	return products, metadata, nil
}
