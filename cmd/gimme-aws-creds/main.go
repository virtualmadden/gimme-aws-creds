package main

import (
	"os"

	"github.com/virtualmadden/gimme-aws-creds/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
