package main

import (
	"fmt"
	flag "github.com/spf13/pflag"
	"log"
	"math/rand"
	"os"
	"time"
)

func usage() {
	fmt.Println("Welcome to Gocker!")
	fmt.Println("Supported commands:")
	fmt.Println("gocker run [--mem] [--swap] [--pids] [--cpus] <image> <command>")
	fmt.Println("gocker exec <container-id> <command>")
	fmt.Println("gocker images")
	fmt.Println("gocker rmi <image-id>")
	fmt.Println("gocker ps")
}

func main() {
	options := []string{"run", "child-mode", "setup-netns", "setup-veth", "ps", "exec", "images", "rmi"}

	if len(os.Args) < 2 || !stringInSlice(os.Args[1], options) {
		usage()
		os.Exit(1)
	}
	rand.Seed(time.Now().UnixNano())

	/* We chroot and write to privileged directories. We need to be root */
	if os.Geteuid() != 0 {
		log.Fatal("You need root privileges to run this program.")
	}

	/* Create the directories we require */
	if err := initGockerDirs(); err != nil {
		log.Fatalf("Unable to create requisite directories: %v", err)
	}

	log.Printf("Cmd args: %v\n", os.Args)

	switch os.Args[1] {
	case "run":
		fs := flag.FlagSet{}
		fs.ParseErrorsWhitelist.UnknownFlags = true

		mem := fs.Int("mem", -1, "Max RAM to allow in MB")
		swap := fs.Int("swap", -1, "Max swap to allow in MB")
		pids := fs.Int("pids", -1, "Number of max processes to allow")
		cpus := fs.Float64("cpus", -1, "Number of CPU cores to restrict to")
		if err := fs.Parse(os.Args[2:]); err != nil {
			fmt.Println("Error parsing: ", err)
		}
		if len(fs.Args()) < 2 {
			log.Fatalf("Please pass image name and command to run")
		}
		/* Create and setup the gocker0 network bridge we need */
		if isUp, _ := isGockerBridgeUp(); !isUp {
			log.Println("Bringing up the gocker0 bridge...")
			if err := setupGockerBridge(); err != nil {
				log.Fatalf("Unable to create gocker0 bridge: %v", err)
			}
		}
		initContainer(*mem, *swap, *pids, *cpus, fs.Args()[0], fs.Args()[1:])
	case "child-mode":
		fs := flag.FlagSet{}
		fs.ParseErrorsWhitelist.UnknownFlags = true

		mem := fs.Int("mem", -1, "Max RAM to allow in  MB")
		swap := fs.Int("swap", -1, "Max swap to allow in  MB")
		pids := fs.Int("pids", -1, "Number of max processes to allow")
		cpus := fs.Float64("cpus", -1, "Number of CPU cores to restrict to")
		image := fs.String("img", "", "Container image")
		if err := fs.Parse(os.Args[2:]); err != nil {
			fmt.Println("Error parsing: ", err)
		}
		if len(fs.Args()) < 2 {
			log.Fatalf("Please pass image name and command to run")
		}
		execContainerCommand(*mem, *swap, *pids, *cpus, fs.Args()[0], *image, fs.Args()[1:])
	case "setup-netns":
		setupNewNetworkNamespace(os.Args[2])
	case "setup-veth":
		setupContainerNetworkInterfaceStep1(os.Args[2])
		setupContainerNetworkInterfaceStep2(os.Args[2])
	case "ps":
		printRunningContainers()
	case "exec":
		execInContainer(os.Args[2])
	case "images":
		printAvailableImages()
	case "rmi":
		if len(os.Args) < 3 {
			usage()
			os.Exit(1)
		}
		deleteImageByHash(os.Args[2])
	default:
		usage()
	}
}
