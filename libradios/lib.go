package libradios

import "net"

func ParseTCPAddress(addr, defaultPort string) (string, bool) {

	// Case:
	// [ipv6]:port
	// ipv4:port
	host, port, err := net.SplitHostPort(addr)

	if err == nil {

		if ip := net.ParseIP(host); ip != nil {

			if port == "" {
				port = defaultPort
			}

			return net.JoinHostPort(host, port), true
		}
	}

	// Plain IP without port
	if ip := net.ParseIP(addr); ip != nil {

		return net.JoinHostPort(addr, defaultPort), true
	}

	return "", false
}
