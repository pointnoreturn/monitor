package libradios

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/grandcat/zeroconf"
	"github.com/joho/godotenv"
)

func Discover(ctx context.Context, timeout time.Duration) []ResolvedService {
	resolver, _ := zeroconf.NewResolver(nil)
	entries := make(chan *zeroconf.ServiceEntry)

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	services := []ResolvedService{}

	go func() {
		_ = resolver.Browse(ctx, "_meshtastic._tcp", "local.", entries)
	}()

	timer := time.NewTimer(timeout)

	for {
		select {
		case e := <-entries:
			if e == nil {
				continue
			}

			endpoint := ""
			if len(e.AddrIPv4) > 0 {
				endpoint = fmt.Sprintf("%s:%d", e.AddrIPv4[0].String(), e.Port)
			} else if len(e.AddrIPv6) > 0 {
				endpoint = fmt.Sprintf("[%s]:%d", e.AddrIPv6[0].String(), e.Port)
			}

			// key=value pairs in Entry.Text
			args, err := godotenv.Unmarshal(strings.Join(e.Text, "\n"))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				args = make(map[string]string)
			}

			services = append(services, ResolvedService{
				Endpoint: endpoint,
				Entry:    e,
				Args:     args,
			})

		case <-ctx.Done():
			return services
		case <-timer.C:
			return services
		}
	}
}
