package main

import (
	"context"
	"errors"
	"net/http"
	"shulker-box/config"
	"shulker-box/logger"
	"sync"

	"github.com/angryboat/go-dispatch"
	"github.com/gorilla/mux"
)

const (
	dispatchEventName_ControllerStopped = `controller.stopped`
)

var (
	sLog = logger.L.WithField(`subsystem`, `control-server`)
)

func runControlServer(cfg config.Config) {
	defer dispatch.Send(dispatch.NullEvent(dispatchEventName_ControllerStopped))

	execute := func() error {
		s := http.Server{
			Addr:    cfg.ControlServerAddr(),
			Handler: createControlServerHandler(cfg.ControlServer.Users),
		}

		sLog.Printf("Starting control server on %s", s.Addr)

		shutdownCancel := dispatch.Receive(context.Background(), dispatchEventName_Shutdown, func(ctx context.Context, _ dispatch.Event) {
			s.Shutdown(ctx)
		})
		defer shutdownCancel()

		killCancel := dispatch.Receive(context.Background(), dispatchEventName_Kill, func(context.Context, dispatch.Event) {
			s.Close()
		})
		defer killCancel()

		return s.ListenAndServe()
	}

	stateCancel := dispatch.Receive(context.Background(), dispatchEventName_MinecraftStateChanged, func(_ context.Context, e dispatch.Event) {
		lastMinecraftStateUpdateMu.Lock()
		lastMinecraftStateUpdate = e.Value().(string)
		lastMinecraftStateUpdateMu.Unlock()
	})
	defer stateCancel()

	for {
		err := execute()

		if errors.Is(err, http.ErrServerClosed) {
			break
		}
	}

	sLog.Print(`Command Server Shutdown`)
}

func createControlServerHandler(users []config.ControlServerUser) http.Handler {
	router := mux.NewRouter()

	router.Methods(`GET`).Path(`/system-status`).Handler(controlServerSystemStatusHandler)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sLog.Printf(`[%s] %s`, r.Method, r.URL.Path)

		if authenticate(w, r, users) {
			router.ServeHTTP(w, r)
		}
	})
}

func authenticate(w http.ResponseWriter, r *http.Request, users []config.ControlServerUser) bool {
	username, password, ok := r.BasicAuth()
	if ok {
		for _, user := range users {
			if user.Username == username && user.Password == password {
				sLog.Printf(`Authenticated %s as %s`, r.RemoteAddr, username)
				return true
			}
		}
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
	http.Error(w, "Unauthorized", http.StatusUnauthorized)

	return false
}

var lastMinecraftStateUpdateMu sync.Mutex
var lastMinecraftStateUpdate = `unknown`

var controlServerSystemStatusHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	lastMinecraftStateUpdateMu.Lock()
	w.Write([]byte(lastMinecraftStateUpdate))
	lastMinecraftStateUpdateMu.Unlock()
})
