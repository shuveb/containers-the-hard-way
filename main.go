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

	//src := "alpine:latest"
	src := "ubuntu:20.04"
	//src := "centos:8"
	runContainer(src, "/bin/sh")
}
