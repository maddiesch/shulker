package shulker

import (
	"net/http"
	"os"

	"github.com/angryboat/go-middleware"
	"github.com/maddiesch/shulker/internal/shulker/router"
	"github.com/pkg/errors"
	"github.com/samber/do"
	"golang.org/x/exp/slog"
)

type ControlServerHandlerService struct {
	logger  *slog.Logger
	db      *DatabaseService
	handler http.Handler
}

func NewControlServerHandlerService(i *do.Injector) (*ControlServerHandlerService, error) {
	logger, err := do.Invoke[*slog.Logger](i)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get logger instance")
	}
	logger = logger.With("subsystem", "controller")

	db, err := do.Invoke[*DatabaseService](i)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get database instance")
	}

	mux := http.NewServeMux()

	mux.Handle("/login", router.Handler{
		http.MethodPost: controllerHandlerPostLogin(db),
	})

	stack := middleware.Stack(
		middleware.Recovery(os.Stderr),
		middleware.ResponseRuntime("X-Runtime"),
		middleware.Logger(middleware.NewStructuredRequestLogger(logger)),
	)

	return &ControlServerHandlerService{
		logger:  logger,
		db:      db,
		handler: stack(mux),
	}, nil
}

func (s *ControlServerHandlerService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.handler.ServeHTTP(w, r)
}

func controllerHandlerPostLogin(db *DatabaseService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

	})
}
