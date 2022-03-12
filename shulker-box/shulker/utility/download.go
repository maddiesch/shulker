package utility

import (
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
)

// DownloadFile fetches a file from the given directory and writes it to the location specified
func DownloadFile(log *zap.Logger, url, path string) error {
	log.Debug("Download", zap.String("url", url), zap.String("path", path))

	startTime := time.Now()
	defer func() {
		log.Debug("Download Completed", zap.String("url", url), zap.String("runtime", time.Since(startTime).String()))
	}()

	tmpFile, err := ioutil.TempFile("/tmp", "shulker-dl")
	if err != nil {
		return err
	}
	defer func() {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
	}()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		return err
	}

	resp.Body.Close()

	if !FileExists(log, filepath.Dir(path)) {
		if err := os.MkdirAll(filepath.Dir(path), 0744); err != nil {
			return err
		}
	}

	tmpFile.Seek(0, 0)

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	if _, err := io.Copy(file, tmpFile); err != nil {
		return err
	}

	return nil
}
