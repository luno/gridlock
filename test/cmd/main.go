package main

import (
	"context"
	"github.com/luno/gridlock"
	"github.com/luno/gridlock/api"
	"github.com/luno/jettison/errors"
	"github.com/luno/jettison/log"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"
)

func deliverForever(ctx context.Context, c *gridlock.Client) {
	for {
		err := c.Deliver(ctx)
		if errors.IsAny(err, context.Canceled, context.DeadlineExceeded) {
			return
		} else if err != nil {
			log.Error(ctx, err)
		}
		time.Sleep(time.Second)
	}
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	c := gridlock.NewClient(gridlock.WithBaseURL("http://localhost/gridlock"))

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		deliverForever(ctx, c)
	}()

	registerNodes(c)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := simulateCalls(ctx, c, time.Now().UnixNano())
			if err != nil && !errors.Is(err, context.Canceled) {
				log.Error(ctx, err)
			}
		}()
	}
	wg.Wait()
}

var targets = map[string]map[string]int{
	"internet": {"fe": 1},
	"fe":       {"console": 3, "exchange": 1, "broker": 5},
	"broker":   {"kraken": 1, "bitstamp": 1, "broker_db": 1},
	"exchange": {"exchange_db": 1},
	"console":  {"ethereum": 1},
}

var success = map[gridlock.CallSuccess]int{
	gridlock.CallGood:    96,
	gridlock.CallWarning: 1,
	gridlock.CallBad:     3,
}

var regions = map[string]int{
	"eu-west-1":      10,
	"ap-southeast-3": 1,
}

func randomMethodPath(r *rand.Rand) []gridlock.Method {
	var ret []gridlock.Method
	source := "internet"
	for {
		target := ChooseWeighted(r, targets[source])
		if target == "" {
			break
		}
		transport := api.TransportGRPC
		if strings.HasSuffix(target, "_db") {
			transport = api.TransportSQL
		}
		m := gridlock.Method{
			Source:       source,
			SourceRegion: ChooseWeighted(r, regions),
			Target:       target,
			TargetRegion: ChooseWeighted(r, regions),
			Transport:    transport,
		}
		ret = append(ret, m)
		source = target
	}
	return ret
}

func registerNodes(c *gridlock.Client) {
	done := map[string]struct{}{"internet": {}}
	todo := []string{"internet"}
	for len(todo) > 0 {
		next := todo[0]
		todo = todo[1:]

		t := api.NodeService
		if strings.HasSuffix(next, "_db") {
			t = api.NodeDatabase
		}
		for r := range regions {
			c.RegisterNode(api.NodeInfo{
				Region:      r,
				Name:        next,
				DisplayName: next,
				Type:        t,
			})
		}

		for tgt := range targets[next] {
			if _, d := done[tgt]; d {
				continue
			}
			todo = append(todo, tgt)
			done[tgt] = struct{}{}
		}
	}
}

func simulateCalls(ctx context.Context, client *gridlock.Client, seed int64) error {
	ti := time.NewTicker(time.Millisecond)
	defer ti.Stop()

	r := rand.New(rand.NewSource(seed))

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ti.C:
			for _, m := range randomMethodPath(r) {
				client.Record(m, ChooseWeighted(r, success))
			}
		}
	}
}
