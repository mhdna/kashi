package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (app *application) routes() http.Handler {
	router := httprouter.New()
	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.healthCheckHandler)
	router.HandlerFunc(http.MethodGet, "/v1/products", app.createProductHandler)
	router.HandlerFunc(http.MethodGet, "/v1/products/:id", app.showProductHandler)

	return router
}
