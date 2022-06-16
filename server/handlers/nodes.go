package handlers

import (
	"encoding/json"
	"github.com/julienschmidt/httprouter"
	"github.com/luno/gridlock/api"
	"github.com/luno/jettison/errors"
	"github.com/luno/jettison/log"
	"net/http"
)

func GetNodesHandler(d Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx := r.Context()
		nl := d.TrafficStats().GetNodes()
		resp := api.GetNodesResponse{NodeInfo: nl}
		respBytes, err := json.Marshal(resp)
		if err != nil {
			log.Error(ctx, errors.Wrap(err, "json marshal"))
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
