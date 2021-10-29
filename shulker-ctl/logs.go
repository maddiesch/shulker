package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hpcloud/tail"
	"github.com/urfave/cli/v2"
)

func logsCommandAction(c *cli.Context) error {
	workingDir, err := os.Getwd()
	if err != nil {
		return err
	}

	logFilePath := filepath.Join(workingDir, c.Value(`path`).(string))

	t, err := tail.TailFile(logFilePath, tail.Config{Follow: true})
	if err != nil {
		return err
	}

	for line := range t.Lines {
		fmt.Println(line.Text)
	}

	return nil
}
