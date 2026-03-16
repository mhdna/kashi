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

	router.HandlerFunc(http.MethodGet, "/v1/colors", app.requirePermission("products:read", app.listColorsHandler))
	router.HandlerFunc(http.MethodGet, "/v1/colors/:id", app.requirePermission("products:read", app.showColorHandler))
	router.HandlerFunc(http.MethodPost, "/v1/colors", app.requirePermission("products:write", app.createColorHandler))
	router.HandlerFunc(http.MethodPatch, "/v1/colors/:id", app.requirePermission("products:write", app.updateColorHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/colors/:id", app.requirePermission("products:write", app.deleteColorHandler))

	router.HandlerFunc(http.MethodPost, "/v1/users", app.registerUserHandler)
	router.HandlerFunc(http.MethodPut, "/v1/users/activated", app.activateUserHandler)
	router.HandlerFunc(http.MethodPost, "/v1/users/authentication", app.createAuthenticationTokenHandler)

	// make sure enableCors is before rateLimit so that rateLimited stuff isn't blocked only because CORS isn't enabled
	return app.recoverPanic(app.enableCORS(app.rateLimit(app.authenticate(router))))
}
