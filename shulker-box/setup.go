package main

import (
	"context"
	"log"
	"os"
	"shulker-box/cargo"
)

func performSetupWithForcedUpdate(cfg shulkerConfig, forceUpdate bool) error {
	if !fileExistsForPath(cfg.WorkingDir) {
		log.Printf("Setup Working Directory: %s", cfg.WorkingDir)

		if err := os.MkdirAll(cfg.WorkingDir, 0744); err != nil {
			return err
		}
	}

	if !fileExistsForPath(cfg.Minecraft.Server.JarPath) || forceUpdate {
		if err := cargo.Download(context.Background(), cfg.Minecraft.Server.DownloadURL, cfg.Minecraft.Server.JarPath); err != nil {
			return err
		}
	}

	return nil
}

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
