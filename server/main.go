package main

import (
	"context"
	"flag"
	"github.com/adamhicks/gridlock/server/handlers"
	"github.com/adamhicks/gridlock/server/ops"
	"github.com/julienschmidt/httprouter"
	"github.com/luno/jettison/errors"
	"github.com/luno/jettison/j"
	"github.com/luno/jettison/log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"
)

type state struct {
	Log ops.TrafficStats
}

func (s state) TrafficStats() ops.TrafficStats {
	return s.Log
}

func main() {
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	var s state
	red, err := ops.NewRedis()
	if err != nil {
		log.Error(ctx, errors.Wrap(err, "failed to connect to redis, falling back to memory db"))
		s.Log = ops.NewLoader(ctx, ops.NewMemDB())
	} else {
		s.Log = ops.NewLoader(ctx, red)
	}

	runWebServer(ctx, handlers.CreateRouter(ctx, s), 80)
}

func runWebServer(ctx context.Context, router *httprouter.Router, port int) {
	srv := &http.Server{
		BaseContext: func(listener net.Listener) context.Context { return ctx },
		Handler:     router,
		Addr:        ":" + strconv.Itoa(port),
	}
	go shutdownOnCancel(ctx, srv)
	log.Info(ctx, "server listening", j.KV("port", port))
	err := srv.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		panic(err)
	}
	log.Info(ctx, "server terminated", j.KV("port", port))
}

func shutdownOnCancel(ctx context.Context, server *http.Server) {
	<-ctx.Done()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	log.Info(ctx, "shutting down http server")
	_ = server.Shutdown(ctx)
}
