package main

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/ishidawataru/sctp"
	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/spf13/cobra"
	"golang.org/x/exp/rand"
)

func runClient(cmd *cobra.Command) {

	// write this based on the server
	raddrStr, portStr, err := net.SplitHostPort(cmd.Flag("upstream").Value.String())
	if err != nil {
		log.Fatalf("failed to split host port: %v", err)
	}
	raddr, err := net.ResolveIPAddr("ip", raddrStr)
	if err != nil {
		log.Fatalf("failed to resolve ip addr: %v", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		log.Fatalf("failed to convert port to int: %v", err)
	}

	laddr, err := net.ResolveIPAddr("ip", cmd.Flag("localaddr").Value.String())
	if err != nil {
		log.Fatalf("failed to resolve local address: %v", err)
	}
	// BUG: handle retry and timeout
	connSCTP, err := sctp.DialSCTP("sctp", &sctp.SCTPAddr{IPAddrs: []net.IPAddr{*laddr}, Port: 0}, &sctp.SCTPAddr{IPAddrs: []net.IPAddr{*raddr}, Port: port})
	if err != nil {
		log.Fatalf("failed to dial: %v", err)
	}
	defer connSCTP.Close()
	err = connSCTP.SubscribeEvents(sctp.SCTP_EVENT_DATA_IO)
	if err != nil {
		log.Printf("failed to subscribe to events: %v", err)
	}

	l, err := net.Listen("tcp", cmd.Flag("server").Value.String())
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	} else {
		log.Printf("listening on %s", cmd.Flag("server").Value.String())
	}
	T := tunnel{
		C:       connSCTP,
		streams: cmap.New[stream](),
	}

	go func() {
		// read from the server, find the streamID and send it to the right TCP connection
		for {
			var buf = make([]byte, 2048)
			n, info, err := connSCTP.SCTPRead(buf[:])
			if err != nil {
				connSCTP.Close()
				return
			}
			if info == nil || n == 0 {
				fmt.Printf("no info with %d bytes\n", n)
				time.Sleep(1 * time.Second)
				continue
			}

			if info.PPID == pid {
				// T.SendToStream(info.Stream, []byte{})
			} else {
				T.SendToStream(info.Stream, buf[:n])
			}

		}
	}()

	for {
		connTCP, err := l.Accept()
		if err != nil {
			log.Fatalf("failed to accept: %v", err)
		}

		var randomSID uint16
		for {
			randomSID = uint16(rand.Uint32())
			if _, ok := T.GetStream(randomSID); !ok {
				break
			}
		}
		s, _ := T.CreateIfNotExistStream(randomSID)

		go tcpHandler(s, connTCP, connSCTP)
	}

}

func tcpHandler(s stream, connTCP net.Conn, connSCTP *sctp.SCTPConn) {
	fmt.Printf("accepted connection from %s. assigning SID %d \n", connTCP.RemoteAddr().String(), s.Info.Stream)
	// activeStreams.Store(randomSID, s)
	t := time.Now()
	// BUG: upload on TG not working
	pipe(s.ctx, s, connTCP)
	fmt.Printf("connection from %s closed. took %s\n", connTCP.RemoteAddr().String(), time.Since(t))
	if _, err := s.SetEnd(); err != nil {
		log.Printf("failed to send end of stream: %v", err)
	}
	connTCP.Close()
}
