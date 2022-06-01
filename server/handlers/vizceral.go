package handlers

import (
	_ "embed"
	"encoding/json"
	"github.com/julienschmidt/httprouter"
	"github.com/luno/gridlock/server/ops"
	"github.com/luno/jettison/log"
	"net/http"
)

func VizceralTrafficHandler(d Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx := r.Context()
		t := d.TrafficStats().GetTraffic()
		g := ops.CompileVizceralGraph(t)
		b, err := json.Marshal(g)
		if err != nil {
			log.Error(ctx, err)
			http.Error(w, "Internal Error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(b)
		if err != nil {
			log.Error(ctx, err)
		}
	}
}
