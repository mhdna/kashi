package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mhdna/kashi/internal/validator"
)

var (
	ErrDuplicateName     = errors.New("duplicate name")
	ErrDuplicateHexValue = errors.New("duplicate hex_value")
)

type Color struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	HexValue string `json:"hex_value"`
	Version  int32  `json:"version"`
}

type ColorModel struct {
	DB *sql.DB
}

func (m ColorModel) Insert(color *Color) error {
	query := `
		INSERT INTO COLORS (name, hex_value)
		values ($1, $2)
		returning id, version`

	args := []any{color.Name, color.HexValue}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, args...).Scan(&color.ID, &color.Version)
	if err != nil {
		switch {
		case strings.Contains(err.Error(), `duplicate key value violates unique constraint "colors_name_key"`):
			return ErrDuplicateName
		case strings.Contains(err.Error(), `duplicate key value violates unique constraint "colors_hex_value_key"`):
			return ErrDuplicateHexValue
		default:
			return err
		}
	}
	return nil
}

func (m ColorModel) Get(id int64) (*Color, error) {
	if id < 1 {
		return nil, ErrRecordNotFound
	}

	query := `
		SELECT id, name, hex_value
		FROM colors
		WHERE id = $1`

	var color Color

	err := m.DB.QueryRow(query, id).Scan(
		&color.ID,
		&color.Name,
		&color.HexValue,
	)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound

		default:
			return nil, err
		}
	}

	return &color, nil
}

func (m ColorModel) Update(color *Color) error {
	query := `
	UPDATE colors
	SET name = $1, hex_value = $2
	WHERE id = $3 AND version = $4
	RETURNING version`

	args := []any{
		color.Name,
		color.HexValue,
		color.ID,
		color.Version,
	}

	err := m.DB.QueryRow(query, args...).Scan(&color.Version)

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

func (m ColorModel) Delete(id int64) error {
	if id < 1 {
		return ErrRecordNotFound
	}

	query := `
		DELETE FROM colors
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

func ValidateColor(v *validator.Validator, color *Color) {
	v.Check(color.Name != "", "name", "must be provided")
	v.Check(len(color.Name) <= 500, "name", "must not be mroe than 100 bytes long")

	v.Check(color.HexValue != "", "hex_value", "must be provided")
	v.Check(len(color.HexValue) == 6, "name", "must 6 bytes long")

	// v.Check(validator.Unique(color.Tags), "tags", "must not contain duplicate values")
}

func (m ColorModel) GetAll(name string, hexValue string, filters Filters) ([]*Color, Metadata, error) {
	query := fmt.Sprintf(`
	SELECT count(*) OVER(), id, name, hex_value, version
	FROM colors
	WHERE (LOWER(hex_value) = LOWER($1) OR $1 = '')
	AND (to_tsvector('simple', name) @@ plainto_tsquery('simple', $2) OR $2 = '')
	ORDER BY %s %s, id DESC
	LIMIT $3 OFFSET $4`, filters.sortColumn(), filters.sortDirection())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)

	defer cancel()

	args := []any{name, hexValue, filters.limit(), filters.offset()}
	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}

	defer rows.Close()

	totalRecords := 0
	colors := []*Color{}

	for rows.Next() {
		var color Color
		err := rows.Scan(
			&totalRecords,
			&color.ID,
			&color.Name,
			&color.HexValue,
			&color.Version,
		)
		if err != nil {
			return nil, Metadata{}, err
		}

		colors = append(colors, &color)
	}

	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}

	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)
	return colors, metadata, nil
}
