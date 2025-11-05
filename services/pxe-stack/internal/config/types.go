package config

import (
	"net"
	"time"
)

type Config struct {
	DHCP DHCPConfig
	TFTP TFTPConfig
	HTTP HTTPConfig
}

type DHCPConfig struct {
	Enabled      bool
	Interface    string
	RangeStart   net.IP
	RangeEnd     net.IP
	SubnetMask   net.IPMask
	Router       net.IP
	DNSServers   []net.IP
	LeaseTime    time.Duration
	ServerIP     net.IP
	NextServer   net.IP
	BootFilename string
}

type TFTPConfig struct {
	Enabled    bool
	Address    string
	RootDir    string
	ReadOnly   bool
	TimeoutSec int
}

type HTTPConfig struct {
	Enabled     bool
	Port        int
	APIEndpoint string
	BrandingFS  string
}
