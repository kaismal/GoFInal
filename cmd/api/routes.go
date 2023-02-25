package main

import (
	"github.com/julienschmidt/httprouter"
	"net/http"
)

func (app *application) routes() http.Handler {
	router := httprouter.New()
	router.NotFound = http.HandlerFunc(app.notFoundResponse)
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)
	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.healthcheckHandler)
	router.HandlerFunc(http.MethodGet, "/v1/replays", app.requirePermission("replays:read", app.listReplaysHandler))
	router.HandlerFunc(http.MethodPost, "/v1/replays", app.requirePermission("replays:write", app.createReplayHandler))
	router.HandlerFunc(http.MethodGet, "/v1/replays/:id", app.requirePermission("replays:read", app.showReplayHandler))
	router.HandlerFunc(http.MethodPatch, "/v1/replays/:id", app.requirePermission("replays:write", app.updateReplayHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/replays/:id", app.requirePermission("replays:write", app.deleteReplayHandler))
	router.HandlerFunc(http.MethodPost, "/v1/users", app.registerUserHandler)
	router.HandlerFunc(http.MethodPut, "/v1/users/activated", app.activateUserHandler)
	router.HandlerFunc(http.MethodPost, "/v1/tokens/authentication", app.createAuthenticationTokenHandler)
	return app.recoverPanic(app.rateLimit(app.authenticate(router)))
}
