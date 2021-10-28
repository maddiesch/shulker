package main

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/maddiesch/go-cargo"
	log "github.com/sirupsen/logrus"
)

func performSetupWithForcedUpdate(cfg shulkerConfig, forceUpdate bool) error {
	if !fileExistsForPath(cfg.WorkingDir) {
		log.Printf("Setup Working Directory: %s", cfg.WorkingDir)

		if err := os.MkdirAll(cfg.WorkingDir, 0744); err != nil {
			return err
		}
	}

	logFile, err := rotateLogAndOpenForWriting(cfg.LogPath)
	if err != nil {
		return err
	}

	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(io.MultiWriter(os.Stdout, logFile))

	if !fileExistsForPath(cfg.Minecraft.Server.JarPath) || forceUpdate {
		if err := downloadFile(context.Background(), cfg.Minecraft.Server.DownloadURL, cfg.Minecraft.Server.JarPath); err != nil {
			return err
		}
	}

	eulaFilePath := filepath.Join(cfg.WorkingDir, `eula.txt`)
	if !fileExistsForPath(eulaFilePath) {
		log.Print(`Eula file not found`)

		if os.Getenv(`AUTO_ACCEPT_MINECRAFT_EULA`) == `true` {
			f, err := os.Create(eulaFilePath)
			if err != nil {
				return err
			}
			defer f.Close()

			f.Write([]byte("# EULA Accepted by Shulker explicit AUTO_ACCEPT_MINECRAFT_EULA=true\neula=true\n"))
		}
	}

	unsafeLogFilePtr = logFile

	return nil
}

var unsafeLogFilePtr *os.File

func fileExistsForPath(p string) bool {
	_, err := os.Stat(p)
	if os.IsNotExist(err) {
		return false
	} else if err != nil {
		panic(err)
	} else {
		return true
	}
}

func downloadFile(ctx context.Context, fromURL, toPath string) error {
	location, err := url.Parse(fromURL)
	if err != nil {
		return err
	}
	jarFile, err := os.Create(toPath)
	if err != nil {
		return err
	}

	defer jarFile.Close()

	log.WithField(`subsystem`, `download`).Debugf("Downloading %s", fromURL)

	_, err = cargo.Download(ctx, cargo.DownloadInput{
		Source: location,
		Dest:   jarFile,
	})

	return err
}

func rotateLogAndOpenForWriting(path string) (*os.File, error) {
	if fileExistsForPath(path) {
		oldLogFile := filepath.Join(filepath.Dir(path), fmt.Sprintf("shulker-%d.log", time.Now().Unix()))
		os.Rename(path, oldLogFile)
	}
	return os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0655)
}
