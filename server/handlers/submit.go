package handlers

import (
	"encoding/json"
	"github.com/adamhicks/gridlock/api"
	"github.com/julienschmidt/httprouter"
	"io"
	"net/http"
)

func SubmitMetricsHandler(d Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		if r.Header.Get("Content-Type") != "application/json" {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		var req api.SubmitMetrics
		b, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		err = json.Unmarshal(b, &req)
		if err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		ctx := r.Context()
		err = d.TrafficStats().Record(ctx, req.Metrics...)
		if err != nil {
			http.Error(w, "Internal Error", http.StatusInternalServerError)
		}
	}
}
