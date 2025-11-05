package dhcp

import (
	"log"
	"net"
	"sync"
	"time"

	"goosed/services/pxe-stack/internal/config"
)

type Server struct {
	cfg     config.DHCPConfig
	logger  *log.Logger
	handler *handler
}

type handler struct {
	cfg       config.DHCPConfig
	logger    *log.Logger
	mu        sync.Mutex
	leases    map[string]lease
	nextIP    net.IP
	startIP   net.IP
	endIP     net.IP
	leaseTime time.Duration
}

type lease struct {
	ip        net.IP
	expiresAt time.Time
}
