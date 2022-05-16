package handlers

import (
	"context"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"strings"
)

type Router interface {
	GET(path string, handle httprouter.Handle)
	POST(path string, handle httprouter.Handle)
}

type subRouter struct {
	r    Router
	base string
}

func SubRouter(r Router, basePath string) Router {
	return subRouter{r: r, base: basePath}
}

func (r subRouter) GET(path string, handle httprouter.Handle) {
	r.r.GET(r.base+path, handle)
}

func (r subRouter) POST(path string, handle httprouter.Handle) {
	r.r.POST(r.base+path, handle)
}

func CreateRouter(ctx context.Context, d Deps) *httprouter.Router {
	r := httprouter.New()
	grid := SubRouter(r, "/gridlock")

	grid.POST("/api/submit", SubmitMetricsHandler(d))
	grid.GET("/api/traffic", GetTrafficHandler(d))
	grid.GET("/api/graph", VizceralTrafficHandler(d))

	createWebApp(ctx, grid)

	r.NotFound = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/gridlock/api/") {
			http.NotFound(w, r)
		} else if strings.HasPrefix(r.URL.Path, "/gridlock/") {
			serveIndex(w, r, nil)
		} else {
			http.Redirect(w, r, "/gridlock", http.StatusTemporaryRedirect)
		}
	})
	return r
}
