package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (app *application) routes() http.Handler {
	router := httprouter.New()
	router.NotFound = http.HandlerFunc(app.notFoundResponse)
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)

	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.healthCheckHandler)

	router.HandlerFunc(http.MethodGet, "/v1/products", app.requirePermission("products:read", app.listProductsHandler))
	router.HandlerFunc(http.MethodGet, "/v1/products/:id", app.requirePermission("products:read", app.showProductHandler))
	router.HandlerFunc(http.MethodPost, "/v1/products", app.requirePermission("products:write", app.createProductHandler))
	router.HandlerFunc(http.MethodPatch, "/v1/products/:id", app.requirePermission("products:write", app.updateProductHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/products/:id", app.requirePermission("products:write", app.deleteProductHandler))

	router.HandlerFunc(http.MethodPost, "/v1/users", app.registerUserHandler)
	router.HandlerFunc(http.MethodPut, "/v1/users/activated", app.activateUserHandler)
	router.HandlerFunc(http.MethodPost, "/v1/users/authentication", app.createAuthenticationTokenHandler)

	return app.recoverPanic(app.rateLimit(app.authenticate(router)))
}
