package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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
	p := r.base + path
	r.r.GET(p, wrap(p, handle))
}

func (r subRouter) POST(path string, handle httprouter.Handle) {
	p := r.base + path
	r.r.POST(p, wrap(p, handle))
}

func wrap(path string, handle httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		t0 := time.Now()
		handle(w, r, p)
		httpHandle.WithLabelValues(path).Observe(time.Since(t0).Seconds())
	}
}

func CreateRouter(ctx context.Context, d Deps) *httprouter.Router {
	r := httprouter.New()
	grid := SubRouter(r, "/gridlock")

	grid.POST("/api/submit", SubmitMetricsHandler(d))
	grid.GET("/api/traffic", GetTrafficHandler(d))
	grid.GET("/api/nodes", GetNodesHandler(d))
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

func CreateDebugRouter() *httprouter.Router {
	r := httprouter.New()
	r.Handler(http.MethodGet, "/debug/metrics", promhttp.Handler())
	r.HandlerFunc(http.MethodGet, "/debug/ready", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("OK"))
	})

	return r
}
