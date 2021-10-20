package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
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
