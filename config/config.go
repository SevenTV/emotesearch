package config

import (
	"time"

	config "github.com/gookit/config/v2"
	"github.com/gookit/config/v2/yaml"
)

type Config struct {
	LogLevel     string
	SyncInterval time.Duration
	Mongo        struct {
		URI        string
		Database   string
		Collection string
		Username   string
		Password   string
	}
	Meilisearch struct {
		Host  string
		Key   string
		Index string
	}
	Health struct {
		Enabled bool
		Port    string
	}
	Prometheus struct {
		Enabled bool
		Port    string
	}
}

func New() *Config {
	cfg := &Config{}
	loader := config.NewWithOptions("loader", config.ParseTime)
	loader.AddDriver(yaml.Driver)

	err := loader.LoadFiles("config.yaml")
	if err != nil {
		panic(err)
	}

	err = loader.Decode(cfg)
	if err != nil {
		panic(err)
	}

	return cfg
}
