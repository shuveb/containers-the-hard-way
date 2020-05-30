package main

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"os"
)

const gockerHomePath 		= "/var/lib/gocker"
const gockerTempPath 		= gockerHomePath + "/tmp"
const gockerImagesPath 		= gockerHomePath + "/images"
const gockerContainersPath 	= "/var/run/gocker/containers"

func doOrDie(err error) {
	if err != nil {
		log.Fatalf("Fatal error: %v\n", err)
	}
}

func createDirsIfDontExist(dirs []string) (err error){
	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err = os.MkdirAll(dir, 0755); err != nil {
				log.Printf("Error creating directory: %v\n", err)
				return err
			}
		}
	}
	return nil
}

func initGockerDirs() (err error) {
	dirs := []string {gockerHomePath, gockerTempPath, gockerImagesPath, gockerContainersPath}
	return createDirsIfDontExist(dirs)
}

func getGockerHomeDir() string {
	return gockerHomePath
}

func getGockerImagesPath() string {
	return gockerImagesPath
}

func getGockerTempPath() string {
	return gockerTempPath
}

func getGockerContainersPath() string {
	return gockerContainersPath
}

func copyFile(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return
}

func parseManifest(manifestPath string, mani *manifest) (err error) {
	data, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, mani); err != nil {
		return err
	}

	return
}
