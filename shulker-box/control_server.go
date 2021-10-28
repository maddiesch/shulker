package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strconv"
	"sync"

	"github.com/angryboat/go-dispatch"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

const (
	dispatchEventName_ControllerStopped = `controller.stopped`
)

var (
	sLog = log.WithField(`subsystem`, `control-server`)
)

func runControlServer(cfg shulkerConfig) {
	defer dispatch.Send(dispatch.NullEvent(dispatchEventName_ControllerStopped))

	execute := func() error {
		s := http.Server{
			Addr:    net.JoinHostPort(cfg.ControlServer.Host, strconv.Itoa(cfg.ControlServer.Port)),
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

func createControlServerHandler(users []ControlServerUser) http.Handler {
	router := mux.NewRouter()

	router.Methods(`GET`).Path(`/system-status`).Handler(controlServerSystemStatusHandler)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sLog.Printf(`[%s] %s`, r.Method, r.URL.Path)

		if authenticate(w, r, users) {
			router.ServeHTTP(w, r)
		}
	})
}

func authenticate(w http.ResponseWriter, r *http.Request, users []ControlServerUser) bool {
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
