package logger

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

var (
	L    = log.New()
	sLog = L.WithField(`subsystem`, `logger`)
)

// TODO: - Write documentation
func CreateLog(path string) (io.WriteCloser, error) {
	l := sLog.WithField(`log_path`, path)

	l.Tracef(`CreateLog`)

	if !filepath.IsAbs(path) {
		workingDir, err := os.Getwd()
		if err != nil {
			return nil, err
		}

		path = filepath.Join(workingDir, path)
	}

	if err := createLogDirectoryIfNeeded(l, filepath.Dir(path)); err != nil {
		return nil, err
	}
	if err := rotateExistingLogWithPath(l, path); err != nil {
		return nil, err
	}
	if err := cleanupPreviousRotatedLogPath(l, filepath.Dir(path)); err != nil {
		return nil, err
	}

	return os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0655)
}

// TODO: - Write documentation
func FileExistsForPath(path string) bool {
	_, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false
	} else if err == nil {
		return true
	} else {
		panic(err)
	}
}

func createLogDirectoryIfNeeded(l *log.Entry, dir string) error {
	if FileExistsForPath(dir) {
		return nil
	}

	l.Tracef(`Creating Log Directory`)

	return os.MkdirAll(dir, 0766)
}

func rotateExistingLogWithPath(l *log.Entry, path string) error {
	if !FileExistsForPath(path) {
		return nil
	}
	l.Debugf(`Rotate Existing Log File`)

	logFileName := fmt.Sprintf(
		"%s-%d.log.gz",
		strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
		time.Now().Unix(),
	)
	logFileForReading, err := os.Open(path)
	if err != nil {
		return err
	}
	defer logFileForReading.Close()

	logFileForWriting, err := os.Create(filepath.Join(filepath.Dir(path), logFileName))
	if err != nil {
		return err
	}
	defer logFileForWriting.Close()
	rotateDest := gzip.NewWriter(logFileForWriting)
	defer rotateDest.Close()

	_, err = io.Copy(rotateDest, logFileForReading)

	return err
}

func cleanupPreviousRotatedLogPath(l *log.Entry, dir string) error {
	l.Tracef(`Rotate Previous Log Files`)

	maxLogFiles := 3

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}

	var oldLogFiles []fs.FileInfo

	for _, stat := range files {
		if !strings.HasSuffix(stat.Name(), `.log.gz`) {
			continue
		}
		oldLogFiles = append(oldLogFiles, stat)
	}
	if len(oldLogFiles) < maxLogFiles {
		return nil
	}

	sort.Slice(oldLogFiles, func(i, j int) bool {
		return oldLogFiles[i].ModTime().Before(oldLogFiles[j].ModTime())
	})

	filesToDelete := oldLogFiles[:len(oldLogFiles)-maxLogFiles]

	for _, stat := range filesToDelete {
		if err := os.Remove(filepath.Join(dir, stat.Name())); err != nil {
			return err
		}
	}

	return nil
}
