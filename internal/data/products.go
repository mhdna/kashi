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
	ID            int64   `json:"id"`
	Code          string  `json:"code"`
	Name          string  `json:"name"`
	Description   string  `json:"description"`
	KindId        int64   `json:"kind_id"`
	CategoryId    int64   `json:"category_id"`
	SubCategoryId int64   `json:"sub_category_id"`
	UnitId        int64   `json:"unit_id"`
	TypeId        int64   `json:"type_id"`
	Year          int32   `json:"year,omitempty"`
	SeasonId      int64   `json:"season_id"`
	BrandId       int64   `json:"brand_id"`
	OriginId      int64   `json:"origin_id"`
	Price         float64 `json:"price"`
	IsActive      bool    `json:"is_active"`
	Version       int     `json:"version"`
	// Runtime     Runtime   `json:"runtime,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	// TODO change below to have a table of updates log
	// UpdatedAt   time.Time `json:"updated_at"`
}

type ProductModel struct {
	DB *sql.DB
}

func (m ProductModel) Insert(product *Product) error {
	query := `
		INSERT INTO products (code, name, description, kind_id, category_id, subcategory_id, unit_id, type_id, year, season_id, brand_id, origin_id, price)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, created_at, version
	`

	args := []any{product.Code, product.Name, product.Description, product.KindId, product.CategoryId, product.SubCategoryId, product.UnitId, product.TypeId, product.Year, product.SeasonId, product.BrandId, product.OriginId, product.Price}

	return m.DB.QueryRow(query, args...).Scan(&product.ID, &product.CreatedAt, &product.Version)
}

func (m ProductModel) Get(id int64) (*Product, error) {
	if id < 1 {
		return nil, ErrRecordNotFound
	}

	query := `
		SELECT id, code, name, description, kind_id, category_id, subcategory_id, unit_id, type_id, year, season_id, brand_id, origin_id, price
		FROM products
		WHERE id = $1`

	var product Product

	err := m.DB.QueryRow(query, id).Scan(
		&product.ID,
		&product.Code,
		&product.Name,
		&product.Description,
		&product.KindId,
		&product.CategoryId,
		&product.SubCategoryId,
		&product.UnitId,
		&product.TypeId,
		&product.Year,
		&product.SeasonId,
		&product.BrandId,
		&product.OriginId,
		&product.Price,
		&product.IsActive,
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
	SET code = $1, name = $2, description = $3, kind_id = $4, category_id = $5, subcategory_id = $6, unit_id = $7, type_id = $8, year = $9, season_id = $10, brand_id = $11, origin_id = $12, price = $13
	WHERE id = $14 AND version = $15
	RETURNING version`

	args := []any{
		product.Code,
		product.Name,
		product.Description,
		product.KindId,
		product.CategoryId,
		product.SubCategoryId,
		product.UnitId,
		product.TypeId,
		product.Year,
		product.SeasonId,
		product.BrandId,
		product.OriginId,
		product.Price,
		product.IsActive,
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
