package main

import (
	"errors"
	"net/http"

	"github.com/mhdna/kashi/internal/data"
	"github.com/mhdna/kashi/internal/validator"
)

type Color struct {
	Name     string `json:"name"`
	HexValue string `json:"hex_value"`
}

func (app *application) showColorHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	color, err := app.models.Colors.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	err = app.writeJSON(w, http.StatusOK, envelop{"color": color}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) createColorHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name     string `json:"name"`
		HexValue string `json:"hex_value"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	color := &data.Color{
		Name:     input.Name,
		HexValue: input.HexValue,
	}

	v := validator.New()
	if data.ValidateColor(v, color); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.models.Colors.Insert(color)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateName):
			v.AddError("name", "a color with a similar name already exists")
			app.failedValidationResponse(w, r, v.Errors)
		case errors.Is(err, data.ErrDuplicateHexValue):
			v.AddError("hex_value", "a color with a similar hex_value already exists")
			app.failedValidationResponse(w, r, v.Errors)
		default:
			app.serverErrorResponse(w, r, err)
			return
		}
		return
	}
	err = app.writeJSON(w, http.StatusCreated, envelop{"color": color}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) updateColorHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	color, err := app.models.Colors.Get(id)
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
		Name     string `json:"name"`
		HexValue string `json:"hex_value"`
	}

	err = app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	if data.ValidateColor(v, color); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	color.Name = input.Name
	color.HexValue = input.HexValue
	err = app.models.Colors.Update(color)
	if err != nil {
		app.editConflicResponse(w, r)
		return
	}
	err = app.writeJSON(w, http.StatusOK, envelop{"color": color}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) deleteColorHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}
	err = app.models.Colors.Delete(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	err = app.writeJSON(w, http.StatusOK, envelop{"message": "color successfully deleted"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) listColorsHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name     string
		HexValue string
		data.Filters
	}

	qs := r.URL.Query()

	v := validator.New()

	input.Name = app.readString(qs, "name", "")
	input.HexValue = app.readString(qs, "hex_value", "")

	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 20, v)

	input.Filters.Sort = app.readString(qs, "sort", "id")
	input.Filters.SortSafeList = []string{"id", "name", "hex_value", "-id", "-name", "-hex_value"}

	if data.ValidateFitlers(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	colors, metadata, err := app.models.Colors.GetAll(input.Name, input.HexValue, input.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	err = app.writeJSON(w, http.StatusOK, envelop{"colors": colors, "metadata": metadata}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
