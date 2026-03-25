package boxapi

import (
	"context"
	"net"

	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/common/dialer"
	"github.com/sagernet/sing/common/bufio"
	"github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
)

func currentTracker(boxInstance *box.Box) adapter.ConnectionTracker {
	if boxInstance == nil {
		return nil
	}
	router := boxInstance.Router()
	if router == nil {
		return nil
	}
	if server := router.V2RayServer(); server != nil {
		return server.StatsService()
	}
	return nil
}

func defaultOutboundTag(boxInstance *box.Box) string {
	if boxInstance == nil || boxInstance.Outbound() == nil || boxInstance.Outbound().Default() == nil {
		return ""
	}
	return boxInstance.Outbound().Default().Tag()
}

func defaultOutbound(boxInstance *box.Box) adapter.Outbound {
	if boxInstance == nil || boxInstance.Outbound() == nil {
		return nil
	}
	return boxInstance.Outbound().Default()
}

func DialContext(ctx context.Context, boxInstance *box.Box, network, addr string) (net.Conn, error) {
	defOutboundTag := defaultOutboundTag(boxInstance)
	conn, err := dialer.NewDetour(boxInstance.Outbound(), defOutboundTag, true).DialContext(ctx, network, metadata.ParseSocksaddr(addr))
	if err != nil {
		return nil, err
	}
	tracker := currentTracker(boxInstance)
	if ss, ok := tracker.(*SbStatsService); ok {
		conn = ss.RoutedConnectionInternal("", defOutboundTag, "", conn, false)
	}
	return conn, nil
}

func DialUDP(ctx context.Context, boxInstance *box.Box) (net.PacketConn, error) {
	defOutboundTag := defaultOutboundTag(boxInstance)
	packetConn, err := dialer.NewDetour(boxInstance.Outbound(), defOutboundTag, true).ListenPacket(ctx, metadata.Socksaddr{})
	if err != nil {
		return nil, err
	}
	routedPacketConn := N.PacketConn(bufio.NewPacketConn(packetConn))
	tracker := currentTracker(boxInstance)
	if ss, ok := tracker.(*SbStatsService); ok {
		outbound := defaultOutbound(boxInstance)
		if outbound != nil {
			routedPacketConn = ss.RoutedPacketConnection(ctx, routedPacketConn, adapter.InboundContext{}, nil, outbound)
		}
	}
	return bufio.NewNetPacketConn(routedPacketConn), nil
}
