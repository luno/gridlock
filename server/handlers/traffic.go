package handlers

import (
	"encoding/json"
	"github.com/adamhicks/gridlock/api"
	"github.com/adamhicks/gridlock/server/ops"
	"github.com/julienschmidt/httprouter"
	"github.com/luno/jettison/log"
	"net/http"
)

func GetTrafficHandler(d Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx := r.Context()
		t := d.TrafficStats().GetTraffic()
		resp := api.GetTraffic{
			Traffic: ops.SummariseTraffic(t),
		}
		b, err := json.Marshal(resp)
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
