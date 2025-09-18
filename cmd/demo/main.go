package main

import "log"

func main() {
	if err := Run(); err != nil {
		log.Fatal(err)
	}
}
