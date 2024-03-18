package health

import (
	"fmt"
	"net"
	"time"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/plugin"
)

func init() { plugin.Register("health", setup) }

func setup(c *caddy.Controller) error {
	addr, lame, lookup, err := parse(c)
	if err != nil {
		return plugin.Error("health", err)
	}

	h := &health{Addr: addr, lameduck: lame, lookup: lookup}

	c.OnStartup(h.OnStartup)
	c.OnRestart(h.OnReload)
	c.OnFinalShutdown(h.OnFinalShutdown)
	c.OnRestartFailed(h.OnStartup)

	// Don't do AddPlugin, as health is not *really* a plugin just a separate webserver running.
	return nil
}

func parse(c *caddy.Controller) (string, time.Duration, []string, error) {
	addr := ""
	dur := time.Duration(0)
	var lookup []string
	for c.Next() {
		args := c.RemainingArgs()

		switch len(args) {
		case 0:
		case 1:
			addr = args[0]
			if _, _, e := net.SplitHostPort(addr); e != nil {
				return "", 0, nil, e
			}
		default:
			return "", 0, nil, c.ArgErr()
		}

		for c.NextBlock() {
			switch c.Val() {
			case "lameduck":
				args := c.RemainingArgs()
				if len(args) != 1 {
					return "", 0, nil, c.ArgErr()
				}
				l, err := time.ParseDuration(args[0])
				if err != nil {
					return "", 0, nil, fmt.Errorf("unable to parse lameduck duration value: '%v' : %v", args[0], err)
				}
				dur = l
			case "lookup":
				for c.NextArg() {
					lookup = append(lookup, c.Val())
				}
				if len(lookup) == 0 {
					return "", 0, nil, c.ArgErr()
				}
			default:
				return "", 0, nil, c.ArgErr()
			}
		}
	}
	return addr, dur, lookup, nil
}
