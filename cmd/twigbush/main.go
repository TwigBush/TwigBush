package main

import (
	"log"

	"github.com/TwigBush/gnap-go/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		log.Fatal(err)
	}
}
