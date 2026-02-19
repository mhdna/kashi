package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/mhdna/kashi/internal/data"
)

func (app *application) createProductHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Created Product")
}

func (app *application) showProductHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.getProductId(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	product := data.Product{
		ID:        id,
		CreatedAt: time.Now(),
		Name:      "Test",
		Price:     102,
	}

	err = app.writeJSON(w, http.StatusOK, product, nil)
	if err != nil {
		app.logger.Error(err.Error())
		http.Error(w, "The server encountered a problem and could not proccess your request", http.StatusInternalServerError)
	}
}
