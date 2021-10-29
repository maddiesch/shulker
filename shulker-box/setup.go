package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"shulker-box/config"
	"shulker-box/logger"
	"time"

	"github.com/maddiesch/go-cargo"
)

func prepareShulkerForRunning(ctx context.Context, cfg config.Config, forceUpdate bool) error {
	if _, err := cfg.JavaCommand(); err != nil {
		logger.L.Errorf(`Failed to find Java: %v`, err)
		return err
	}

	if !logger.FileExistsForPath(cfg.WorkingDir) {
		logger.L.Infof(`Prepare working directory - %s`, cfg.WorkingDir)

		if err := os.MkdirAll(cfg.WorkingDir, 0744); err != nil {
			return err
		}
	}

	if !logger.FileExistsForPath(cfg.ServerJar()) || forceUpdate {
		logger.L.Infof(`Updating Server Jar`)

		if err := downloadFile(ctx, cfg.Minecraft.Server.DownloadURL, cfg.ServerJar()); err != nil {
			return err
		}
	}

	for _, plugin := range cfg.Minecraft.Plugins {
		if err := downloadPluginIfNeeded(ctx, cfg.WorkingDir, plugin, forceUpdate); err != nil {
			return err
		}
	}

	if err := createMojangEulaFileIfNeeded(ctx, cfg.WorkingDir, os.Getenv(`AUTO_ACCEPT_MINECRAFT_EULA`) == `true`); err != nil {
		return err
	}

	return nil
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

	logger.L.WithField(`subsystem`, `downloader`).Debugf(`Download file %s`, fromURL)

	_, err = cargo.Download(ctx, cargo.DownloadInput{
		Source:           location,
		Dest:             jarFile,
		ValidateResponse: cargo.ValidateStatusCodeEqual(http.StatusOK),
	})

	if err != nil {
		jarFile.Close()
		os.Remove(jarFile.Name())
	}

	return err
}

func createMojangEulaFileIfNeeded(ctx context.Context, workingDir string, explicitAccept bool) error {
	eulaFilePath := filepath.Join(workingDir, `eula.txt`)
	if logger.FileExistsForPath(eulaFilePath) {
		return nil
	}

	logger.L.Infof(`Missing Mojang EULA (Explicit Auto-Accept: %v)`, explicitAccept)

	var buf bytes.Buffer
	buf.WriteString(`# Minecraft EULA file created by Shulker`)
	buf.WriteByte('\n')
	buf.WriteString(`# Auto-Accept explicitly enabled by end user using environment "AUTO_ACCEPT_MINECRAFT_EULA=true"`)
	buf.WriteByte('\n')
	buf.WriteString(fmt.Sprintf(`# Created At - %s`, time.Now().UTC().Format(time.RFC1123)))
	buf.WriteByte('\n')
	buf.WriteString(fmt.Sprintf(`eula=%v`, explicitAccept))
	buf.WriteByte('\n')

	f, err := os.Create(eulaFilePath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(buf.Bytes())

	return err
}

func downloadPluginIfNeeded(ctx context.Context, workingDir string, p config.MinecraftPlugin, forceUpdate bool) error {
	pluginDir := filepath.Join(workingDir, `plugins`)
	if !logger.FileExistsForPath(pluginDir) {
		logger.L.Debugf(`Creating plugins directory - %s`, pluginDir)

		if err := os.MkdirAll(pluginDir, 0755); err != nil {
			return err
		}
	}

	pluginJarFile := filepath.Join(pluginDir, fmt.Sprintf("%s.jar", p.Name))
	if !logger.FileExistsForPath(pluginJarFile) || forceUpdate {
		logger.L.Debugf(`Update plugin %s`, p.Name)
		if err := downloadFile(ctx, p.Source, pluginJarFile); err != nil {
			if p.Required {
				return err
			}

			logger.L.Errorf(`Download for plugin (%s) failed: %v`, p.Name, err)

			return nil
		}
	}

	return nil
}
