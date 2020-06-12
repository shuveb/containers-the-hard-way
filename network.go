package main

import (
	"fmt"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
	"log"
	"math/rand"
	"net"
	"syscall"
)

func createMACAddress() net.HardwareAddr {
	hw := make(net.HardwareAddr, 6)
	hw[0] = 0x02
	hw[1] = 0x42
	rand.Read(hw[2:])
	return hw
}

func createIPAddress() string {
	byte1 := rand.Intn(254)
	byte2 := rand.Intn(254)
	return fmt.Sprintf("172.29.%d.%d", byte1, byte2)
}

/*
	Go through the list of interfaces and return true if the gocker0 bridge is up
*/

func isGockerBridgeUp() (bool, error) {
	if links, err := netlink.LinkList(); err != nil {
		log.Printf("Unable to get list of links.\n")
		return false, err
	} else {
		for _, link := range links {
			if link.Type() == "bridge" && link.Attrs().Name == "gocker0" {
				return true, nil
			}
		}
		return false, err
	}
}

/*
	This function sets up the "gocker0" bridge, which is our main bridge
	interface. To keep things simple, we assign the hopefully unassigned
	and obscure private IP 172.29.0.1 to it, which is from the range of
	IPs which we will also use for our containers.
*/

func setupGockerBridge() error {
	linkAttrs := netlink.NewLinkAttrs()
	linkAttrs.Name = "gocker0"
	gockerBridge := &netlink.Bridge{LinkAttrs: linkAttrs}
	if err := netlink.LinkAdd(gockerBridge); err != nil {
		return err
	}
	addr, _ := netlink.ParseAddr("172.29.0.1/16")
	netlink.AddrAdd(gockerBridge, addr)
	netlink.LinkSetUp(gockerBridge)
	return nil
}

func setupVirtualEthOnHost(containerID string) error {
	veth0 := "veth0_" + containerID[:6]
	veth1 := "veth1_" + containerID[:6]
	linkAttrs := netlink.NewLinkAttrs()
	linkAttrs.Name = veth0
	veth0Struct := &netlink.Veth{
		LinkAttrs:        linkAttrs,
		PeerName:         veth1,
		PeerHardwareAddr: createMACAddress(),
	}
	if err := netlink.LinkAdd(veth0Struct); err != nil {
		return err
	}
	netlink.LinkSetUp(veth0Struct)
	gockerBridge, _ := netlink.LinkByName("gocker0")
	netlink.LinkSetMaster(veth0Struct, gockerBridge)

	return nil
}

func setupContainerNetworkInterfaceStep1(containerID string) {
	nsMount := getGockerNetNsPath()  + "/" + containerID

	fd, err := syscall.Open(nsMount, syscall.O_RDONLY, 0)
	defer syscall.Close(fd)
	if err != nil {
		log.Fatalf("Unable to open: %v\n", err)
	}
	/* Set veth1 of the new container to the new network namespace */
	veth1 := "veth1_" + containerID[:6]
	veth1Link, err := netlink.LinkByName(veth1)
	if err != nil {
		log.Fatalf("Unable to fetch veth1: %v\n", err)
	}
	if err := netlink.LinkSetNsFd(veth1Link, fd); err != nil {
		log.Fatalf("Unable to set network namespace for veth1: %v\n", err)
	}
}

func setupContainerNetworkInterfaceStep2(containerID string) {
	nsMount := getGockerNetNsPath()  + "/" + containerID
	fd, err := syscall.Open(nsMount, syscall.O_RDONLY, 0)
	defer syscall.Close(fd)
	if err != nil {
		log.Fatalf("Unable to open: %v\n", err)
	}
	if err := syscall.Unshare(syscall.CLONE_NEWNET); err !=nil {
		log.Fatalf("Unshare system call failed: %v\n", err)
	}
	if err := unix.Setns(fd, syscall.CLONE_NEWNET); err != nil {
		log.Fatalf("Setns system call failed: %v\n", err)
	}

	veth1 := "veth1_" + containerID[:6]
	veth1Link, err := netlink.LinkByName(veth1)
	if err != nil {
		log.Fatalf("Unable to fetch veth1: %v\n", err)
	}
	addr, _ := netlink.ParseAddr(createIPAddress() + "/16")
	if err := netlink.AddrAdd(veth1Link, addr); err != nil {
		log.Fatalf("Error assigning IP to veth1: %v\n", err)
	}

	/* Bring up the interface */
	doOrDieWithMsg(netlink.LinkSetUp(veth1Link), "Unable to bring up veth1")

	/* Add a default route */
	route := netlink.Route{
		Scope: netlink.SCOPE_UNIVERSE,
		LinkIndex: veth1Link.Attrs().Index,
		Gw: net.ParseIP("172.29.0.1"),
		Dst: nil,
	}
	doOrDieWithMsg(netlink.RouteAdd(&route), "Unable to add default route")
}

/*
	This is the function that sets the IP address for the local interface.
	There seems to be a bug in the netlink library in that it does not
	succeed in looking up the local interface by name, always returning an
	error. As a workaround, we loop through the interfaces, compare the name,
	set the IP and make the interface up.

*/

func setupLocalInterface() {
	links, _ := netlink.LinkList()
	for _, link := range links {
		if link.Attrs().Name == "lo" {
			loAddr, _ := netlink.ParseAddr("127.0.0.1/32")
			if err := netlink.AddrAdd(link, loAddr); err != nil {
				log.Println("Unable to configure local interface!")
			}
			netlink.LinkSetUp(link)
		}
	}
}

func setupNewNetworkNamespace(containerID string) {
	_ = createDirsIfDontExist([]string{getGockerNetNsPath()})
	nsMount := getGockerNetNsPath()  + "/" + containerID
	if _, err := syscall.Open(nsMount, syscall.O_RDONLY|syscall.O_CREAT|syscall.O_EXCL, 0644); err != nil {
		log.Fatalf("Unable to open bind mount file: :%v\n", err)
	}

	fd, err := syscall.Open("/proc/self/ns/net", syscall.O_RDONLY, 0)
	defer syscall.Close(fd)
	if err != nil {
		log.Fatalf("Unable to open: %v\n", err)
	}

	if err := syscall.Unshare(syscall.CLONE_NEWNET); err !=nil {
		log.Fatalf("Unshare system call failed: %v\n", err)
	}
	if err := syscall.Mount("/proc/self/ns/net", nsMount, "bind", syscall.MS_BIND, ""); err != nil {
		log.Fatalf("Mount system call failed: %v\n", err)
	}
	if err := unix.Setns(fd, syscall.CLONE_NEWNET); err != nil {
		log.Fatalf("Setns system call failed: %v\n", err)
	}
}

func joinContainerNetworkNamespace(containerID string) error {
	nsMount := getGockerNetNsPath()  + "/" + containerID
	fd, err := syscall.Open(nsMount, syscall.O_RDONLY, 0)
	if err != nil {
		log.Printf("Unable to open: %v\n", err)
		return err
	}
	if err := syscall.Unshare(syscall.CLONE_NEWNET); err !=nil {
		log.Printf("Unshare system call failed: %v\n", err)
		return err
	}
	if err := unix.Setns(fd, syscall.CLONE_NEWNET); err != nil {
		log.Printf("Setns system call failed: %v\n", err)
		return err
	}
	return nil
}