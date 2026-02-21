package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/mhdna/kashi/internal/data"
	"github.com/mhdna/kashi/internal/validator"
)

func (app *application) showProductHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.getProductId(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	product := data.Product{
		ID:        id,
		CreatedAt: time.Now(),
		Name:      "Test",
		Price:     102,
	}

	err = app.writeJSON(w, http.StatusOK, envelop{"product": product}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) createProductHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Code    string       `json:"code"`
		Name    string       `json:"name"`
		Runtime data.Runtime `json:"runtime"`
		Year    int32        `json:"year,omitempty"`
		// Tags    []string     `json:"tags"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// copy values from input into the product struct
	product := &data.Product{
		Name:    input.Name,
		Code:    input.Code,
		Runtime: input.Runtime,
		Year:    input.Year,
	}

	v := validator.New()

	// passing around the validator is easier than initializing it in functions
	if data.ValidateProduct(v, product); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	fmt.Fprintf(w, "%+v\n", input)
}
