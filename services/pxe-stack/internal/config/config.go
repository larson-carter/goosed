package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

func Load() (Config, error) {
	cfg := Config{}

	cfg.DHCP.Enabled = getEnvBool("PXE_ENABLE_DHCP", true)
	cfg.DHCP.Interface = getEnv("PXE_DHCP_INTERFACE", "eth0")
	if start := os.Getenv("PXE_DHCP_RANGE_START"); start != "" {
		cfg.DHCP.RangeStart = net.ParseIP(start)
		if cfg.DHCP.RangeStart == nil {
			return Config{}, fmt.Errorf("invalid PXE_DHCP_RANGE_START: %q", start)
		}
	}
	if end := os.Getenv("PXE_DHCP_RANGE_END"); end != "" {
		cfg.DHCP.RangeEnd = net.ParseIP(end)
		if cfg.DHCP.RangeEnd == nil {
			return Config{}, fmt.Errorf("invalid PXE_DHCP_RANGE_END: %q", end)
		}
	}
	if mask := os.Getenv("PXE_DHCP_SUBNET_MASK"); mask != "" {
		ip := net.ParseIP(mask)
		if ip == nil {
			return Config{}, fmt.Errorf("invalid PXE_DHCP_SUBNET_MASK: %q", mask)
		}
		cfg.DHCP.SubnetMask = net.IPMask(ip.To4())
	}
	if router := os.Getenv("PXE_DHCP_ROUTER"); router != "" {
		cfg.DHCP.Router = net.ParseIP(router)
		if cfg.DHCP.Router == nil {
			return Config{}, fmt.Errorf("invalid PXE_DHCP_ROUTER: %q", router)
		}
	}
	if dns := os.Getenv("PXE_DHCP_DNS"); dns != "" {
		servers := strings.Split(dns, ",")
		cfg.DHCP.DNSServers = make([]net.IP, 0, len(servers))
		for _, s := range servers {
			ip := net.ParseIP(strings.TrimSpace(s))
			if ip == nil {
				return Config{}, fmt.Errorf("invalid DNS server %q", s)
			}
			cfg.DHCP.DNSServers = append(cfg.DHCP.DNSServers, ip)
		}
	}
	if lease := os.Getenv("PXE_DHCP_LEASE_SECONDS"); lease != "" {
		secs, err := strconv.Atoi(lease)
		if err != nil || secs <= 0 {
			return Config{}, fmt.Errorf("invalid PXE_DHCP_LEASE_SECONDS: %q", lease)
		}
		cfg.DHCP.LeaseTime = time.Duration(secs) * time.Second
	} else {
		cfg.DHCP.LeaseTime = 24 * time.Hour
	}
	if sip := os.Getenv("PXE_DHCP_SERVER_IP"); sip != "" {
		cfg.DHCP.ServerIP = net.ParseIP(sip)
		if cfg.DHCP.ServerIP == nil {
			return Config{}, fmt.Errorf("invalid PXE_DHCP_SERVER_IP: %q", sip)
		}
	}
	if ns := os.Getenv("PXE_DHCP_NEXT_SERVER"); ns != "" {
		cfg.DHCP.NextServer = net.ParseIP(ns)
		if cfg.DHCP.NextServer == nil {
			return Config{}, fmt.Errorf("invalid PXE_DHCP_NEXT_SERVER: %q", ns)
		}
	}
	cfg.DHCP.BootFilename = getEnv("PXE_DHCP_BOOT_FILE", "undionly.kpxe")

	if cfg.DHCP.Enabled {
		if cfg.DHCP.RangeStart == nil || cfg.DHCP.RangeEnd == nil {
			return Config{}, fmt.Errorf("PXE_DHCP_RANGE_START and PXE_DHCP_RANGE_END are required when DHCP is enabled")
		}
		if cfg.DHCP.RangeStart.To4() == nil || cfg.DHCP.RangeEnd.To4() == nil {
			return Config{}, fmt.Errorf("PXE_DHCP range must be IPv4 addresses")
		}
		if bytesCompare(cfg.DHCP.RangeStart.To4(), cfg.DHCP.RangeEnd.To4()) > 0 {
			return Config{}, fmt.Errorf("PXE_DHCP_RANGE_START must be <= PXE_DHCP_RANGE_END")
		}
		if cfg.DHCP.ServerIP == nil {
			return Config{}, fmt.Errorf("PXE_DHCP_SERVER_IP is required when DHCP is enabled")
		}
		if cfg.DHCP.ServerIP.To4() == nil {
			return Config{}, fmt.Errorf("PXE_DHCP_SERVER_IP must be an IPv4 address")
		}
		if cfg.DHCP.SubnetMask == nil {
			cfg.DHCP.SubnetMask = cfg.DHCP.RangeStart.DefaultMask()
		}
		if cfg.DHCP.Router == nil {
			cfg.DHCP.Router = cfg.DHCP.ServerIP
		}
	}

	cfg.TFTP.Enabled = getEnvBool("PXE_ENABLE_TFTP", true)
	cfg.TFTP.Address = getEnv("PXE_TFTP_ADDRESS", ":69")
	cfg.TFTP.RootDir = getEnv("PXE_TFTP_ROOT", "/var/lib/tftpboot")
	cfg.TFTP.ReadOnly = getEnvBool("PXE_TFTP_READ_ONLY", true)
	cfg.TFTP.TimeoutSec = getEnvInt("PXE_TFTP_TIMEOUT", 5)

	cfg.HTTP.Enabled = getEnvBool("PXE_ENABLE_HTTP", true)
	cfg.HTTP.Port = getEnvInt("PXE_HTTP_PORT", 8080)
	cfg.HTTP.APIEndpoint = getEnv("PXE_HTTP_API_ENDPOINT", "http://api.goose.local")
	cfg.HTTP.BrandingFS = os.Getenv("PXE_HTTP_BRANDING_FS")

	return cfg, nil
}

func bytesCompare(a, b net.IP) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}
	aa := a.To4()
	bb := b.To4()
	if aa == nil || bb == nil {
		return strings.Compare(a.String(), b.String())
	}
	for i := 0; i < len(aa); i++ {
		if aa[i] < bb[i] {
			return -1
		}
		if aa[i] > bb[i] {
			return 1
		}
	}
	return 0
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		b, err := strconv.ParseBool(v)
		if err == nil {
			return b
		}
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}
