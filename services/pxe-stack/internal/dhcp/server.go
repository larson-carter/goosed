package dhcp

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync/atomic"
	"time"

	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/insomniacslk/dhcp/dhcpv4/server4"

	"goosed/services/pxe-stack/internal/config"
)

func NewServer(cfg config.DHCPConfig, logger *log.Logger) (*Server, error) {
	if logger == nil {
		logger = log.Default()
	}
	h := &handler{
		cfg:       cfg,
		logger:    logger,
		leases:    make(map[string]lease),
		startIP:   cfg.RangeStart.To4(),
		endIP:     cfg.RangeEnd.To4(),
		nextIP:    cfg.RangeStart.To4(),
		leaseTime: cfg.LeaseTime,
	}
	return &Server{cfg: cfg, logger: logger, handler: h}, nil
}

func (s *Server) Run(ctx context.Context, ready *atomic.Bool) error {
	srv, err := server4.NewServer(s.cfg.Interface, nil, s.handler.handle)
	if err != nil {
		return fmt.Errorf("start listener on %s: %w", s.cfg.Interface, err)
	}
	ready.Store(true)
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve()
	}()

	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("dhcp serve: %w", err)
		}
	case <-ctx.Done():
		srv.Close()
		<-errCh
	}
	return nil
}

func (h *handler) handle(conn net.PacketConn, peer net.Addr, req *dhcpv4.DHCPv4) {
	switch req.MessageType() {
	case dhcpv4.MessageTypeDiscover:
		h.respond(conn, peer, req, dhcpv4.MessageTypeOffer)
	case dhcpv4.MessageTypeRequest:
		h.respond(conn, peer, req, dhcpv4.MessageTypeAck)
	case dhcpv4.MessageTypeRelease:
		h.release(req.ClientHWAddr.String())
	default:
		// ignore other messages
	}
}

func (h *handler) respond(conn net.PacketConn, peer net.Addr, req *dhcpv4.DHCPv4, msgType dhcpv4.MessageType) {
	mac := req.ClientHWAddr.String()
	ip := h.assign(mac)
	if ip == nil {
		h.logger.Printf("WARN no available lease for %s", mac)
		return
	}

	reply, err := dhcpv4.NewReplyFromRequest(req)
	if err != nil {
		h.logger.Printf("ERROR create reply: %v", err)
		return
	}
	reply.UpdateOption(dhcpv4.OptMessageType(msgType))
	reply.YourIPAddr = ip
	reply.ServerIPAddr = h.cfg.ServerIP
	reply.BootFileName = h.cfg.BootFilename
	reply.Options.Update(dhcpv4.OptServerIdentifier(h.cfg.ServerIP))
	reply.Options.Update(dhcpv4.OptSubnetMask(h.cfg.SubnetMask))
	reply.Options.Update(dhcpv4.OptRouter(h.cfg.Router))
	if len(h.cfg.DNSServers) > 0 {
		reply.Options.Update(dhcpv4.OptDNS(h.cfg.DNSServers...))
	}
	reply.Options.Update(dhcpv4.OptIPAddressLeaseTime(h.leaseTime))
	if h.cfg.NextServer != nil {
		reply.ServerIPAddr = h.cfg.NextServer
		reply.Options.Update(dhcpv4.OptTFTPServerName(h.cfg.NextServer.String()))
	}

	if _, err := conn.WriteTo(reply.ToBytes(), peer); err != nil {
		h.logger.Printf("ERROR send %s to %s: %v", msgType, mac, err)
	}
}

func (h *handler) assign(mac string) net.IP {
	h.mu.Lock()
	defer h.mu.Unlock()

	if lease, ok := h.leases[mac]; ok && lease.expiresAt.After(time.Now()) {
		return lease.ip
	}

	for ip := cloneIP(h.nextIP); ; ip = incrementIP(ip) {
		if compareIP(ip, h.endIP) > 0 {
			return nil
		}
		if !h.isAllocated(ip) {
			expires := time.Now().Add(h.leaseTime)
			h.leases[mac] = lease{ip: ip, expiresAt: expires}
			h.nextIP = incrementIP(ip)
			return ip
		}
		if compareIP(ip, h.endIP) == 0 {
			break
		}
	}

	for ip := cloneIP(h.startIP); compareIP(ip, h.endIP) <= 0; ip = incrementIP(ip) {
		if !h.isAllocated(ip) {
			expires := time.Now().Add(h.leaseTime)
			h.leases[mac] = lease{ip: ip, expiresAt: expires}
			h.nextIP = incrementIP(ip)
			return ip
		}
	}
	return nil
}

func (h *handler) release(mac string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.leases, mac)
}

func (h *handler) isAllocated(ip net.IP) bool {
	for _, lease := range h.leases {
		if lease.expiresAt.After(time.Now()) && ip.Equal(lease.ip) {
			return true
		}
	}
	return false
}
