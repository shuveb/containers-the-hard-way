package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

func createContainerID() string {
	randBytes := make([]byte, 6)
	rand.Read(randBytes)
	return fmt.Sprintf("%02x%02x%02x%02x%02x%02x",
						randBytes[0], randBytes[1], randBytes[2],
						randBytes[3], randBytes[4], randBytes[5])
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

func unmountNetworkNamespace(containerID string) {
	netNsPath := getGockerNetNsPath() + "/" + containerID
	if err := syscall.Unmount(netNsPath, 0); err != nil {
		log.Fatalf("Uable to mount network namespace: %v at %s", err, netNsPath)
	}
}

func unmountContainerFs(containerID string) {
	mountedPath := getGockerContainersPath() + "/" + containerID + "/fs/mnt"
	if err := syscall.Unmount(mountedPath, 0); err != nil {
		log.Fatalf("Uable to mount container file system: %v at %s", err, mountedPath)
	}
}
func copyNameserverConfig(containerID string) error {
	return copyFile("/etc/resolv.conf",
		getContainerFSHome(containerID) + "/mnt/etc/resolv.conf")
}
/*
	Called if this program is executed with "child-mode" as the first argument
*/
func execContainerCommand(mem int, swap int, pids int, cpus float64,
		containerID string, imageShaHex string, args []string) {
	mntPath := getContainerFSHome(containerID) + "/mnt"
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	imgConfig := parseContainerConfig(imageShaHex)
	doOrDieWithMsg(syscall.Sethostname([]byte(containerID)), "Unable to set hostname")
	doOrDieWithMsg(joinContainerNetworkNamespace(containerID), "Unable to join container network namespace")
	createCGroups(containerID, true)
	configureCGroups(containerID, mem, swap, pids, cpus)
	doOrDieWithMsg(copyNameserverConfig(containerID), "Unable to copy resolve.conf")
	doOrDieWithMsg(syscall.Chroot(mntPath), "Unable to chroot")
	doOrDieWithMsg(os.Chdir("/"), "Unable to change directory")
	createDirsIfDontExist([]string{"/proc"})
	doOrDieWithMsg(syscall.Mount("proc", "/proc", "proc", 0, ""), "Unable to mount proc")
	doOrDieWithMsg(syscall.Mount("tmpfs", "/tmp", "tmpfs", 0, ""), "Unable to mount tmpfs")
	setupLocalInterface()
	cmd.Env = imgConfig.Config.Env
	cmd.Run()
	doOrDie(syscall.Unmount("/proc", 0))
	doOrDie(syscall.Unmount("/tmp", 0))
}

func prepareAndExecuteContainer(mem int, swap int, pids int, cpus float64,
					containerID string, imageShaHex string, cmdArgs []string) {

	/* Setup the network namespace  */
	cmd := &exec.Cmd{
		Path: "/proc/self/exe",
		Args: []string{"/proc/self/exe", "setup-netns", containerID},
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	cmd.Run()

	/* Namespace the virtual interface  */
	cmd = &exec.Cmd{
		Path: "/proc/self/exe",
		Args: []string{"/proc/self/exe", "fence-veth", containerID},
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	cmd.Run()

	/* Setup the virtual interface  */
	cmd = &exec.Cmd{
		Path: "/proc/self/exe",
		Args: []string{"/proc/self/exe", "setup-veth", containerID},
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	cmd.Run()
	/*
	From namespaces(7)
	       Namespace Flag            Isolates
	       --------- ----   		 --------
	       Cgroup    CLONE_NEWCGROUP Cgroup root directory
	       IPC       CLONE_NEWIPC    System V IPC,
	                                 POSIX message queues
	       Network   CLONE_NEWNET    Network devices,
	                                 stacks, ports, etc.
	       Mount     CLONE_NEWNS     Mount points
	       PID       CLONE_NEWPID    Process IDs
	       Time      CLONE_NEWTIME   Boot and monotonic
	                                 clocks
	       User      CLONE_NEWUSER   User and group IDs
	       UTS       CLONE_NEWUTS    Hostname and NIS
	                                 domain name
	*/
	fmt.Println("building child args:", mem, pids, cpus)
	var opts []string
	if mem > 0 {
		opts = append(opts, "--mem=" + strconv.Itoa(mem))
	}
	if swap >= 0 {
		opts = append(opts, "--swap=" + strconv.Itoa(swap))
	}
	if pids > 0 {
		opts = append(opts, "--pids=" + strconv.Itoa(pids))
	}
	if cpus > 0 {
		opts = append(opts, "--cpus=" + strconv.FormatFloat(cpus, 'f', 1, 64))
	}
	opts = append(opts, "--img=" + imageShaHex)
	fmt.Println("args:", opts)
	args := append([]string{containerID}, cmdArgs...)
	args = append(opts, args...)
	args = append([]string{"child-mode"}, args...)
	fmt.Println("args:", args)
	cmd = exec.Command("/proc/self/exe", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: 	syscall.CLONE_NEWPID |
						syscall.CLONE_NEWNS |
						syscall.CLONE_NEWUTS |
						syscall.CLONE_NEWIPC,
		Unshareflags: syscall.CLONE_NEWNS,
	}
	doOrDie(cmd.Run())
}

func initContainer(mem int, swap int, pids int, cpus float64, src string, args []string)  {
	containerID := createContainerID()
	log.Printf("New container ID: %s\n", containerID)
	imageShaHex := downloadImageIfRequired(src)
	log.Printf("Image to overlay mount: %s\n", imageShaHex)
	createContainerDirectories(containerID)
	mountOverlayFileSystem(containerID, imageShaHex)
	if err := setupVirtualEthOnHost(containerID); err != nil {
		log.Fatalf("Unable to setup Veth0 on host: %v", err)
	}
	prepareAndExecuteContainer(mem, swap, pids, cpus, containerID, imageShaHex, args)
	log.Printf("Container done.\n")
	unmountNetworkNamespace(containerID)
	unmountContainerFs(containerID)
	removeCGroups(containerID)
}
