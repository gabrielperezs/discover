package discoverlib

import (
	"time"
)

type ConfigBase struct {
	Hostname string
	Port     string
	Protocol string
	Weight   int64
	Refresh  time.Duration
	Timeout  time.Duration
}
