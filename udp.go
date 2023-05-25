// Copyright (C) 2020 Storj Labs, Inc.
// See LICENSE for copying information.

package jaeger

import (
	"context"
	"go.uber.org/zap"
	"net"
)

type UDPTransport struct {
	conn *net.UDPConn
	log  *zap.Logger
}

var _ Transport = &UDPTransport{}

func OpenUDPTransport(ctx context.Context, log *zap.Logger, agentAddr string, maxPacketSize int) (*UDPTransport, error) {
	var err error

	dialer := net.Dialer{}
	conn, err := dialer.DialContext(ctx, "udp", agentAddr)
	if err != nil {
		log.Debug("failed open  UDP connection to Jaeger", zap.Error(err))
		return nil, err
	}

	udpConn, ok := conn.(*net.UDPConn)
	if !ok {
		log.Debug("Connection type mismatch", zap.Error(err))
		return nil, err
	}

	if err := udpConn.SetWriteBuffer(maxPacketSize); err != nil {
		log.Debug("failed to set max packet size on Jaeger UDP connection", zap.Error(err), zap.Int("maxPacketSize", maxPacketSize))
		return nil, err
	}
	return &UDPTransport{
		log:  log,
		conn: udpConn,
	}, nil

}

func (u *UDPTransport) Write(bytes []byte) error {
	_, err := u.conn.Write(bytes)
	return err
}

func (u *UDPTransport) Close() {
	err := u.conn.Close()
	if err != nil {
		u.log.Debug("failed to close Jaeger UDP connection", zap.Error(err))
	}
}
