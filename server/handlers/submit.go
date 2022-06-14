package handlers

import (
	"encoding/json"
	"github.com/julienschmidt/httprouter"
	"github.com/luno/gridlock/api"
	"github.com/luno/jettison/errors"
	"github.com/luno/jettison/log"
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
		err = d.NodeRegistry().RegisterNodes(ctx, req.NodeInfo...)
		if err != nil {
			log.Error(ctx, errors.Wrap(err, "submit metrics"))
			http.Error(w, "Internal Error", http.StatusInternalServerError)
		}

		err = d.TrafficStats().Record(ctx, req.Metrics...)
		if err != nil {
			log.Error(ctx, errors.Wrap(err, "submit metrics"))
			http.Error(w, "Internal Error", http.StatusInternalServerError)
		}
	}
}
