package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type runningContainerInfo struct {
	containerId string
	image string
	command string
	pid int
}

/*
	This isn't a great implementation and can possibly be simplified
	using regex. But for now, here we are. This function gets the
	current mount points, figures out which image is mounted for a
	given container ID, looks it up in our images database which we
	maintain and returns the image and tag information.
*/

func getDistribution(containerID string) (string, error) {
	var lines []string
	file, err := os.Open("/proc/mounts")
	if err != nil {
		fmt.Println("Unable to read /proc/mounts")
		return "", err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	for _, line := range lines {
		if strings.Contains(line, containerID) {
			parts := strings.Split(line, " ")
			for _, part := range parts {
				if strings.Contains(part, "lowerdir=") {
					options := strings.Split(part, ",")
					for _, option := range options {
						if strings.Contains(option, "lowerdir=") {
							imagesPath := getGockerImagesPath()
							leaderString := "lowerdir=" + imagesPath + "/"
							trailerString := option[len(leaderString):]
							imageID := trailerString[:12]
							image, tag := getImageAndTagForHash(imageID)
							return fmt.Sprintf("%s:%s", image, tag), nil
						}
					}
				}
			}
		}
	}
	return "", nil
}

/*
	Get the list of running container IDs.

	Implementation logic:
	- Gocker creates multiple folders in the /sys/fs/cgroup hierarchy
	- For example, for setting cpu limits, gocker uses /sys/fs/cgroup/cpu/gocker
	- Inside that folder are folders one each for currently running containers
	- Those folder names are the container IDs we create.
	- This function does not stop here. It gathers more information about running
		containers. See struct runningContainerInfo for details.
	- Inside each of those folders is a "cgroup.procs" file that has the list
		of PIDs of processes inside of that container. From the PID, we can
		get the mounted path from which the process was started. From that
		mounted path, we can get the image of the containers since containers
		are mounted via the overlay file system.
*/

func getRunningContainers() ([]runningContainerInfo, error) {
	var containers []runningContainerInfo
	var procs []string
	basePath := "/sys/fs/cgroup/cpu/gocker"

	entries, err := ioutil.ReadDir(basePath)
	if os.IsNotExist(err) {
		return containers, nil
	} else {
		if err != nil {
			return nil, err
		} else {
			for _, entry := range entries {
				if entry.IsDir() {
					file, err := os.Open(basePath + "/" + entry.Name() + "/cgroup.procs")
					if err != nil {
						fmt.Println("Unable to read cgroup.procs")
						return nil, err
					}
					defer file.Close()
					scanner := bufio.NewScanner(file)
					scanner.Split(bufio.ScanLines)
					for scanner.Scan() {
						procs = append(procs, scanner.Text())
					}
					if len(procs) > 0 {
						pid, err :=	strconv.Atoi(procs[len(procs) -1 ])
						if err != nil {
							fmt.Println("Unable to read PID")
							return nil, err
						}
						cmd, err := os.Readlink("/proc/" + strconv.Itoa(pid) + "/exe")
						containerMntPath := getGockerContainersPath() + "/" + entry.Name() + "/fs/mnt"
						realContainerMntPath, err := filepath.EvalSymlinks(containerMntPath)
						if err != nil {
							fmt.Println("Unable to resolve path")
							return nil, err
						}

						if err != nil {
							fmt.Println("Unable to read command link.")
							return nil, err
						}
						image, _ := getDistribution(entry.Name())
						container := runningContainerInfo{
							containerId: entry.Name(),
							image:       image,
							command:     cmd[len(realContainerMntPath):],
							pid:		 pid,
						}
						containers = append(containers, container)
					}
				}
			}
			return containers, nil
		}
	}
}

func printRunningContainers() {
	containers, err := getRunningContainers()
	if err != nil {
		os.Exit(1)
	}

	fmt.Println("CONTAINER ID\tIMAGE\t\tCOMMAND")
	for _, container := range containers {
		fmt.Printf("%s\t%s\t%s\n", container.containerId, container.image, container.command)
	}
}
