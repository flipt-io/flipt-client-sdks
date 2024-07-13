package main

import (
	"context"

	"go.uber.org/zap"

	"go.flipt.io/flipt-engine-go"
)

func main() {
	cfg := &flipt_engine_go.Config{
		Enabled:    false,
		Host:       "",
		Version:    "",
		Timeout:    0,
		Interval:   0,
		Namespaces: nil,
	}

	logger := zap.NewNop()
	http := flipt_engine_go.NewHTTPParser(cfg.Host, cfg.Version, cfg.Timeout)
	snapshot := flipt_engine_go.NewSnapshot()
	flipt_engine_go.SetEvaluator(snapshot, logger)

	ctx := context.Background()

	engine := flipt_engine_go.NewEngine(cfg, http, snapshot, logger.With(zap.String("worker", "flipt_client_side")))
	engine.Run(ctx)
}
