package pluginK8S

import (
	"log"
	"net/url"
	"os/user"
	"strconv"
	"strings"
	"time"

	"github.com/gabrielperezs/discover/discoverlib"
)

type Config struct {
	discoverlib.ConfigBase
	Namespace      string
	MasterURL      string
	KubeConfigPath string
}

func (c *Config) Load(u *url.URL) error {
	for k, v := range u.Query() {
		switch strings.ToLower(k) {
		case "refresh":
			c.Refresh, _ = time.ParseDuration(v[0])
		case "namespace":
			c.Namespace = v[0]
		case "path":
			c.KubeConfigPath = v[0]
		case "weight":
			c.Weight, _ = strconv.ParseInt(v[0], 10, 64)
		default:
			log.Printf("WARN: unknown value in dns plugin %s %s", k, v)
		}
	}

	c.Port = u.Port()
	for i, s := range strings.Split(u.Scheme, "+") {
		if i == 1 {
			c.Protocol = s
		}
	}

	if c.Namespace == "" {
		c.Namespace = "default"
	}

	// Default path for kubeconfig
	if c.KubeConfigPath == "" {
		if usr, err := user.Current(); err == nil {
			c.KubeConfigPath = usr.HomeDir + "/.kube/config"
		}
	}
	return nil
}
