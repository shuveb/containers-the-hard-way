package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
)

type runningContainerInfo struct {
	containerId string
	image string
	command string
	pid int
}

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
						container := runningContainerInfo{
							containerId: entry.Name(),
							image:       "",
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

	fmt.Println("CONTAINER ID\tIMAGE\tCOMMAND")
	for _, container := range containers {
		fmt.Printf("%s\t\t%s", container.containerId, container.command)
	}
}
