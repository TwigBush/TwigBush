package main

import "log"

func main() {
	log.Print("Initializing TwigBush Demo")
	if err := Run(); err != nil {
		log.Fatal(err)
	}
}
