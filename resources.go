package discover

import (
	"log"
	"time"

	"github.com/gabrielperezs/discover/discoverlib"
	"github.com/gabrielperezs/discover/resource"
)

type Resources []*resource.Resource

func (d *Resources) update(p discoverlib.Plugin, addrs []string, hc resource.HealthCheck) (updates bool) {
	t := time.Now()
	for _, addr := range addrs {
		if r := d.exists(addr); r != nil {
			r.Update()
			continue
		}
		r := resource.New(p, addr, false, hc)
		if r == nil {
			log.Panicf("What?")
		}
		*d = append(*d, r)
		updates = true
	}

	for _, r := range *d {
		if r.Before(t) {
			r.Close()
			updates = true
		}
	}
	return
}

func (d *Resources) exists(h string) *resource.Resource {
	for _, r := range *d {
		if r.Host == h {
			return r
		}
	}
	return nil
}

func (d *Resources) clean() {
	n := make(Resources, 0)
	for _, v := range *d {
		if v == nil || v.IsClose() {
			continue
		}
		n = append(n, v)
	}
	*d = n
}

func (d *Resources) clone() Resources {
	n := make(Resources, 0)
	for _, v := range *d {
		n = append(n, v)
	}
	return n
}
