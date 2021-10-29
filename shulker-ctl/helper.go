package main

import "github.com/urfave/cli/v2"

func firstFlagValue(name string, c *cli.Context, fallback string) interface{} {
	for _, c := range c.Lineage() {
		for _, n := range c.LocalFlagNames() {
			if n == name {
				return c.Value(name)
			}
		}
	}
	return fallback
}
