package main

import (
	"context"
	"flag"
	"github.com/adamhicks/gridlock/server/handlers"
	"github.com/adamhicks/gridlock/server/ops"
	"github.com/gomodule/redigo/redis"
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

	rawConn, err := redis.DialURLContext(ctx, "redis://127.0.0.1:6379")
	if err != nil {
		panic(err)
	}
	s := state{Log: ops.NewLoader(ctx, ops.RedisDB{RedisConn: rawConn})}

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
