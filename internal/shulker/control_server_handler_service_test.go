package shulker_test

import (
	"context"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/maddiesch/shulker/internal/shulker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slog"
)

func TestControlServerHandlers(t *testing.T) {
	app := shulker.NewApp(shulker.NewAppInput{
		Logger: slog.New(slog.NewTextHandler(os.Stderr, nil)),
		Config: shulker.Config{
			DatabasePath:  ":memory:?cache=shared",
			ServerAddress: "127.0.0.1",
			ServerPort:    "9999",
		},
	})

	err := app.Start(context.Background())
	require.NoError(t, err)
	defer app.Stop(context.Background())

	sendRequest := func(m string, p string, b string) ([]byte, int, *http.Response) {
		req, err := http.NewRequest(m, "http://127.0.0.1:9999"+p, strings.NewReader(b))
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)

		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)

		return body, resp.StatusCode, resp
	}

	t.Run("POST /login", func(t *testing.T) {
		t.Run("when given a valid username and password", func(t *testing.T) {
			_, status, _ := sendRequest(http.MethodPost, "/login", `{"username":"admin","password":"password"}`)

			assert.Equal(t, 200, status)
		})

		t.Run("when given an invalid username", func(t *testing.T) {
			_, status, _ := sendRequest(http.MethodPost, "/login", `{"username":"foobar","password":"password"}`)

			assert.Equal(t, 401, status)
		})
	})
}
