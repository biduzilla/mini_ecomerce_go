package api

import (
	"log"

	"github.com/joeshaw/envdecode"
)

type Config struct {
	Server struct {
		Port int `env:"GATEWAY_PORT,required"`
	}
	Services struct {
		AuthURL    string `env:"MS_AUTH_URL,required"`
		ProductURL string `env:"MS_PRODUCT_URL,required"`
		StockURL   string `env:"MS_STOCK_URL,required"`
		OrderURL   string `env:"MS_ORDER_URL,required"`
	}
}

func NewConfig() *Config {
	var c Config
	if err := envdecode.StrictDecode(&c); err != nil {
		log.Fatalf("Failed to decode config: %s", err)
	}
	return &c
}
