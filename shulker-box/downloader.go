package main

import (
	"io"
	"log"
	"net/http"
	"os"
)

func downloadLargeFileFromURL(url, path string) error {
	log.Printf("Downloading... %s", url)

	out, err := os.CreateTemp("", "shulker-download-*")
	if err != nil {
		return err
	}
	defer func() {
		out.Close()
		if err := os.Remove(out.Name()); err != nil {
			panic(err)
		}
	}()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return err
	}

	if _, err := out.Seek(0, 0); err != nil {
		return err
	}

	final, err := os.Create(path)
	if err != nil {
		return err
	}
	defer final.Close()

	if _, err := io.Copy(final, out); err != nil {
		return err
	}

	return nil
}

func checkFileExistsAtPath(p string) bool {
	_, err := os.Stat(p)
	if os.IsNotExist(err) {
		return false
	} else if err != nil {
		panic(err)
	} else {
		return true
	}
}
