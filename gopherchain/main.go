package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/jamiealquiza/envy"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

var (
	device   = flag.String("device", "gopherchaintun0", "TUN device name")
	proxy    = flag.String("proxy", "socks5://10.1.1.1:1080", "Proxy address. can't be localhost")
	ipMask   = flag.String("ipmask", "100.200.200.1/32", "IP address of the TUN device")
	nsName   = flag.String("nsname", "gopherchain", "Name of the new network namespace")
	logLevel = flag.String("loglevel", "debug", "Log level")
)

var lvl = new(slog.LevelVar)
var log = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
	AddSource: true,
	Level:     lvl,
}))

// deleteipv6 deletes the IPv6 address of the link
func deleteipv6(link netlink.Link) error {
	addrs, err := netlink.AddrList(link, netlink.FAMILY_V6)
	if err != nil {
		return err
	}
	for _, addr := range addrs {
		if err := netlink.AddrDel(link, &addr); err != nil {
			return err
		}
	}
	return nil
}

func main() {

	envy.Parse("GOPHERCHAIN") // Expose environment variables.
	flag.Parse()

	// Set the log level
	if err := lvl.UnmarshalText([]byte(*logLevel)); err != nil {
		panic(err)
	}

	// Lock the OS Thread so we don't accidentally switch namespaces
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Create a new network namespace
	newns, err := netns.NewNamed(*nsName)
	if err != nil {
		log.Error("failed to create new network namespace",
			"error", err)
		return
	}
	defer netns.DeleteNamed(*nsName)
	defer newns.Close()

	// create a new tuntap device
	err = netlink.LinkAdd(&netlink.Tuntap{Mode: netlink.TUNTAP_MODE_TUN,
		LinkAttrs: netlink.LinkAttrs{
			Name:    *device,
			NetNsID: int(newns),
		},
	})
	if err != nil {
		log.Error("failed to create TUN device",
			"error", err)
		return
	}

	// get the link id of the TUN device
	link, err := netlink.LinkByName(*device)
	if err != nil {
		log.Error("failed to get TUN device",
			"error", err)
		return
	}

	// set the IP address of the TUN device
	ip, ipnet, _ := net.ParseCIDR(*ipMask)
	if err := netlink.AddrAdd(link, &netlink.Addr{IPNet: ipnet}); err != nil {
		log.Error("failed to set IP address of TUN device",
			"error", err)
		return
	}

	// turn on the TUN device
	if err := netlink.LinkSetUp(link); err != nil {
		log.Error("failed to turn on TUN device",
			"error", err)
		return
	}

	TUNReady := make(chan struct{})
	// create a new TUN device with Tor as upstream, and a random name
	// since this is blocking, we need to run it in a goroutine
	go NewTUN(context.Background(), TUNReady, newns,
		// WithInterface(Device),
		WithDevice(*device),
		WithProxy(*proxy),
	)
	// wait for the TUN device to be created
	<-TUNReady

	// delete the default route in the new network namespace
	// so that all traffic will be routed to the TUN device
	routes, err := netlink.RouteList(link, netlink.FAMILY_ALL)
	if err != nil {
		log.Error("failed to list routes in new network namespace",
			"error", err)
		return
	}
	for _, r := range routes {
		if err := netlink.RouteDel(&r); err != nil {
			log.Error("failed to delete route in new network namespace",
				"error", err)
			return
		}
	}

	// route all traffic of the namespace to the TUN device
	if err := netlink.RouteAdd(&netlink.Route{
		LinkIndex: link.Attrs().Index,
		Scope:     netlink.SCOPE_UNIVERSE,
		Gw:        ip,
	}); err != nil {
		log.Error("failed to add route to TUN device",
			"error", err)
		time.Sleep(20 * time.Second)
		return
	}

	// delete the IPv6 address of the TUN device
	if err := deleteipv6(link); err != nil {
		log.Error("failed to delete IPv6 address of TUN device",
			"error", err)
		return
	}

	info := fmt.Sprintf(`** Gopherchain has started **

A tun device is created and moved to the new network namespace
you have several options to interact with the new network namespace

1) Use the nsenter command. replace ip link show with any command, or bash to put the entire bash session in the new network namespace

$ sudo nsenter -t %d -n ip link show

2) use the named namespace path
	
$ sudo nsenter --net=/run/netns/%s ip link show

3) use the ip netns command

$ sudo ip netns exec gopherchain ip link show

Press Ctrl+C to delete the new network namespace and exit`, os.Getpid(), *nsName)

	// Create a table writer
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.Style().Box = table.StyleBoxBold
	t.AppendRow(table.Row{info})
	t.Render()

	// add signal handler to delete the namespace and wait forever
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	log.Info("waiting for signal to delete the new network namespace")

}
