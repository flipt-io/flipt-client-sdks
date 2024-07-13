package flipt_engine_go

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
)

type Config struct {
	Enabled    bool
	Host       string
	Version    string
	Timeout    time.Duration
	Interval   time.Duration
	Namespaces []string
}

type Engine struct {
	cfg      *Config
	http     *HTTPParser
	snapshot *Snapshot
	logger   *zap.Logger
}

func NewEngine(
	cfg *Config,
	http *HTTPParser,
	snapshot *Snapshot,
	logger *zap.Logger,

) *Engine {
	return &Engine{
		cfg:      cfg,
		http:     http,
		snapshot: snapshot,
		logger:   logger,
	}
}

func (r *Engine) Run(ctx context.Context) {
	defer func() {
		if p := recover(); p != nil {
			r.logger.Error("panic occurred in flipt client-side engine", zap.Any("panic", p))
		}
	}()

	// initial snapshot
	err := r.replaceSnapshot(ctx)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		r.logger.Error("replace snapshot", zap.Error(err))
	}

	ticker := time.NewTicker(r.cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := r.replaceSnapshot(ctx)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				r.logger.Error("replace snapshot", zap.Error(err))
			}
		}
	}
}

func (r *Engine) replaceSnapshot(ctx context.Context) error {
	r.snapshot.cleanStore() // clear from memory before making a backup

	for _, namespace := range r.cfg.Namespaces {
		response, err := r.http.Do(ctx, namespace)
		if err != nil {
			return err
		}

		doc, err := r.http.Parse(response)
		if err != nil {
			return err
		}

		evalNamespace := r.snapshot.makeSnapshot(doc)
		r.snapshot.replaceStore(evalNamespace)
	}

	return nil
}
