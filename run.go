package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

func createContainerID() (contID string){
	randBytes := make([]byte, 6)
	rand.Read(randBytes)
	contID = fmt.Sprintf("%02x%02x%02x%02x%02x%02x",
						randBytes[0], randBytes[1], randBytes[2],
						randBytes[3], randBytes[4], randBytes[5])
	return contID
}

func getContainerFSHome(contanerID string) string {
	return getGockerContainersPath() + "/" + contanerID + "/fs"
}

func createContainerDirectories(containerID string) {
	contHome := getGockerContainersPath() + "/" + containerID
	contDirs := []string{contHome + "/fs", contHome + "/fs/mnt", contHome + "/fs/upperdir", contHome + "/fs/workdir"}
	if err := createDirsIfDontExist(contDirs); err != nil {
		log.Fatalf("Unable to create required directories: %v\n", err)
	}
}

func mountOverlayFileSystem(containerID string, imageShaHex string) {
	var srcLayers []string
	pathManifest :=  getManifestPathForImage(imageShaHex)
	mani := manifest{}
	parseManifest(pathManifest, &mani)
	if len(mani) == 0 || len(mani[0].Layers) == 0 {
		log.Fatal("Could not find any layers.")
	}
	if len(mani) > 1 {
		log.Fatal("I don't know how to handle more than one manifest.")
	}

	imageBasePath := getBasePathForImage(imageShaHex)
	for _, layer := range mani[0].Layers {
		srcLayers = append([]string{imageBasePath + "/" + layer[:12] + "/fs"}, srcLayers...)
		//srcLayers = append(srcLayers, imageBasePath + "/" + layer[:12] + "/fs")
	}
	contFSHome := getContainerFSHome(containerID)
	mntOptions := "lowerdir="+strings.Join(srcLayers, ":")+",upperdir="+contFSHome+"/upperdir,workdir="+contFSHome+"/workdir"
	if err:= syscall.Mount("none", contFSHome + "/mnt", "overlay", 0, mntOptions); err != nil {
		log.Fatalf("Mount failed: %v\n", err)
	}
}

/*
	Called if this program is executed with "child-mode" as the first argument
*/
func execContainerCommand(containerID string) {
	mntPath := getContainerFSHome(containerID) + "/mnt"
	cmd := exec.Command(os.Args[3], os.Args[4:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	doOrDie(syscall.Sethostname([]byte(containerID)))
	doOrDie(syscall.Chroot(mntPath))
	doOrDie(os.Chdir("/"))
	doOrDie(syscall.Mount("proc", "/proc", "proc", 0, ""))
	cmd.Run()
	doOrDie(syscall.Unmount("/proc", 0))
}

func spawnChild(containerID string) {
	args := append([]string{containerID}, os.Args[2:]...)
	args = append([]string{"child-mode"}, args...)
	log.Printf("Child command args: %s\n", args)
	cmd := exec.Command("/proc/self/exe", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWUTS,
		Unshareflags: syscall.CLONE_NEWNS,
	}
	doOrDie(cmd.Run())
}

func initContainer(src string)  {
	containerID := createContainerID()
	log.Printf("New container ID: %s\n", containerID)
	imageShaHex := downloadImageIfRequired(src)
	log.Printf("Image to overlay mount: %s\n", imageShaHex)
	createContainerDirectories(containerID)
	mountOverlayFileSystem(containerID, imageShaHex)
	spawnChild(containerID)
	log.Printf("Container done.\n")
	doOrDie(syscall.Unmount(getContainerFSHome(containerID) + "/mnt", 0))
}
