package logger_test

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"shulker-box/logger"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	logger.L.SetLevel(log.TraceLevel)

	os.Exit(m.Run())
}

func TestCreateLog(t *testing.T) {
	t.Run(`create log file and then open to rotate`, func(t *testing.T) {
		filePath := createTempLogFilePath()

		w, err := logger.CreateLog(filePath)
		require.NoError(t, err)

		w.Write([]byte("Hello world!\n"))

		w.Close()

		w, err = logger.CreateLog(filePath)
		require.NoError(t, err)

		w.Close()
	})
}

func createTempLogFilePath() string {
	data := make([]byte, 16)
	rand.Read(data)
	name := base64.RawURLEncoding.EncodeToString(data)
	return filepath.Join(os.TempDir(), `shulker-test`, fmt.Sprintf("log-%s.log", name))
}
