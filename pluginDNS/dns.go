package pluginDNS

import (
	"net"
	"time"

	"github.com/tevino/abool"
)

type PluginDNS struct {
	cfg     Config
	C       chan []string
	t       *time.Timer
	refresh time.Duration
	exit    *abool.AtomicBool
}

func New(c Config) *PluginDNS {
	l := &PluginDNS{
		cfg:     c,
		C:       make(chan []string, 1),
		t:       time.NewTimer(c.Refresh),
		refresh: c.Refresh,
		exit:    abool.New(),
	}
	l.update()
	go l.interval()
	return l
}

func (l *PluginDNS) Get() chan []string {
	return l.C
}

func (l *PluginDNS) Protocol() string {
	return l.cfg.Protocol
}

func (l *PluginDNS) Weight() int64 {
	return l.cfg.Weight
}

func (l *PluginDNS) Timeout() time.Duration {
	return l.cfg.Timeout
}

func (l *PluginDNS) Exit() {
	l.exit.Set()
	l.t.Stop()
	time.Sleep(l.refresh + (time.Millisecond * 100))
	close(l.C)
}

func (l *PluginDNS) interval() {
	for range l.t.C {
		l.update()
		l.t.Reset(l.refresh)
	}
}

func (l *PluginDNS) update() {
	if hosts, err := l.get(); err == nil {
		for i, v := range hosts {
			hosts[i] = v + ":" + l.cfg.Port
		}
		if !l.exit.IsSet() {
			l.C <- hosts
		}
	}
}

func (l *PluginDNS) get() ([]string, error) {
	return net.LookupHost(l.cfg.Hostname)
}
