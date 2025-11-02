package main

import (
	"context"
	"errors"
	"os"

	"github.com/alecthomas/kong"
)

type CLI struct {
	Serve Serve `embed:""`
}

func main() {
	var cli CLI

	kong.Parse(&cli)

	err := serve(&cli)
	if err != nil && errors.Is(err, context.Canceled) {
		os.Exit(1)
	}
}
