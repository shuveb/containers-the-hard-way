package main

import (
	"io/ioutil"
	"os"
	"strconv"
)

func createCGroups(containerID string) {
	cgroups := []string{"/sys/fs/cgroup/memory/gocker/" + containerID,
						"/sys/fs/cgroup/pids/gocker/" + containerID,
						"/sys/fs/cgroup/cpu/gocker/" + containerID}

	doOrDieWithMsg(createDirsIfDontExist(cgroups),
		"Unable to create cgroup directories")

	for _, cgroupDir := range cgroups {
		doOrDieWithMsg(ioutil.WriteFile(cgroupDir + "/notify_on_release", []byte("1"), 0700),
			"Unable to write to cgroup notification file")
		doOrDieWithMsg(ioutil.WriteFile(cgroupDir + "/cgroup.procs",
			[]byte(strconv.Itoa(os.Getpid())), 0700), "Unable to write to cgroup procs file")
	}
}

func removeCGroups(containerID string) {
	cgroups := []string{"/sys/fs/cgroup/memory/gocker/" + containerID,
		"/sys/fs/cgroup/pids/gocker/" + containerID,
		"/sys/fs/cgroup/cpu/gocker/" + containerID}

	for _, cgroupDir := range cgroups {
		doOrDieWithMsg(os.Remove(cgroupDir), "Unable to remove cgroup dir")
	}
}

func setMemoryLimit(containerID string, limitMB int) {
	filePath := "/sys/fs/cgroup/memory/gocker/" + containerID + "/memory.limit_in_bytes"
	doOrDieWithMsg(ioutil.WriteFile(filePath,
				[]byte(strconv.Itoa(limitMB*1024*1024)), 0644),
				"Unable to write memory limit")

}