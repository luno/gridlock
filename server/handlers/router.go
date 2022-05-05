package handlers

import (
	"context"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"strings"
)

func CreateRouter(ctx context.Context, d Deps) *httprouter.Router {
	r := httprouter.New()

	r.POST("/api/submit", SubmitMetricsHandler(d))
	r.GET("/api/traffic", GetTrafficHandler(d))
	r.GET("/api/graph", VizceralTrafficHandler(d))

	createWebApp(ctx, r)

	r.NotFound = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
		} else {
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		}
	})
	return r
}
