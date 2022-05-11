package main

import (
	"context"
	"github.com/adamhicks/gridlock"
	"github.com/adamhicks/gridlock/api"
	"github.com/luno/jettison/errors"
	"github.com/luno/jettison/log"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"time"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	c := gridlock.NewClient(gridlock.WithBaseURL("http://localhost"))
	go func() {
		err := c.Deliver(ctx)
		if err != nil {
			log.Error(ctx, err)
		}
	}()

	var wg sync.WaitGroup
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

var sources = map[string]int{
	"fe":       50,
	"console":  25,
	"exchange": 15,
}

var targets = map[string]int{
	"console":  25,
	"exchange": 10,
	"broker":   1,
	"ethereum": 1,
}

var success = map[gridlock.CallSuccess]int{
	gridlock.CallGood:    96,
	gridlock.CallWarning: 1,
	gridlock.CallBad:     3,
}

func randomMethod(r *rand.Rand) gridlock.Method {
	return gridlock.Method{
		Source:       ChooseWeighted(r, sources),
		SourceRegion: "eu-west-1",
		Target:       ChooseWeighted(r, targets),
		TargetRegion: "eu-west-1",
		Transport:    api.TransportGRPC,
	}
}

func simulateCalls(ctx context.Context, client *gridlock.Client, seed int64) error {
	ti := time.NewTicker(time.Microsecond)
	defer ti.Stop()

	r := rand.New(rand.NewSource(seed))

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ti.C:
			client.Record(randomMethod(r), ChooseWeighted(r, success))
		}
	}
}
