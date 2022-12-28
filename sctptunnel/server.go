package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/ishidawataru/sctp"
	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/spf13/cobra"
)

var pid = binary.BigEndian.Uint32([]byte{0, 0, 0, 39})

func handleTCPInsideTunnel(s stream, remoteAddr string) {
	remoteConn, err := net.DialTimeout("tcp", remoteAddr, 5*time.Second)
	if err != nil {
		log.Printf("failed to dial: %v", err)
		if _, err := s.SetEnd(); err != nil {
			log.Printf("failed to send end of stream: %v", err)
		}
		if _, err := io.Copy(io.Discard, s); err != nil {
			log.Printf("failed to discard stream: %v", err)
		}
		return
	}
	log.Printf("dial to %s from source port %s", remoteConn.RemoteAddr(), remoteConn.LocalAddr())

	pipe(s.ctx, s, remoteConn)
	if err := remoteConn.Close(); err != nil {
		log.Printf("failed to close remote connection: %v", err)
	}
}

// handle a single SCTP connection which can result in 56k TCP connections
func handleTunnel(T tunnel, remoteAddr string) {
	// each connection comes with a stream ID that identifies the TCP connection
	// after reading a few bytes, we can determine the stream ID
	var buf [2048 + 256]byte
	var n int
	var info *sctp.SndRcvInfo
	var err error
	for {

		n, info, err = T.C.SCTPRead(buf[:])
		if err != nil {
			T.C.Close()
			return
		}
		if info == nil || n == 0 {
			fmt.Printf("no info with %d bytes\n", n)
			time.Sleep(1 * time.Second)
			continue
		}

		s, existed := T.CreateIfNotExistStream(info.Stream)
		if !existed {
			go handleTCPInsideTunnel(s, remoteAddr)
		}
		if info.PPID == pid {
			go func() {
				time.Sleep(3 * time.Second)
				s.ctxC()
			}()
			//TODO: find a way to cancel the above goroutine when this happens, and dispose the byte
		}
		T.SendToStream(info.Stream, buf[:n])

	}
}

func runServer(cmd *cobra.Command) {
	laddr, port, err := net.SplitHostPort(cmd.Flag("server").Value.String())
	if err != nil {
		log.Fatalf("failed to parse server address: %v", err)
	}
	laddrIP, err := net.ResolveIPAddr("ip", laddr)
	if err != nil {
		log.Fatalf("failed to resolve server address: %v", err)
	}
	portInt, err := strconv.Atoi(port)
	if err != nil {
		log.Fatalf("failed to parse server port: %v", err)
	}
	ln, err := sctp.ListenSCTP("sctp", &sctp.SCTPAddr{IPAddrs: []net.IPAddr{*laddrIP}, Port: portInt})
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	log.Printf("Listening on %s:%d", laddrIP, portInt)

	for {
		conn, err := ln.AcceptSCTP()
		if err != nil {
			log.Fatalf("failed to accept: %v", err)
		} else {
			log.Printf("Accepted Connection from %s", conn.RemoteAddr())
		}

		err = conn.SubscribeEvents(sctp.SCTP_EVENT_DATA_IO | sctp.SCTP_EVENT_PEER_ERROR)
		if err != nil {
			log.Printf("failed to subscribe to events: %v", err)
		}

		// get upstream and pass it on
		upstream := cmd.Flag("upstream").Value.String()
		T := tunnel{
			C:       conn,
			streams: cmap.New[stream](),
			// streams: make(map[uint16]stream),
		}
		go handleTunnel(T, upstream)

	}

}
