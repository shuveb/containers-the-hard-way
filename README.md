# Containers the hard way: Gocker: A mini Docker written in Go
It is a set of Linux's operating system primitives that provide the illusion of a container. A process or a set of processes can shed their environment or namespaces and live in new namespaces of their own, separate from the host's `default` namespace. Container management systems like Docker make it incredibly easy to manage containers on your machine. But how are these containers constructed? It is just a sequence of Linux system calls (involving namespaces and cgroups, mainly), at the very basic level while also leveraging other existing Linux technologies for container file system, networking, etc.

## What is Gocker?
Gocker is an implementation from scratch of the core functionalities of Docker in the Go programming language. The main aim here is to provide an understanding of how exactly containers work at the Linux system call level. Gocker allows you to create containers, manage container images, execute processes in existing containers, etc.

## Gocker explanation
Gocker and how it works is explained at the Linux system call level [on the Unixism blog](https://unixism.net/2020/06/containers-the-hard-way-gocker-a-mini-docker-written-in-go/). If you are interested in that level of detail, please read it.

## Why Gocker?
When I came across [bocker](https://github.com/p8952/bocker), which is Docker-like container management written system in Bash shell script, I found 2 problems with it:
* Bocker uses various Linux utilities. While you get the point, command line utilities are opaque, and you don't get to understand what they are doing at the Linux system call level. Also, a single command can sometime issue a more than one pertinent system calls.
* Boker's last commit is more than 5 years ago, and it does not work anymore. Docker Hub API changes seem to have broken it.

Gocker on the other hand is pure Go source code which allows you to see what exactly goes on at the Linux system call level. This should give you a way better understanding of how containers actually work.

 Don't get me wrong here. Bocker is still a fantastic and very creatively written tool. If you want to understand how containers work, you should still take a look at it and I'm confident you'll learn a thing or two from it, just like I did.
 
 ## Gocker capabilities
 Gocker can emulate the core of Docker, letting you manager Docker images (which it gets from Docker Hub), run containers, list running containers or execute a process in an already running container:
 * Run a process in a container
     * gocker run <--cpus=cpus-max> <--mem=mem-max> <--pids=pids-max> <image[:tag]> </path/to/command>
 * List running containers
    * gocker ps
 * Execute a process in a running container
    * gocker exec <container-id> </path/to/command>
 * List locally available images
     * gocker images
 * Remove a locally available image
     * gocker rmi <image-id>

### Other capabilities     
* Gocker uses the Ovelay file system to create containers quickly without the need to copy whole file systems while also sharing the same container image between multiple container instances.
* Gocker containers get their own networking namespace and are able to access the internet. See limitations below.
* You can control system resources like CPU percentage, the amount of RAM and the number of processes. Gocker achieves this by leveraging cgroups.
    
## Gocker container isolation
Containers created with Gocker get the following namespaces of their own (see `run.go`):
* File system (via `chroot`)
* PID 
* IPC
* UTS (hostname)
* Mount
* Network

While cgroups to limit the following are created, continers are left to use unlimited resources unless you specify the `--mem`, `--cpus` or `--pids` options to the `gocker run` command. These flags limit the maximum RAM, CPU cores and PIDs the container can consume respectively.
* Number of CPU cores
* RAM
* Number of PIDs (to limit processes)

 ## An example Gocker session
 ```
 ➜  sudo ./gocker images          
 2020/06/12 08:32:23 Cmd args: [./gocker images]
 IMAGE	             TAG	   ID
 centos
 	          latest 470671670cac
 redis
 	          latest c349430fd524
 ubuntu
 	           18.04 c3c304cb4f22
 	          latest 1d622ef86b13
➜  sudo ./gocker run alpine /bin/sh
2020/06/12 08:33:33 Cmd args: [./gocker run alpine /bin/sh]
2020/06/12 08:33:33 New container ID: 7bfe9b0f1c2e
2020/06/12 08:33:33 Downloading metadata for alpine:latest, please wait...
2020/06/12 08:33:36 imageHash: a24bb4013296
2020/06/12 08:33:36 Checking if image exists under another name...
2020/06/12 08:33:36 Image doesn't exist. Downloading...
2020/06/12 08:33:38 Successfully downloaded alpine
2020/06/12 08:33:38 Uncompressing layer to: /var/lib/gocker/images/a24bb4013296/fe8bebfdf212/fs 
2020/06/12 08:33:38 Image to overlay mount: a24bb4013296
2020/06/12 08:33:38 Cmd args: [/proc/self/exe setup-netns 7bfe9b0f1c2e]
2020/06/12 08:33:38 Cmd args: [/proc/self/exe setup-veth 7bfe9b0f1c2e]
2020/06/12 08:33:38 Cmd args: [/proc/self/exe child-mode --img=a24bb4013296 7bfe9b0f1c2e /bin/sh]
/ # ifconfig 
lo        Link encap:Local Loopback  
          inet addr:127.0.0.1  Mask:255.0.0.0
          inet6 addr: ::1/128 Scope:Host
          UP LOOPBACK RUNNING  MTU:65536  Metric:1
          RX packets:0 errors:0 dropped:0 overruns:0 frame:0
          TX packets:0 errors:0 dropped:0 overruns:0 carrier:0
          collisions:0 txqueuelen:1000 
          RX bytes:0 (0.0 B)  TX bytes:0 (0.0 B)

veth1_7bfe9b Link encap:Ethernet  HWaddr 02:42:6E:E8:FC:06  
          inet addr:172.29.41.13  Bcast:172.29.255.255  Mask:255.255.0.0
          inet6 addr: fe80::42:6eff:fee8:fc06/64 Scope:Link
          UP BROADCAST RUNNING MULTICAST  MTU:1500  Metric:1
          RX packets:22 errors:0 dropped:0 overruns:0 frame:0
          TX packets:7 errors:0 dropped:0 overruns:0 carrier:0
          collisions:0 txqueuelen:1000 
          RX bytes:2328 (2.2 KiB)  TX bytes:586 (586.0 B)

/ # ps aux
PID   USER     TIME  COMMAND
    1 root      0:00 /proc/self/exe child-mode --img=a24bb4013296 7bfe9b0f1c2e /bin/sh
    7 root      0:00 /bin/sh
    9 root      0:00 ps aux
/ # apk add python3
fetch http://dl-cdn.alpinelinux.org/alpine/v3.12/main/x86_64/APKINDEX.tar.gz
fetch http://dl-cdn.alpinelinux.org/alpine/v3.12/community/x86_64/APKINDEX.tar.gz
(1/10) Installing libbz2 (1.0.8-r1)
(2/10) Installing expat (2.2.9-r1)
(3/10) Installing libffi (3.3-r2)
(4/10) Installing gdbm (1.13-r1)
(5/10) Installing xz-libs (5.2.5-r0)
(6/10) Installing ncurses-terminfo-base (6.2_p20200523-r0)
(7/10) Installing ncurses-libs (6.2_p20200523-r0)
(8/10) Installing readline (8.0.4-r0)
(9/10) Installing sqlite-libs (3.32.1-r0)
(10/10) Installing python3 (3.8.3-r0)
Executing busybox-1.31.1-r16.trigger
OK: 53 MiB in 24 packages
/ # python3
Python 3.8.3 (default, May 15 2020, 01:53:50) 
[GCC 9.3.0] on linux
Type "help", "copyright", "credits" or "license" for more information.
>>> exit()
/ # exit
2020/06/12 08:34:34 Container done.
➜  sudo ./gocker run ubuntu /bin/bash
2020/06/12 08:35:13 Cmd args: [./gocker run ubuntu /bin/bash]
2020/06/12 08:35:13 New container ID: c7eb7bab7e4c
2020/06/12 08:35:13 Image already exists. Not downloading.
2020/06/12 08:35:13 Image to overlay mount: 1d622ef86b13
2020/06/12 08:35:13 Cmd args: [/proc/self/exe setup-netns c7eb7bab7e4c]
2020/06/12 08:35:13 Cmd args: [/proc/self/exe setup-veth c7eb7bab7e4c]
2020/06/12 08:35:13 Cmd args: [/proc/self/exe child-mode --img=1d622ef86b13 c7eb7bab7e4c /bin/bash]
root@c7eb7bab7e4c:/# 
```
[On another terminal]
```
➜  sudo ./gocker ps
[sudo] password for shuveb: 
2020/06/12 08:36:19 Cmd args: [./gocker ps]
CONTAINER ID	IMAGE		COMMAND
c7eb7bab7e4c	ubuntu:latest	/usr/bin/bash
➜  sudo ./gocker exec c7eb7bab7e4c /bin/bash
2020/06/12 08:37:15 Cmd args: [./gocker exec c7eb7bab7e4c /bin/bash]
root@c7eb7bab7e4c:/# ps aux
USER         PID %CPU %MEM    VSZ   RSS TTY      STAT START   TIME COMMAND
root           1  0.0  0.0 1153100 6132 ?        Sl   03:05   0:00 /proc/self/exe child-mode --img=1d622ef86b13 
root           8  0.0  0.0   4116  3236 ?        S+   03:05   0:00 /bin/bash
root          11  0.0  0.0   4116  3376 ?        S    03:07   0:00 /bin/bash
root          14  0.0  0.0   5888  2956 ?        R+   03:07   0:00 ps aux
root@c7eb7bab7e4c:/# 
```
## Gocker limitations
Here are some limitations I'd love to fix in a future release:

* Gocker does not currently support exposing container ports on the host. Whenever Docker containers need to expose ports on the host, Docker uses the program `docker-proxy` as a proxy to get that done. Gocker needs a similar proxy developed. While Gocker containers can access the internet today, the ability to expose ports on the host will be a great feature to have (mainly to learn how that's done).
* Gocker does not do error handling well. Should something go wrong especially when attempting to run a container, Gocker might not cleanly unmount some file systems.

## Containers accessing internet
When you run Gocker for the first time, a new bridge, `gocker0` is created. Since all container network interfaces are connected to this bridge, they can talk to each other without you having to do anything. For containers to be able to reach the internet though, you need to enable packet forwarding on the host. For this, a convenience script `enable_internet.sh` has been provided. You might need to change it to reflect the name of your internet connected interface before you run it. There are instructions in the script. After you run this, Gocker containers should be able to reach the internet and install packages, etc.

## External Go libraries used
* [GoContainerRegistry](https://github.com/google/go-containerregistry) for downloading container images from a container registry, the default being Docker Hub.
* [PFlag](https://github.com/spf13/pflag) for handling command line flags.
* [Netlink](https://github.com/vishvananda/netlink) to configure Linux network interfaces without having to get bogged down by Netlink socket programming.
* [Unix](https://golang.org/x/sys/unix) Because Unix :)

## Disclaimer
Gocker runs as root. Use at your own risk. This is my first Go program beyond a reasonable number of lines, and I'm sure there are better ways to write Go programs and there might still be a lot of bugs lingering in here. Here are some things Gocker does to your system so you know:

* It creates the `gocker0` bridge if it does not exist.
* It blindly assumes that the IP address range `172.29.*.*` is available and uses it.
* It creates various namespaces and cgroups.
* It mounts overlay file systems.

To this end, the safest way to run Gocker might be in a virtual machine.

### Distributions
I developed Gocker on my day-to-day Arch Linux based computer. I also tested Gocker on an Ubuntu 20.04 virtual machine. It works great.

## About me
My name is Shuveb Hussain and I'm the author of the Linux-focused blog [Unixism.net](https://unixism.net). You can [follow me on Twitter](https://twitter.com/shuveb) where I post tech-related content mostly focusing on Linux, performance, scalability and cloud technologies.
