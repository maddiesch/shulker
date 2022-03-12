package utility

import (
	"errors"
	"os"

	"go.uber.org/zap"
)

func FileExists(l *zap.Logger, path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	} else if errors.Is(err, os.ErrNotExist) {
		return false
	} else {
		l.Error("Failed to find file status. This should not be possible...", zap.Error(err))

		return false
	}
}
