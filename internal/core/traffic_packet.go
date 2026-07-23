package core

import (
	"github.com/sagernet/sing/common/buf"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
)

func (tc *trafficConnInternal) wrapPacket(conn N.PacketConn) N.PacketConn {
	tc.closer = conn.Close
	return &wrappedPacketConn{
		PacketConn: conn,
		onRead: func(n int) {
			tc.onTraffic(0, int64(n))
		},
		onWrite: func(n int) {
			tc.onTraffic(int64(n), 0)
		},
		onClose: func() {
			tc.tracker.connections.Delete(tc.id)
		},
	}
}

type wrappedPacketConn struct {
	N.PacketConn
	onRead  func(int)
	onWrite func(int)
	onClose func()
}

func (w *wrappedPacketConn) ReadPacket(buffer *buf.Buffer) (M.Socksaddr, error) {
	destination, err := w.PacketConn.ReadPacket(buffer)
	if buffer != nil && buffer.Len() > 0 {
		w.onRead(buffer.Len())
	}
	return destination, err
}

func (w *wrappedPacketConn) WritePacket(buffer *buf.Buffer, destination M.Socksaddr) error {
	n := 0
	if buffer != nil {
		n = buffer.Len()
	}
	err := w.PacketConn.WritePacket(buffer, destination)
	if err == nil && n > 0 {
		w.onWrite(n)
	}
	return err
}

func (w *wrappedPacketConn) Close() error {
	w.onClose()
	return w.PacketConn.Close()
}
