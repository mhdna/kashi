package main

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/mhdna/kashi/internal/data"
	"github.com/mhdna/kashi/internal/validator"
)

func (app *application) showProductHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	product, err := app.models.Products.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)

		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelop{"product": product}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) createProductHandler(w http.ResponseWriter, r *http.Request) {
	// TODO remove category and other things that are in other tables
	var input struct {
		Code        string  `json:"code"`
		Name        string  `json:"name"`
		Description string  `json:"description"`
		Kind        string  `json:"kind"`
		Type        string  `json:"type"`
		Year        int32   `json:"year,omitempty"`
		Unit        string  `json:"unit"`
		Season      string  `json:"season"`
		Price       float64 `json:"price"`
		Cost        float64 `json:"cost"`
		Category    string  `json:"category"`
		IsActive    bool    `json:"is_active"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	product := &data.Product{
		Code:        input.Code,
		Name:        input.Name,
		Description: input.Description,
		Kind:        input.Kind,
		Type:        input.Type,
		Year:        input.Year,
		Unit:        input.Unit,
		Season:      input.Season,
		Price:       input.Price,
		Cost:        input.Cost,
		Category:    input.Category,
		IsActive:    input.IsActive,
	}

	v := validator.New()

	// passing around the validator is easier than initializing it in functions
	if data.ValidateProduct(v, product); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.models.Products.Insert(product)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/products/%d", product.ID))

	err = app.writeJSON(w, http.StatusCreated, envelop{"product": product}, headers)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) updateProductHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	product, err := app.models.Products.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	var input struct {
		Code        *string  `json:"code"`
		Name        *string  `json:"name"`
		Description *string  `json:"description"`
		Kind        *string  `json:"kind"`
		Type        *string  `json:"type"`
		Year        *int32   `json:"year"`
		Unit        *string  `json:"unit"`
		Season      *string  `json:"season"`
		Price       *float64 `json:"price"`
		Cost        *float64 `json:"cost"`
		Category    *string  `json:"category"`
		IsActive    *bool    `json:"is_active"`
	}

	err = app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if input.Code != nil {
		product.Code = *input.Code
	}
	if input.Name != nil {
		product.Name = *input.Name
	}
	if input.Description != nil {
		product.Description = *input.Description
	}
	if input.Kind != nil {
		product.Kind = *input.Kind
	}
	if input.Type != nil {
		product.Type = *input.Type
	}
	if input.Year != nil {
		product.Year = *input.Year
	}
	if input.Unit != nil {
		product.Unit = *input.Unit
	}
	if input.Season != nil {
		product.Season = *input.Season
	}
	if input.Price != nil {
		product.Price = *input.Price
	}
	if input.Cost != nil {
		product.Cost = *input.Cost
	}
	if input.Category != nil {
		product.Category = *input.Category
	}
	if input.IsActive != nil {
		product.IsActive = *input.IsActive
	}

	v := validator.New()

	if data.ValidateProduct(v, product); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.models.Products.Update(product)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}

	err = app.writeJSON(w, http.StatusOK, envelop{"product": product}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) deleteProductHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	err = app.models.Products.Delete(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)

		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelop{"message": "product successfully deleted"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) listProductsHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Code        string
		Name        string
		Description string
		Kind        string
		Type        string
		Year        string
		Unit        string
		Season      string
		Price       string
		Cost        string
		Category    string
		IsActive    string
		data.Filters
	}

	v := validator.New()
	qs := r.URL.Query()

	input.Code = app.readString(qs, "code", "")
	input.Name = app.readString(qs, "name", "")

	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 20, v)
	input.Filters.ExactSearch = app.readBool(qs, "exact", false, v)

	input.Filters.Sort = app.readString(qs, "sort", "id")
	input.Filters.SortSafeList = []string{"id", "name", "year", "-id", "-name", "-year"}

	if data.ValidateFitlers(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	products, err := app.models.Products.GetAll(input.Code, input.Name, input.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	err = app.writeJSON(w, http.StatusOK, envelop{"products": products}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
