package shulker

import (
	"context"

	"github.com/maddiesch/go-raptor"
	"github.com/pkg/errors"
	"github.com/samber/do"
	"golang.org/x/exp/slog"
)

type DatabaseService struct {
	db  *raptor.Conn
	log *slog.Logger
}

func NewDatabaseService(i *do.Injector) (*DatabaseService, error) {
	config, err := do.Invoke[Config](i)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get config")
	}

	log, err := do.Invoke[*slog.Logger](i)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get logger instance")
	}

	db, err := raptor.New(config.DatabasePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create database connection")
	}

	return &DatabaseService{
		db:  db,
		log: log.With(slog.String("subsystem", "database")),
	}, nil
}

func (s *DatabaseService) HealthCheck() error {
	return s.db.Ping(context.Background())
}

func (s *DatabaseService) Shutdown() error {
	s.log.Debug("Closing Database connection...")

	return s.db.Close()
}
