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
	var input struct {
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
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	product := &data.Product{
		Code:          input.Code,
		Name:          input.Name,
		Description:   input.Description,
		KindId:        input.KindId,
		CategoryId:    input.CategoryId,
		SubCategoryId: input.SubCategoryId,
		UnitId:        input.UnitId,
		TypeId:        input.TypeId,
		Year:          input.Year,
		SeasonId:      input.SeasonId,
		BrandId:       input.BrandId,
		OriginId:      input.OriginId,
		Price:         input.Price,
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
		Code          *string  `json:"code"`
		Name          *string  `json:"name"`
		Description   *string  `json:"description"`
		KindId        *int64   `json:"kind_id"`
		CategoryId    *int64   `json:"category_id"`
		SubCategoryId *int64   `json:"sub_category_id"`
		UnitId        *int64   `json:"unit_id"`
		TypeId        *int64   `json:"type_id"`
		Year          *int32   `json:"year,omitempty"`
		SeasonId      *int64   `json:"season_id"`
		BrandId       *int64   `json:"brand_id"`
		OriginId      *int64   `json:"origin_id"`
		Price         *float64 `json:"price"`
		IsActive      *bool    `json:"is_active"`
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
	if input.KindId != nil {
		product.KindId = *input.KindId
	}
	if input.CategoryId != nil {
		product.CategoryId = *input.CategoryId
	}
	if input.SubCategoryId != nil {
		product.SubCategoryId = *input.SubCategoryId
	}
	if input.UnitId != nil {
		product.UnitId = *input.UnitId
	}
	if input.TypeId != nil {
		product.TypeId = *input.TypeId
	}
	if input.Year != nil {
		product.Year = *input.Year
	}
	if input.SeasonId != nil {
		product.SeasonId = *input.SeasonId
	}
	if input.BrandId != nil {
		product.BrandId = *input.BrandId
	}
	if input.OriginId != nil {
		product.OriginId = *input.OriginId
	}
	if input.Price != nil {
		product.Price = *input.Price
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
		Code          string
		Name          string
		Description   string
		KindId        int64
		CategoryId    int64
		SubCategoryId int64
		UnitId        int64
		TypeId        int64
		Year          int32
		SeasonId      int64
		BrandId       int64
		OriginId      int64
		Price         float64
		IsActive      bool
		data.Filters
	}

	v := validator.New()
	qs := r.URL.Query()

	input.Code = app.readString(qs, "code", "")
	input.Name = app.readString(qs, "name", "")

	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 20, v)

	input.Filters.Sort = app.readString(qs, "sort", "id")
	input.Filters.SortSafeList = []string{"id", "name", "year", "-id", "-name", "-year"}

	if data.ValidateFitlers(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	products, metadata, err := app.models.Products.GetAll(input.Code, input.Name, input.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	err = app.writeJSON(w, http.StatusOK, envelop{"products": products, "metadata": metadata}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
