package main

import (
	"fmt"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "shulker-ctl",
		Usage: "fight the loneliness!",
		Commands: []*cli.Command{
			versionCommand,
			controllerCommand,
			logCommand,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

var versionCommand = &cli.Command{
	Name:      `version`,
	Usage:     `print the version`,
	UsageText: `shulker-ctl version`,
	HideHelp:  true,
	Action: func(c *cli.Context) error {
		return output(`Shulker CTL - Version 1`)
	},
}

var controllerCommand = &cli.Command{
	Name:  `control`,
	Usage: `Send commands to a running control server`,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  `host`,
			Value: `127.0.0.1`,
		},
		&cli.StringFlag{
			Name:  `port`,
			Value: `3000`,
		},
	},
	Subcommands: []*cli.Command{
		controlCommand_Status,
	},
}

var logCommand = &cli.Command{
	Name:  `logs`,
	Usage: ``,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     `path`,
			Aliases:  []string{`p`},
			Required: true,
		},
	},
	Action: logsCommandAction,
}

var output = func(msg string, args ...interface{}) error {
	final := fmt.Sprintf(msg, args...)
	if !strings.HasSuffix(final, "\n") {
		final += "\n"
	}

	_, err := fmt.Fprint(os.Stdout, final)

	return err
}
