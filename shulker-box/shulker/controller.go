package shulker

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
	"sync"

	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
)

type Controller struct {
	log      *zap.Logger
	listener net.Listener
	idents   []ControllerIdentity
	m        *Minecraft
	mu       sync.Mutex
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
	Config    Config
}

func NewController(ctx context.Context, in NewControllerInput) (*Controller, error) {
	c := &Controller{
		log:    in.Log.Named("Controller"),
		idents: in.Config.Controller.Identities,
	}

	in.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return c.start(ctx, in.Config)
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

func (c *Controller) start(ctx context.Context, config Config) error {
	c.log.Debug("Starting Shulker Controller")

	l, err := net.Listen(config.Controller.Protocol, config.Controller.ListenOn)
	if err != nil {
		c.log.Error("Failed to start Controller Listener", zap.Error(err), zap.String("proto", config.Controller.Protocol), zap.String("addr", config.Controller.ListenOn))
		return err
	}
	c.listener = l

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				runtime.Goexit()
			}
			go c.handleConn(conn)
		}
	}()

	return nil
}

func (c *Controller) stop(ctx context.Context) error {
	c.log.Debug("Stopping Shulker Controller")

	if err := c.listener.Close(); err != nil {
		c.log.Error("Error closing listener", zap.Error(err))
		return err
	}

	if u, ok := c.listener.Addr().(*net.UnixAddr); ok {
		os.Remove(u.Name)
	}

	return nil
}

func (c *Controller) handleConn(conn net.Conn) {
	state := make(map[string]interface{})

	defer func() {
		spew.Dump(state)

		c.log.Debug("Closed Connection", zap.String("addr", conn.RemoteAddr().String()))
		conn.Close()
	}()

	c.log.Debug("Accept", zap.String("addr", conn.RemoteAddr().String()))

	if _, err := conn.Write([]byte("SHULKER/1\n")); err != nil {
		c.log.Error("Failed to write Shulker Version", zap.Error(err))
		runtime.Goexit()
	}

	for {
		input, err := readFromConn(conn)

		if err != nil {
			switch err {
			case io.EOF:
				continue
			default:
				c.log.Error("Conn Error", zap.Error(err))
			}
			runtime.Goexit()
		}

		var buffer buffer.Buffer

		err = c.processRequest(input, &buffer, state)
		switch err {
		case nil:
			if buffer.Len() > 0 {
				out := buffer.Bytes()
				if !bytes.HasSuffix(out, []byte{'\n'}) {
					out = append(out, '\n')
				}
				if _, err := conn.Write(out); err != nil {
					c.log.Error("Failed to write to connection", zap.Error(err))
					runtime.Goexit()
				}
			}
		default:
			errMsg := fmt.Sprintf(`ERR "%v"`, err)
			if _, err := conn.Write(append([]byte(errMsg), '\n')); err != nil {
				c.log.Error("Failed to write to connection", zap.Error(err))
				runtime.Goexit()
			}
		}
	}
}

func (c *Controller) processRequest(in []byte, w io.Writer, state map[string]interface{}) error {
	parts := bytes.SplitN(in, []byte{' '}, 2)
	if len(parts) == 1 {
		parts = append(parts, []byte{})
	}

	switch string(parts[0]) {
	case "PING":
		w.Write([]byte("PONG"))
	case "EXIT":
		runtime.Goexit()
	case "IDENT":
		u, p, err := parseIdentity(parts[1])
		if err != nil {
			return err
		}
		if access, ok := c.authenticate(u, p); ok {
			state["id"] = currentIdent{u: u, l: access}
			w.Write([]byte("OK"))
		} else {
			return errors.New("invalid username or password")
		}
	case "IAM":
		if i, ok := state["id"]; ok {
			if i, ok := i.(currentIdent); ok {
				w.Write([]byte(i.u))
				return nil
			}
		}
		w.Write([]byte(`ERR "Not Identified"`))
	case "REST":
		c.m.Command([]byte("stop"))
		w.Write([]byte("OK"))
	default:
		return fmt.Errorf("unknown command %s", string(parts[0]))
	}

	return nil
}

type currentIdent struct {
	u string
	l string
}

func readFromConn(conn net.Conn) ([]byte, error) {
	var buffer bytes.Buffer
	for {
		reader := bufio.NewReader(io.LimitReader(conn, 65536))
		ba, isPrefix, err := reader.ReadLine()
		if err != nil {
			return nil, err
		}
		buffer.Write(ba)
		if !isPrefix {
			break
		}
	}
	return buffer.Bytes(), nil
}

func parseIdentity(in []byte) (string, string, error) {
	rawInput, err := base64.StdEncoding.DecodeString(string(in))
	if err != nil {
		return "", "", errors.Wrap(err, "failed to decode username/password")
	}
	parts := bytes.SplitN(rawInput, []byte{':'}, 2)
	if len(parts) != 2 {
		return "", "", errors.New("invalid format for username/password")
	}
	return string(parts[0]), string(parts[1]), nil
}

func (c *Controller) authenticate(username, password string) (string, bool) {
	c.log.Debug("Authenticate", zap.String("username", username))

	for _, id := range c.idents {
		if id.Username == username && id.Password == password {
			return id.AccessLevel, true
		}
	}

	return "", false
}
