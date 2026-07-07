package api

import (
	"database/sql"

	"ms_order/internal/core/jsonlog"
)

type repositories struct {
}

func NewRepositories(
	db *sql.DB,
	logger jsonlog.Logger,
) *repositories {
	return &repositories{}
}
