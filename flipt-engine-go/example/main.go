package main

import (
	"context"
	"time"

	"go.uber.org/zap"

	"go.flipt.io/flipt-engine-go"
)

func main() {
	cfg := &flipt_engine_go.Config{
		Enabled:    true,
		Host:       "localhost:8900",
		Version:    "1.39.0",
		Timeout:    10 * time.Second,
		Interval:   30 * time.Minute,
		Namespaces: []string{"default", "mobile"},
	}

	logger := zap.NewNop()
	http := flipt_engine_go.NewHTTPParser(cfg.Host, cfg.Version, cfg.Timeout)
	snapshot := flipt_engine_go.NewSnapshot()
	flipt_engine_go.SetEvaluator(snapshot, logger)

	ctx := context.Background()

	engine := flipt_engine_go.NewEngine(cfg, http, snapshot, logger.With(zap.String("worker", "flipt_client_side")))
	engine.Run(ctx)
}
