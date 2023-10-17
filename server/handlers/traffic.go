package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/luno/gridlock/api"
	"github.com/luno/gridlock/server/ops"
	"github.com/luno/jettison/log"
)

func GetTrafficHandler(d Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx := r.Context()
		t := d.TrafficStats().GetMetricLog()

		q := r.URL.Query()
		var ts time.Time
		if q.Has("ts") {
			unixTs, err := strconv.ParseInt(q.Get("ts"), 10, 64)
			if err != nil {
				http.Error(w, "Bad ts parameter", http.StatusBadRequest)
				return
			}
			ts = time.Unix(unixTs, 0)
		}

		resp := api.GetTrafficResponse{
			Traffic: ops.SummariseTraffic(t, ts),
		}
		respBytes, err := json.Marshal(resp)
		if err != nil {
			log.Error(ctx, err)
			http.Error(w, "Internal Error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(respBytes)
		if err != nil {
			log.Error(ctx, err)
		}
	}
}
