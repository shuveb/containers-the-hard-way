package main

import (
	"log"
	"math/rand"
	"os"
	"time"
)

func main() {
	log.Println("Welcome to Gocker!")
	rand.Seed(time.Now().UnixNano())

	/* We chroot and write to privileged directories. We need to be root */
	if os.Geteuid() != 0 {
		log.Fatal("You need root privileges to run this program.")
	}
	initGockerDirs()

	if os.Args[1] == "child-mode" {
		execContainerCommand(os.Args[2])
		os.Exit(0)
	}
	initContainer(os.Args[1])
}
