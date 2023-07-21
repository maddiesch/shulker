package shulker

import (
	"net/http"

	"github.com/pkg/errors"
	"github.com/samber/do"
	"golang.org/x/exp/slog"
)

type ControlServerHandlerService struct {
	logger *slog.Logger
	db     *DatabaseService
}

func NewControlServerHandlerService(i *do.Injector) (*ControlServerHandlerService, error) {
	logger, err := do.Invoke[*slog.Logger](i)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get logger instance")
	}

	db, err := do.Invoke[*DatabaseService](i)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get database instance")
	}

	return &ControlServerHandlerService{
		logger: logger.With("subsystem", "controller"),
		db:     db,
	}, nil
}

func (s *ControlServerHandlerService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.logger.Info("Request Received", slog.String("method", r.Method), slog.String("path", r.URL.Path))

	w.WriteHeader(http.StatusNotImplemented)
}
