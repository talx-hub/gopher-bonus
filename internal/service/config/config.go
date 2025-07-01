package config

import (
	"context"
	"flag"
	"log/slog"

	"github.com/caarlos0/env/v6"

	"github.com/talx-hub/gopher-bonus/internal/model"
)

type Config struct {
	RunAddr       string `env:"RUN_ADDRESS"   envDefault:"localhost:8080"`
	DatabaseURI   string `env:"DATABASE_URI"   envDefault:""`
	AccrualAddr   string `env:"ACCRUAL_SYSTEM_ADDRESS"   envDefault:"localhost:8081"`
	SecretKey     string `env:"SECRET_KEY"     envDefault:""`
	LogLevel      string `env:"LOG_LEVEL"      envDefault:"info"`
	UsePagination bool   `env:"USE_PAGINATION" envDefault:"false"`
}

type Builder struct {
	cfg *Config
	log *slog.Logger
}

func NewBuilder(log *slog.Logger) *Builder {
	return &Builder{
		cfg: &Config{
			RunAddr:       "",
			DatabaseURI:   "",
			AccrualAddr:   "",
			SecretKey:     "",
			LogLevel:      "",
			UsePagination: false,
		},
		log: log,
	}
}

func (b *Builder) FromEnv() *Builder {
	if err := env.Parse(b.cfg); err != nil {
		b.log.LogAttrs(context.Background(),
			slog.LevelError, "Failed to parse config", slog.Any(model.KeyLoggerError, err))
	}
	return b
}

func (b *Builder) FromFlags() *Builder {
	flag.StringVar(&b.cfg.RunAddr, "a", b.cfg.RunAddr, "Run address")
	flag.StringVar(&b.cfg.DatabaseURI, "d", b.cfg.DatabaseURI, "Database URI")
	flag.StringVar(&b.cfg.AccrualAddr, "r", b.cfg.AccrualAddr, "Accrual address")
	flag.StringVar(&b.cfg.SecretKey, "k", b.cfg.SecretKey, "Secret key")
	flag.StringVar(&b.cfg.LogLevel, "l", b.cfg.LogLevel, "Log level")
	flag.BoolVar(&b.cfg.UsePagination, "p", b.cfg.UsePagination, "Use pagination")

	flag.Parse()
	return b
}

func (b *Builder) GetConfig() *Config {
	return b.cfg
}
