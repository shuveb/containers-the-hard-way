package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"strconv"
)

func createCGroups(containerID string, createCGroupDirs bool) {
	cgroups := []string{"/sys/fs/cgroup/memory/gocker/" + containerID,
						"/sys/fs/cgroup/pids/gocker/" + containerID,
						"/sys/fs/cgroup/cpu/gocker/" + containerID}

	if createCGroupDirs {
		doOrDieWithMsg(createDirsIfDontExist(cgroups),
			"Unable to create cgroup directories")
	}

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

func setMemoryLimit(containerID string, limitMB int, swapLimitInMB int) {
	memFilePath := "/sys/fs/cgroup/memory/gocker/" + containerID +
											"/memory.limit_in_bytes"
	swapFilePath := "/sys/fs/cgroup/memory/gocker/" + containerID +
		"/memory.memsw.limit_in_bytes"
	doOrDieWithMsg(ioutil.WriteFile(memFilePath,
				[]byte(strconv.Itoa(limitMB*1024*1024)), 0644),
				"Unable to write memory limit")

	/*
		memory.memsw.limit_in_bytes contains the total amount of memory the
		control group can consume: this includes both swap and RAM.
		If if memory.limit_in_bytes is specified but memory.memsw.limit_in_bytes
		is left untouched, processes in the control group will continue to
		consume swap space.
	*/
	if swapLimitInMB >= 0 {
		doOrDieWithMsg(ioutil.WriteFile(swapFilePath,
			[]byte(strconv.Itoa((limitMB*1024*1024)+(swapLimitInMB*1024*1024))),
			0644), "Unable to write memory limit")
	}
}

func setCpuLimit(containerID string, limit float64)  {
	cfsPeriodPath := "/sys/fs/cgroup/cpu/gocker/" + containerID +
		"/cpu.cfs_period_us"
	cfsQuotaPath := "/sys/fs/cgroup/cpu/gocker/" + containerID +
		"/cpu.cfs_quota_us"

	if limit > float64(runtime.NumCPU()) {
		fmt.Printf("Ignoring attempt to set CPU quota to great than number of available CPUs")
		return
	}

	doOrDieWithMsg(ioutil.WriteFile(cfsPeriodPath,
		[]byte(strconv.Itoa(1000000)), 0644),
		"Unable to write CFS period")

	doOrDieWithMsg(ioutil.WriteFile(cfsQuotaPath,
		[]byte(strconv.Itoa(int(1000000 * limit))), 0644),
		"Unable to write CFS period")

}

func setPidsLimit(containerID string, limit int)  {
	maxProcsPath := "/sys/fs/cgroup/pids/gocker/" + containerID +
		"/pids.max"

	doOrDieWithMsg(ioutil.WriteFile(maxProcsPath,
		[]byte(strconv.Itoa(limit)), 0644),
		"Unable to write pids limit")

}

func configureCGroups(containerID string, mem int, swap int, pids int, cpus float64) {
	if mem > 0 {
		setMemoryLimit(containerID, mem, swap)
	}
	if cpus > 0 {
		setCpuLimit(containerID, cpus)
	}
	if pids > 0 {
		setPidsLimit(containerID, pids)
	}
}
