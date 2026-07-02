package api

import (
	"database/sql"
	"ms_auth/internal/core/jsonlog"
	"ms_auth/internal/features/user"
)

type repositories struct {
	userRepository user.UserRepository
}

func NewRepositories(
	db *sql.DB,
	logger jsonlog.Logger,
) *repositories {
	return &repositories{
		userRepository: *user.NewRepository(db, logger),
	}
}
