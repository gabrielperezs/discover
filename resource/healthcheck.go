package resource

import (
	"time"
)

var (
	defaultInterval = 10 * time.Second
)

type HealthCheck struct {
	URL         string
	RespCode    int
	RespContent string
	Interval    string
	exit        bool
}
