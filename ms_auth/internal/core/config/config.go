package config

import (
	"log"
	"time"

	"github.com/joeshaw/envdecode"
)

type Config struct {
	Server struct {
		Port    int
		Timeout time.Duration
	}
	Env string
	DB  struct {
		DSN          string
		MaxOpenConns int
		MaxIdleConns int
		MaxIdleTime  string
	}
	Limiter struct {
		RPS     float64
		Burst   int
		Enabled bool
	}
	CORS struct {
		TrustedOrigins []string
	}
	Security struct {
		PrivateKeyPath string
		PublicKeyPath  string
	}
	Cache struct {
		Addr     string
		Password string
		Db       int
	}
}

type Conf struct {
	Server      ConfServer
	DB          ConfDB
	RateLimiter ConfRL
	Security    ConfSecurity
	Cache       ConfCache
}

type ConfServer struct {
	Port    int           `env:"SERVER_PORT,required"`
	Debug   bool          `env:"SERVER_DEBUG,required"`
	Timeout time.Duration `env:"SERVER_TIMEOUT,required"`
}

type ConfDB struct {
	DSN          string `env:"DB_DSN,required"`
	MaxOpenConns int    `env:"DB_MAX_OPEN_CONNS,required"`
	MaxIdleConns int    `env:"DB_MAX_IDLE_CONNS,required"`
	MaxIdleTime  string `env:"DB_MAX_IDLE_TIME,required"`
}

type ConfRL struct {
	RPS     float64 `env:"LIMITER_RPS,required"`
	Burst   int     `env:"LIMITER_BURST,required"`
	Enabled bool    `env:"LIMITER_ENABLED,required"`
}

type ConfSecurity struct {
	PrivateKeyPath string `env:"PRIVATE_KEY_PATH,required"`
	PublicKeyPath  string `env:"PUBLIC_KEY_PATH,required"`
}

type ConfCache struct {
	Addr     string `env:"CACHE_ADDR,required"`
	Password string `env:"CACHE_PASSWORD,required"`
	Db       int    `env:"CACHE_DB,required"`
}

func New() *Conf {
	var c Conf
	if err := envdecode.StrictDecode(&c); err != nil {
		log.Fatalf("Failed to decode: %s", err)
	}

	return &c
}

func NewDB() *ConfDB {
	var c ConfDB
	if err := envdecode.StrictDecode(&c); err != nil {
		log.Fatalf("Failed to decode: %s", err)
	}

	return &c
}
