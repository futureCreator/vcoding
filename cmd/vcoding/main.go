package main

import (
	"os"

	"github.com/futureCreator/vcoding/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
