// Package config загружает настройки приложения из YAML и окружения
package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v3"
)

type Config struct {
	HTTP struct {
		ListenAddr string `yaml:"listen_addr" split_words:"true"`
	} `yaml:"http"`
	Postgres struct {
		Addr     string `yaml:"addr"`
		Database string `yaml:"database"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
	} `yaml:"postgres"`
	Redis  Redis `yaml:"redis"`
	Limits struct {
		IP       uint32 `yaml:"ip"`
		Login    uint32 `yaml:"login"`
		Password uint32 `yaml:"password"`
	} `yaml:"limits"`
	HMACSecret string `yaml:"hmac_secret" split_words:"true"`
}

type Redis struct {
	Addr           string        `yaml:"addr"`
	Password       string        `yaml:"password"`
	MaxIdle        int           `yaml:"max_idle" split_words:"true"`
	IdleTimeout    time.Duration `yaml:"idle_timeout" split_words:"true"`
	ConnectTimeout time.Duration `yaml:"connect_timeout" split_words:"true"`
	ReadTimeout    time.Duration `yaml:"read_timeout" split_words:"true"`
	WriteTimeout   time.Duration `yaml:"write_timeout" split_words:"true"`
}

func Load(path string) (Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return Config{}, err
	}
	defer file.Close()
	var cfg Config
	decoder := yaml.NewDecoder(file)
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("invalid YAML configuration in %s", path)
	}
	if err := envconfig.Process("ABF", &cfg); err != nil {
		return Config{}, err
	}
	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (cfg Config) validate() error {
	switch {
	case cfg.HMACSecret == "":
		return errors.New("hmac_secret is required")
	case cfg.HTTP.ListenAddr == "":
		return errors.New("http.listen_addr is required")
	case cfg.Postgres.Addr == "" || cfg.Postgres.Database == "" || cfg.Postgres.User == "":
		return errors.New("postgres.addr, postgres.database and postgres.user are required")
	case cfg.Redis.Addr == "":
		return errors.New("redis.addr is required")
	case cfg.Redis.MaxIdle < 0:
		return errors.New("redis.max_idle must be non-negative")
	case cfg.Redis.IdleTimeout < 0:
		return errors.New("redis.idle_timeout must be non-negative")
	case cfg.Redis.ConnectTimeout <= 0:
		return errors.New("redis.connect_timeout must be positive")
	case cfg.Redis.ReadTimeout <= 0:
		return errors.New("redis.read_timeout must be positive")
	case cfg.Redis.WriteTimeout <= 0:
		return errors.New("redis.write_timeout must be positive")
	}
	return nil
}
