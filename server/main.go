package main

import (
	"context"
	"flag"
	"github.com/julienschmidt/httprouter"
	"github.com/luno/gridlock/server/handlers"
	"github.com/luno/gridlock/server/ops"
	"github.com/luno/jettison/errors"
	"github.com/luno/jettison/j"
	jlog "github.com/luno/jettison/log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"time"
)

type state struct {
	Log      ops.TrafficStats
	Registry ops.NodeDB
}

func (s state) TrafficStats() ops.TrafficStats {
	return s.Log
}

func (s state) NodeRegistry() ops.NodeDB {
	return s.Registry
}

func main() {
	InitLogging()
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	mdb := ops.NewMemDB()
	s := state{
		Registry: mdb,
	}
	pool, err := ops.NewRedisPool(ctx)
	if err != nil {
		jlog.Error(ctx, errors.Wrap(err, "failed to connect to redis, falling back to memory db"))
		s.Log = ops.NewLoader(ctx, mdb)
	} else {
		s.Registry = ops.NewRedisNodeRegistry(pool)
		s.Log = ops.NewLoader(ctx, ops.NewRedisNodeDB(pool))
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
