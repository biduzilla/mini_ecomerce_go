package api

import (
	"database/sql"
	"expvar"
	"ms_product/internal/core/config"
	"ms_product/internal/core/database"
	"ms_product/internal/core/jsonlog"
	"os"
	"runtime"
	"sync"
	"time"
)

type application struct {
	config config.Config
	Logger jsonlog.Logger
	wg     sync.WaitGroup
	db     *sql.DB
}

const version = "1.0.0"

func NewApp(cfg config.Config) *application {
	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)

	db, err := database.OpenDB(cfg)
	if err != nil {
		logger.PrintError(err, nil)
		return nil
	}

	logger.PrintInfo("database connection pool established", nil)

	expvar.NewString("version").Set(version)

	expvar.Publish("goroutines", expvar.Func(func() any {
		return runtime.NumGoroutine()
	}))

	expvar.Publish("database", expvar.Func(func() any {
		return db.Stats()
	}))

	expvar.Publish("timestamp", expvar.Func(func() any {
		return time.Now().Unix()
	}))

	return &application{
		config: cfg,
		Logger: logger,
		db:     db,
	}
}
