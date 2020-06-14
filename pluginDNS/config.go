package pluginDNS

import (
	"log"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gabrielperezs/discover/discoverlib"
)

var (
	defaultRefresh = 5 * time.Second
)

type Config struct {
	discoverlib.ConfigBase
}

func (c *Config) Load(u *url.URL) (err error) {
	for k, v := range u.Query() {
		switch strings.ToLower(k) {
		case "refresh":
			c.Refresh, _ = time.ParseDuration(v[0])
		case "weight":
			c.Weight, _ = strconv.ParseInt(v[0], 10, 64)
		default:
			log.Printf("WARN: unknown value in dns plugin %s %s", k, v)
		}
	}
	c.Hostname, c.Port, err = net.SplitHostPort(u.Host)

	if c.Refresh.Nanoseconds() == 0 {
		c.Refresh = defaultRefresh
	}

	for i, s := range strings.Split(u.Scheme, "+") {
		if i == 1 {
			c.Protocol = s
		}
	}

	return
}
