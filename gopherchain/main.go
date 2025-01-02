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

	"github.com/google/nftables"
	"github.com/google/nftables/binaryutil"
	"github.com/google/nftables/expr"
	"github.com/jamiealquiza/envy"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
	"golang.org/x/sys/unix"
)

var (
	device   = flag.String("device", "gopherchaintun0", "TUN device name")
	proxy    = flag.String("proxy", "socks5://10.1.1.1:1080", "Proxy address. can't be localhost")
	ipMask   = flag.String("ipmask", "100.200.200.1/32", "IP address of the TUN device")
	nsName   = flag.String("nsname", "gopherchain", "Name of the new network namespace")
	magicdns = flag.String("magicdns", "", `if a dns server value is specified,
	starts a local dns server and forwards all traffic to udp53 to this dns server
  - udp://1.1.1.1:53
  - tcp://9.9.9.9:5353
  - https://dns.adguard.com
  - quic://dns.adguard.com:8853
  - tcp-tls://dns.adguard.com:853`)
	logLevel = flag.String("loglevel", "debug", "Log level")
)

var lvl = new(slog.LevelVar)
var log = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
	AddSource: true,
	Level:     lvl,
}))

const (
	ROUTE_TABLE = 500
	FWMARK      = 555
)

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
	_, ipnet, _ := net.ParseCIDR(*ipMask)
	if err := netlink.AddrAdd(link, &netlink.Addr{IPNet: ipnet}); err != nil {
		log.Error("failed to set IP address of TUN device",
			"error", err)
		return
	}

	// add 127.0.0.1 to the loopback interface
	lo, err := netlink.LinkByName("lo")
	if err != nil {
		log.Error("failed to get loopback interface",
			"error", err)
		return
	}
	if err := netlink.AddrAdd(lo, &netlink.Addr{IPNet: &net.IPNet{IP: net.IPv4(127, 0, 0, 1), Mask: net.CIDRMask(8, 32)}}); err != nil {
		log.Error("failed to add localhost to loopback interface",
			"error", err)
		return
	}

	// turn on the TUN device
	if err := netlink.LinkSetUp(link); err != nil {
		log.Error("failed to turn on TUN device",
			"error", err)
		return
	}
	// turn on loopback
	if err := netlink.LinkSetUp(lo); err != nil {
		log.Error("failed to turn on loopback interface",
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
		WithMark(FWMARK),
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
	//ip route add default dev "$TUN" table "$TABLE"
	if err := netlink.RouteAdd(&netlink.Route{
		LinkIndex: link.Attrs().Index,
		Dst:       &net.IPNet{IP: net.IPv4(0, 0, 0, 0), Mask: net.CIDRMask(0, 0)},
		Table:     ROUTE_TABLE,
	}); err != nil {
		log.Error("failed to add route to TUN device",
			"error", err)
		return

	}
	// # policy routing (https://github.com/xjasonlyu/tun2socks/blob/main/docker/entrypoint.sh)
	// ip rule add not fwmark "01" table unix.RT_TABLE_MAIN
	rule1 := netlink.NewRule()
	rule1.Invert = true
	rule1.Table = ROUTE_TABLE
	rule1.Mark = FWMARK
	if err := netlink.RuleAdd(rule1); err != nil {
		log.Error("failed to add rule to TUN device",
			"error", err)
		return
	}

	// ip rule add fwmark "01" to "100.200.200.1/32" prohibit
	rule2 := netlink.NewRule()
	rule2.Mark = FWMARK
	rule2.Dst = ipnet
	rule2.Type = unix.RTN_PROHIBIT
	if err := netlink.RuleAdd(rule2); err != nil {
		log.Error("failed to add rule to TUN device",
			"error", err)
		return
	}

	// add a route to loopback for 127.0.0.1/8
	// if err := netlink.RouteAdd(&netlink.Route{
	// 	LinkIndex: lo.Attrs().Index,
	// 	Scope:     netlink.SCOPE_SITE,
	// 	Dst:       &net.IPNet{IP: net.IPv4(127, 0, 0, 1), Mask: net.CIDRMask(8, 32)},
	// }); err != nil {
	// 	log.Error("failed to add route to loopback",
	// 		"error", err)
	// 	return
	// }

	// delete the IPv6 address of the TUN device
	if err := deleteipv6(link); err != nil {
		log.Error("failed to delete IPv6 address of TUN device",
			"error", err)
		return
	}

	// if a dns server is specified, start a local dns server
	if *magicdns != "" {
		if err := setupMagicDNS(newns, *ipnet, *magicdns, false); err != nil {
			log.Error("failed to setup magic dns",
				"error", err)
			return
		}
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

func setupMagicDNS(link netns.NsHandle, ipnet net.IPNet, dns string, skipTLSVerifiction bool) error {
	// start a new resolver based on the dns string
	res, err := NewDNSClient(dns, skipTLSVerifiction, "")
	if err != nil {
		return err
	}

	// create a new dns server and grab the port
	port := ServeDNS(link, &res)

	// iptables -t nat -A PREROUTING -p udp --dport 53 -j DNAT --to 127.0.0.1:5555;
	// iptables -t nat -A POSTROUTING -p udp -d 127.0.0.1 --dport port -o DEVICE -j MASQUERADE;

	// Configure nftables rules
	conn, err := nftables.New(nftables.WithNetNSFd(int(link)))
	if err != nil {
		return fmt.Errorf("failed to create nftables connection: %v", err)
	}

	// Create nat table if it doesn't exist
	nat := conn.AddTable(&nftables.Table{
		Family: nftables.TableFamilyIPv4,
		Name:   "nat",
	})

	// Create postrouting chain
	post := conn.AddChain(&nftables.Chain{
		Name:     "postrouting",
		Table:    nat,
		Type:     nftables.ChainTypeNAT,
		Hooknum:  nftables.ChainHookPostrouting,
		Priority: nftables.ChainPriorityNATSource,
	})

	// Add postrouting rule (MASQUERADE)
	conn.AddRule(&nftables.Rule{
		Table: nat,
		Chain: post,
		Exprs: []expr.Any{
			// Add counter
			&expr.Counter{},
			// Match UDP
			&expr.Meta{Key: expr.MetaKeyL4PROTO, Register: 1},
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     []byte{unix.IPPROTO_UDP},
			},
			// Match destination IP 127.0.0.1
			&expr.Payload{
				DestRegister: 1,
				Base:         expr.PayloadBaseNetworkHeader,
				Offset:       16, // IPv4 destination address offset
				Len:          4,
			},
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     []byte{127, 0, 0, 1},
			},
			// Match destination port
			&expr.Payload{
				DestRegister: 1,
				Base:         expr.PayloadBaseTransportHeader,
				Offset:       2,
				Len:          2,
			},
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     binaryutil.BigEndian.PutUint16(uint16(port)),
			},
			// Perform MASQUERADE
			&expr.Masq{},
		},
	})

	chain := conn.AddChain(&nftables.Chain{
		Name:     "output",
		Table:    nat,
		Type:     nftables.ChainTypeNAT,
		Hooknum:  nftables.ChainHookOutput,
		Priority: nftables.ChainPriorityFilter,
	})
	// add a rule to redirect 53 to the local dns server
	conn.AddRule(&nftables.Rule{
		Table: nat,
		Chain: chain,
		Exprs: []expr.Any{
			// Add counter
			&expr.Counter{},
			// Match UDP
			&expr.Meta{Key: expr.MetaKeyL4PROTO, Register: 1},
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     []byte{unix.IPPROTO_UDP},
			},
			// Match destination port 53
			&expr.Payload{
				DestRegister: 1,
				Base:         expr.PayloadBaseTransportHeader,
				Offset:       2,
				Len:          2,
			},
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     binaryutil.BigEndian.PutUint16(uint16(53)),
			},
			// Redirect to 127.0.0.1:53
			&expr.Immediate{
				Register: 1,
				Data:     []byte{127, 0, 0, 1},
			},
			&expr.Immediate{
				Register: 2,
				Data:     binaryutil.BigEndian.PutUint16(uint16(port)),
			},
			&expr.Redir{
				RegisterProtoMin: 2,
			},
		},
	})

	if err := conn.Flush(); err != nil {
		return fmt.Errorf("failed to apply nftables rules: %v", err)
	}

	return nil
}
