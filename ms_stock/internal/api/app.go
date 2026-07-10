package api

import (
	"database/sql"
	"expvar"
	"runtime"
	"sync"
	"time"

	"ms_stock/internal/core/config"
	"ms_stock/internal/core/database"
	"ms_stock/internal/core/jsonlog"
	"ms_stock/internal/core/messaging"

	"github.com/IBM/sarama"
)

type application struct {
	config config.Config
	Logger jsonlog.Logger
	wg     sync.WaitGroup
	db     *sql.DB

	kafkaProducer sarama.SyncProducer
	kafkaConsumer sarama.ConsumerGroup
}

const version = "1.0.0"

func NewApp(cfg config.Config, logger jsonlog.Logger) (*application, error) {
	db, err := database.OpenDB(cfg)
	if err != nil {
		logger.PrintError(err, nil)
		return nil, err
	}

	logger.PrintInfo("database connection pool established", nil)

	producers, consumers, err := messaging.InitKafka(cfg.Kafka.Brokers, cfg.Kafka.GroupID)
	if err != nil {
		logger.PrintError(err, nil)
		return nil, err
	}

	logger.PrintInfo("kafka producer and consumer established", nil)

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
		config:        cfg,
		Logger:        logger,
		db:            db,
		kafkaProducer: producers,
		kafkaConsumer: consumers,
	}, nil
}
