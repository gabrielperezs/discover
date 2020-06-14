package discoverlib

import "time"

type Plugin interface {
	Get() chan []string
	Protocol() string
	Weight() int64
	Timeout() time.Duration
	Exit()
}
