package main

import (
	"io/ioutil"
	"net"
	"net/http"
	"net/url"

	"github.com/urfave/cli/v2"
)

var controlCommand_Status = &cli.Command{
	Name: `status`,
	Action: func(c *cli.Context) error {
		host := firstFlagValue(`host`, c, `127.0.0.1`).(string)
		port := firstFlagValue(`port`, c, `3000`).(string)

		uri := &url.URL{
			Scheme: "http",
			Host:   net.JoinHostPort(host, port),
			Path:   `/system-status`,
			User:   url.UserPassword(`admin`, `password`),
		}

		resp, err := http.Get(uri.String())
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		return output(string(body))
	},
}
