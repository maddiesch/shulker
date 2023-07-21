package shulker

import (
	"context"
	"embed"
	"io/fs"
	"strings"

	"github.com/maddiesch/go-raptor"
	"github.com/maddiesch/go-raptor/statement"
	"github.com/maddiesch/go-raptor/statement/conditional"
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

//go:embed migrations/*.sql
var databaseMigrations embed.FS

func (s *DatabaseService) ExecuteDatabaseMigration(ctx context.Context) error {
	s.log.Debug("Performing database migration")

	_, err := s.db.Exec(ctx, `CREATE TABLE IF NOT EXISTS "Migrations" ("Name" TEXT PRIMARY KEY);`)
	if err != nil {
		return errors.Wrap(err, "failed to create migrations table")
	}

	files, err := fs.ReadDir(databaseMigrations, "migrations")
	if err != nil {
		return errors.Wrap(err, "failed to read migrations directory")
	}
	for _, entry := range files {
		if !strings.HasSuffix(entry.Name(), ".up.sql") {
			continue
		}
		query := statement.Exists(
			statement.Select("1").From("Migrations").Where(conditional.Equal("Name", entry.Name())),
		)

		var exists bool

		if err := s.db.QueryRowStatement(ctx, query).Scan(&exists); err != nil {
			return errors.Wrap(err, "failed to check if migration exists")
		}

		if exists {
			continue
		}

		s.log.Debug("Execute migration", slog.String("migration", entry.Name()))

		content, err := fs.ReadFile(databaseMigrations, "migrations/"+entry.Name())
		if err != nil {
			return errors.Wrap(err, "failed to read migration file")
		}

		err = s.db.Transact(ctx, func(tx raptor.DB) error {
			if _, err := tx.Exec(ctx, string(content)); err != nil {
				return errors.Wrap(err, "failed to execute migration")
			}

			_, err := raptor.ExecStatement(ctx, tx, statement.Insert().Into("Migrations").Value("Name", entry.Name()))

			return err
		})

		if err != nil {
			s.log.Error("Failed to execute migration", slog.String("migration", entry.Name()), slog.String("error", err.Error()))
			return errors.Wrap(err, "failed to execute migration")
		}
	}

	return nil
}

func (s *DatabaseService) Get(ctx context.Context, key string) ([]byte, bool) {
	var value []byte

	err := s.db.QueryRowStatement(ctx, statement.Select("Value").From("KeyValue").Where(conditional.Equal("Key", key))).Scan(&value)
	if errors.Is(err, raptor.ErrNoRows) {
		return nil, false
	}

	return value, err == nil
}

func (s *DatabaseService) Set(ctx context.Context, key string, value []byte) error {
	return s.db.Transact(ctx, func(tx raptor.DB) error {
		return s.SetX(ctx, tx, key, value)
	})
}

func (s *DatabaseService) SetX(ctx context.Context, e raptor.Executor, key string, value []byte) error {
	_, err := raptor.ExecStatement(ctx, e, statement.Insert().OrReplace().Into("KeyValue").Value("Key", key).Value("Value", value))
	return err
}

func (s *DatabaseService) HealthCheck() error {
	return s.db.Ping(context.Background())
}

func (s *DatabaseService) Shutdown() error {
	s.log.Debug("Closing Database connection...")

	return s.db.Close()
}
