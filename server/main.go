package main

import (
	"context"
	"flag"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/luno/gridlock/server/handlers"
	"github.com/luno/gridlock/server/ops"
	"github.com/luno/gridlock/server/ops/config"
	"github.com/luno/jettison/errors"
	"github.com/luno/jettison/j"
	jlog "github.com/luno/jettison/log"
)

type state struct {
	Log ops.TrafficStats
}

func (s state) TrafficStats() ops.TrafficStats {
	return s.Log
}

func main() {
	InitLogging()
	flag.Parse()
	config.MustLoadConfig()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	var s state
	pool, err := ops.NewRedisPool(ctx)
	if err != nil {
		jlog.Error(ctx, errors.Wrap(err, "failed to connect to redis, falling back to memory db"))
		mdb := ops.NewMemDB()
		s.Log = ops.NewLoader(ctx, mdb, mdb)
	} else {
		s.Log = ops.NewLoader(ctx,
			ops.NewRedisTrafficDB(pool),
			ops.NewRedisNodeDB(pool),
		)
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		runWebServer(ctx, handlers.CreateRouter(ctx, s), 80)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		runWebServer(ctx, handlers.CreateDebugRouter(), 8080)
	}()

	wg.Wait()
}

func runWebServer(ctx context.Context, router *httprouter.Router, port int) {
	srv := &http.Server{
		BaseContext: func(listener net.Listener) context.Context { return ctx },
		Handler:     router,
		Addr:        ":" + strconv.Itoa(port),
	}
	go shutdownOnCancel(ctx, srv)
	jlog.Info(ctx, "server listening", j.KV("port", port))
	err := srv.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		panic(err)
	}
	jlog.Info(ctx, "server terminated", j.KV("port", port))
}

func shutdownOnCancel(ctx context.Context, server *http.Server) {
	<-ctx.Done()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	jlog.Info(ctx, "shutting down http server")
	_ = server.Shutdown(ctx)
}
