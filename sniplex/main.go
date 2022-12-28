// this app can not deal with the huge number of TCP_TIMEWAIT connections. in order to reuse those connections
// use net.ipv4.tcp_tw_reuse
// https://fromdual.com/huge-amount-of-time-wait-connections
package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/exp/slog"
)

var sniDict = make(map[string]string)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "sniplex",
		Short: "sniplex",
		Run: func(cmd *cobra.Command, args []string) {
			// Do Stuff Here
		},
	}
	upstream := rootCmd.PersistentFlags().StringArrayP("upstream", "u", []string{}, "use it: www.google.com,142.250.66.228:443 it can also have default:127.0.0.1:4433 as default")
	bind := rootCmd.PersistentFlags().StringP("bind", "i", "0.0.0.0", "which IP to bind to")
	loglevel := rootCmd.PersistentFlags().IntP("loglevel", "v", 0, "default log level. -4:debug, 0:info, 4:warn, 8:err")
	port := rootCmd.PersistentFlags().IntP("port", "p", 443, "which port to bind to")
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Whoops. There was an error while executing your CLI '%s'", err)
		os.Exit(1)
	}
	for _, v := range *upstream {
		keys := strings.Split(v, ",")
		sniDict[keys[0]] = keys[1]
	}

	slog.SetDefault(slog.New(slog.HandlerOptions{Level: slog.Level(*loglevel)}.NewJSONHandler(os.Stderr)))

	runHTTPS(*bind, *port)

}

func runHTTPS(bind string, port int) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", bind, port))
	if err != nil {
		slog.Error("can't resolve TCP", err)
		panic(255)
	}
	l, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		slog.Error("can't listen on TCP", err)
		panic(255)
	}
	slog.Info("Started server on",
		"ip", bind,
		"port", port)
	// defer l.Close()
	// for {
	// 	c, err := l.AcceptTCP()
	// 	if err != nil {
	// 		slog.Error("error accepting new connection", err)
	// 	}
	// 	go handle443(c)
	// }

	pending, complete := make(chan *net.TCPConn), make(chan *net.TCPConn)

	for i := 0; i < 1024; i++ {
		go handleConn(pending, complete)
	}
	go closeConn(complete)

	for {
		conn, err := l.AcceptTCP()
		if err != nil {
			panic(err)
		}
		pending <- conn
	}
}

func handle443(conn *net.TCPConn) {
	incoming := make([]byte, 2048)
	n, err := conn.Read(incoming)
	if err != nil {
		slog.Error("error reading incoming connection", err)
		return
	}
	sni, err := GetHostname(incoming)
	if err != nil {
		slog.Info(err.Error())
		return
	}

	var rAddr net.IP
	var rPort int
	var s string
	if socket, ok := sniDict[sni]; ok {
		s = socket
	} else {
		if socket, ok := sniDict["default"]; ok {
			s = socket
		} else {
			slog.Info("Connection attempted to non-existing target")
			return
		}
	}
	rAddrStr, rPortStr, err := net.SplitHostPort(s)
	if err != nil {
		slog.Error("can't split host and port", err)
		return
	}
	rAddr = net.ParseIP(rAddrStr)
	rPort, err = strconv.Atoi(rPortStr)
	if err != nil {
		slog.Error("port not understood", err)
		return
	}

	// TODO: handle timeout and context here
	// if rAddr.IsLoopback() || rAddr.IsPrivate() || rAddr.Equal(net.IPv4(0, 0, 0, 0)) {
	// 	slog.Info("connection to private IP ignored")
	// 	return nil
	// }
	slog.Info("establishing connection", "remote_ip", rAddr, "host", sni)
	target, err := net.DialTCP("tcp", nil, &net.TCPAddr{IP: rAddr, Port: rPort})
	if err != nil {
		slog.Error("could not connect to target", err)
		return
	}
	defer target.Close()
	if _, err := target.Write(incoming[:n]); err != nil {
		return
	}
	pipe(conn, target)

}

func pipe(conn1 net.Conn, conn2 net.Conn) {
	chan1 := getChannel(conn1)
	chan2 := getChannel(conn2)
	for {
		select {
		case b1 := <-chan1:
			if b1 == nil {
				return
			}
			if _, err := conn2.Write(b1); err != nil {
				return
			}
		case b2 := <-chan2:
			if b2 == nil {
				return
			}
			if _, err := conn1.Write(b2); err != nil {
				return
			}
		}
	}
}

func getChannel(conn net.Conn) chan []byte {
	c := make(chan []byte)
	go func() {
		b := make([]byte, 1024)
		for {
			n, err := conn.Read(b)
			if n > 0 {
				res := make([]byte, n)
				copy(res, b[:n])
				c <- res
			}
			if err != nil {
				c <- nil
				break
			}
		}
	}()
	return c
}

func handleConn(in <-chan *net.TCPConn, out chan<- *net.TCPConn) {
	for conn := range in {
		handle443(conn)
		out <- conn
	}
}

func closeConn(in <-chan *net.TCPConn) {
	for conn := range in {
		conn.Close()
	}
}
