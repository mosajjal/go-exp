package main

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/ishidawataru/sctp"
	cmap "github.com/orcaman/concurrent-map/v2"
)

type stream struct {
	Info  *sctp.SndRcvInfo
	c     *sctp.SCTPConn
	In    chan []byte
	ctx   context.Context
	ctxC  context.CancelFunc
	RWMux *sync.RWMutex
}

// implement io.Reader and io.Writer
func (s stream) Read(p []byte) (n int, err error) {
	b := <-s.In
	if b == nil {
		return 0, fmt.Errorf("nil")
	}
	// if len(b) == 1 && b[0] == 0xff {
	// 	return 0, io.EOF
	// }
	n = copy(p, b)
	return
}

func (s stream) Write(p []byte) (n int, err error) {
	s.RWMux.Lock()
	defer s.RWMux.Unlock()
	n, err = s.c.SCTPWrite(p, s.Info)
	return
}

func (s *stream) SetEnd() (n int, err error) {
	newInfo := *s.Info
	newInfo.PPID = pid
	s.Info = &newInfo
	// n, err = s.c.SCTPWrite([]byte{0xff}, &newInfo)
	return
}

type tunnel struct {
	C       *sctp.SCTPConn
	streams cmap.ConcurrentMap[string, stream]
}

func (t *tunnel) CreateStream(streamID uint16) {
	ctx, ctxC := context.WithCancel(context.Background())
	t.streams.Set(fmt.Sprintf("%d", streamID), stream{
		Info:  &sctp.SndRcvInfo{Stream: streamID, PPID: 0},
		c:     t.C,
		In:    make(chan []byte, 1024),
		ctx:   ctx,
		ctxC:  ctxC,
		RWMux: &sync.RWMutex{},
	})
}

func (t *tunnel) GetStream(streamID uint16) (stream, bool) {
	s, ok := t.streams.Get(fmt.Sprintf("%d", streamID))
	if !ok {
		return stream{}, false
	}
	return s, true
}

func (t *tunnel) CreateIfNotExistStream(streamID uint16) (stream, bool) {
	s, ok := t.GetStream(streamID)
	if !ok {
		t.CreateStream(streamID)
		s, _ = t.GetStream(streamID)
	}
	return s, ok
}

func (t *tunnel) SendToStream(streamID uint16, b []byte) {
	s, ok := t.GetStream(streamID)
	if !ok {
		return
	}
	s.In <- b

}

func pipe(ctx context.Context, conn1 io.ReadWriter, conn2 io.ReadWriter) {
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
			// fmt.Printf("wrote %x to %+v\n", b1, conn2) //TODO:remove
		case b2 := <-chan2:
			if b2 == nil {
				return
			}
			if _, err := conn1.Write(b2); err != nil {
				return
			}
			// fmt.Printf("wrote %x to %+v\n", b2, conn1) //TODO:remove

		case <-ctx.Done():
			return
		}
	}
}

func getChannel(conn io.ReadWriter) chan []byte {
	c := make(chan []byte)
	go func() {
		b := make([]byte, 2048)
		for {
			n, err := conn.Read(b)
			// fmt.Printf("read %x from %+v\n", b[:n], conn) //TODO:remove
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
