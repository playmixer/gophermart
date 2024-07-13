package logger

import (
	"fmt"

	"go.uber.org/zap"
)

type option func(*zap.Config)

func OutputPath(path string) option {
	return func(c *zap.Config) {
		// _ = os.Mkdir(path, os.ModePerm)
		c.OutputPaths = append(c.OutputPaths, path)
	}
}

func New(level string, options ...option) (*zap.Logger, error) {
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return nil, fmt.Errorf("logger failed parse level %w", err)
	}

	cfg := zap.NewProductionConfig()
	cfg.Level = lvl

	for _, opt := range options {
		opt(&cfg)
	}

	zl, err := cfg.Build()
	if err != nil {
		return nil, fmt.Errorf("failed build logger %w", err)
	}

	return zl, nil
}
