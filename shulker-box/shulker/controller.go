package shulker

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Controller struct {
	log      *zap.Logger
	listener net.Listener
	m        *Minecraft
	mu       sync.Mutex
	s        *http.Server
}

type ControllerIdentity struct {
	Username    string `hcl:"username,label"`
	Password    string `hcl:"password"`
	AccessLevel string `hcl:"access_level"`
}

type NewControllerInput struct {
	fx.In

	Log       *zap.Logger
	Lifecycle fx.Lifecycle
	Runtime   fx.Shutdowner
	Config    Config
}

func NewController(ctx context.Context, in NewControllerInput) (*Controller, error) {
	c := &Controller{
		log: in.Log.Named("Controller"),
		s: &http.Server{
			Handler: newControllerServerHandler(in.Log, in.Config.Controller.Identities, in.Runtime),
		},
	}

	in.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return c.start(ctx, in.Runtime, in.Config)
		},
		OnStop: func(ctx context.Context) error {
			return c.stop(ctx)
		},
	})

	return c, nil
}

func (c *Controller) Bind(m *Minecraft) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.log.Info("Controller Binding to Minecraft Instance")

	c.m = m

	return nil
}

func (c *Controller) start(ctx context.Context, runtime fx.Shutdowner, config Config) error {
	c.log.Debug("Starting Shulker Controller")

	l, err := net.Listen(config.Controller.Protocol, config.Controller.ListenOn)
	if err != nil {
		c.log.Error("Failed to start Controller Listener", zap.Error(err), zap.String("proto", config.Controller.Protocol), zap.String("addr", config.Controller.ListenOn))
		return err
	}
	c.listener = l

	go func() {
		err := c.s.Serve(l)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			c.log.Error("Control Server Error", zap.Error(err))
			runtime.Shutdown()
		}
	}()

	return nil
}

func (c *Controller) stop(ctx context.Context) error {
	c.log.Debug("Stopping Shulker Controller")

	return c.s.Shutdown(ctx)
}

func newControllerServerHandler(l *zap.Logger, users []ControllerIdentity, run fx.Shutdowner) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/check", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("/shutdown", func(w http.ResponseWriter, r *http.Request) {
		if err := run.Shutdown(); err != nil {
			l.Error("Shutdown Failed", zap.Error(err))
		}
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(http.StatusText(http.StatusNotFound)))
	})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l.Info("Controller Action", zap.String("method", r.Method), zap.String("path", r.URL.Path))

		w.Header().Set("Server", fmt.Sprintf("Shulker/%s", Version))

		httpAuthorizeRequest(
			httpRequireMethod("POST", mux),
			l,
			users,
		).ServeHTTP(w, r)
	})
}

func httpAuthorizeRequest(next http.Handler, l *zap.Logger, users []ControllerIdentity) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if ok {
			for _, user := range users {
				if user.Username == username && user.Password == password {
					l.Debug("Authenticated!", zap.String("username", user.Username), zap.String("access-level", user.AccessLevel))
					next.ServeHTTP(w, r)
					return
				}
			}
		}

		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Not Authorized"))
	})
}

func httpRequireMethod(method string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == method {
			next.ServeHTTP(w, r)
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte("405 - Request Method Not Allowed"))
		}
	})
}
