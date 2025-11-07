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
	ifaceCandidates := getEnv("PXE_DHCP_INTERFACE", "eth0,en0")
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
	resolvedInterface, err := resolveDHCPInterface(ifaceCandidates, cfg.DHCP.ServerIP)
	if err != nil {
		return Config{}, err
	}
	cfg.DHCP.Interface = resolvedInterface

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
	if fallbacks := os.Getenv("PXE_HTTP_FALLBACK_PORTS"); fallbacks != "" {
		ports, err := parsePortList(fallbacks)
		if err != nil {
			return Config{}, fmt.Errorf("invalid PXE_HTTP_FALLBACK_PORTS: %w", err)
		}
		cfg.HTTP.FallbackPorts = ports
	}

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

func resolveDHCPInterface(spec string, serverIP net.IP) (string, error) {
	candidates := strings.Split(spec, ",")
	trimmed := make([]string, 0, len(candidates))
	for _, c := range candidates {
		name := strings.TrimSpace(c)
		if name == "" {
			continue
		}
		trimmed = append(trimmed, name)
	}

	tryAuto := false
	if len(trimmed) == 0 {
		tryAuto = true
	} else {
		next := make([]string, 0, len(trimmed))
		for _, name := range trimmed {
			if strings.EqualFold(name, "auto") {
				tryAuto = true
				continue
			}
			next = append(next, name)
		}
		trimmed = next
	}

	if tryAuto {
		if serverIP == nil {
			return "", fmt.Errorf("PXE_DHCP_INTERFACE=auto requires PXE_DHCP_SERVER_IP")
		}
		if name, err := interfaceByIP(serverIP); err == nil {
			return name, nil
		} else {
			return "", err
		}
	}

	for _, name := range trimmed {
		if _, err := net.InterfaceByName(name); err == nil {
			return name, nil
		}
	}

	availableIfaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("resolve PXE_DHCP_INTERFACE: candidates %q not found and unable to list interfaces: %w", trimmed, err)
	}
	available := make([]string, 0, len(availableIfaces))
	for _, iface := range availableIfaces {
		available = append(available, iface.Name)
	}
	return "", fmt.Errorf("resolve PXE_DHCP_INTERFACE: none of the candidates %q are present on this host (available: %s)", trimmed, strings.Join(available, ", "))
}

func interfaceByIP(ip net.IP) (string, error) {
	if ip == nil {
		return "", fmt.Errorf("cannot resolve interface for nil IP")
	}
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("list interfaces: %w", err)
	}
	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var candidate net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				candidate = v.IP
			case *net.IPAddr:
				candidate = v.IP
			}
			if candidate == nil {
				continue
			}
			if candidate.To4() != nil && candidate.Equal(ip) {
				return iface.Name, nil
			}
		}
	}
	return "", fmt.Errorf("no network interface found with address %s", ip.String())
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

func parsePortList(value string) ([]int, error) {
	parts := strings.Split(value, ",")
	ports := make([]int, 0, len(parts))
	seen := make(map[int]struct{}, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		port, err := strconv.Atoi(trimmed)
		if err != nil {
			return nil, fmt.Errorf("%q is not a valid integer", trimmed)
		}
		if port <= 0 || port > 65535 {
			return nil, fmt.Errorf("port %d is outside the valid range 1-65535", port)
		}
		if _, exists := seen[port]; exists {
			continue
		}
		seen[port] = struct{}{}
		ports = append(ports, port)
	}
	if len(ports) == 0 {
		return nil, nil
	}
	return ports, nil
}
