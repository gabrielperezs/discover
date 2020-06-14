package discover

import (
	"errors"
	"math"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gabrielperezs/discover/discoverlib"
	"github.com/gabrielperezs/discover/pluginDNS"
	"github.com/gabrielperezs/discover/pluginK8S"
	"github.com/gabrielperezs/discover/resource"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	exitingDelay   = 10 * time.Second
	ErrErrorPlugin = errors.New("Unknown plugin")

	statsNoResources = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "wbrouter_discover_no_resources",
	}, []string{"Label"})
	statsResources = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "wbrouter_discover_resources",
	}, []string{"Label"})
	statsResourcesUnHealthy = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "wbrouter_discover_resources_unhealthy",
	}, []string{"Label"})
)

type Config struct {
	Label       string
	DiscoverURI []string
	HealtCheck  resource.HealthCheck
}

type Discover struct {
	Label       string
	Plugins     []discoverlib.Plugin
	healthCheck resource.HealthCheck
	atomicRes   atomic.Value
	resources   Resources
	count       int64
	exit        bool
	n           int64
}

func New(c Config) *Discover {
	d := &Discover{
		Label:       c.Label,
		Plugins:     make([]discoverlib.Plugin, 0),
		resources:   make(Resources, 0),
		healthCheck: c.HealtCheck,
	}
	d.atomicRes.Store(make(Resources, 0))

	d.loadPlugins(c.DiscoverURI)
	go d.listener()

	return d
}

func (d *Discover) NextHealthy() *resource.Resource {
	r := d.Resources()
	size := len(r)
	for i := 0; i < size; i++ {
		n := atomic.AddInt64(&d.n, 1)
		if n >= math.MaxInt64-1000 {
			if atomic.CompareAndSwapInt64(&d.n, math.MaxInt64-1000, 0) {
				n = atomic.AddInt64(&d.n, 1)
			}
		}
		l := r[n%int64(size)]
		if l.IsHealthy() {
			return l
		}
	}
	statsNoResources.WithLabelValues(d.Label).Add(1)
	return nil
}

func (d *Discover) loadPlugins(uris []string) error {
	for _, s := range uris {
		u, err := url.ParseRequestURI(s)
		if err != nil {
			return err
		}

		switch strings.ToLower(u.Scheme) {
		case "k8s":
			c := pluginK8S.Config{}
			if err := c.Load(u); err != nil {
				return err
			}
			d.Plugins = append(d.Plugins, pluginK8S.New(c))
		case "dns":
			c := pluginDNS.Config{}
			if err := c.Load(u); err != nil {
				return err
			}
			d.Plugins = append(d.Plugins, pluginDNS.New(c))
		default:
			return ErrErrorPlugin
		}
	}
	return nil
}

func (d *Discover) listener() {
	cases := make([]reflect.SelectCase, len(d.Plugins))
	for i, p := range d.Plugins {
		cases[i] = reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(p.Get()),
		}
	}

	remaining := len(cases)
	for remaining > 0 {
		chosen, value, ok := reflect.Select(cases)
		if !ok {
			// The chosen channel has been closed, so zero out the channel to disable the case
			cases[chosen].Chan = reflect.ValueOf(nil)
			remaining--
			continue
		}

		d.stats()

		slice, ok := value.Interface().([]string)
		if !ok {
			continue
		}
		d.update(slice, chosen)
	}
}

func (d *Discover) Len() int {
	return int(atomic.LoadInt64(&d.count))
}

func (d *Discover) Resources() Resources {
	return d.atomicRes.Load().(Resources)
}

func (d *Discover) Exit() {
	d.exit = true
	go d.lazyExit()
}

func (d *Discover) stats() {
	var healthy, unhealthy float64
	for _, r := range d.Resources() {
		if r.IsHealthy() {
			healthy++
		} else {
			unhealthy++
		}
	}
	statsResources.WithLabelValues(d.Label).Set(healthy)
	statsResourcesUnHealthy.WithLabelValues(d.Label).Set(unhealthy)
}

func (d *Discover) lazyExit() {
	wg := &sync.WaitGroup{}
	for _, p := range d.Plugins {
		wg.Add(1)
		go func(p discoverlib.Plugin) {
			p.Exit()
			wg.Done()
		}(p)
	}
	wg.Wait()
	// Clean local resources
	for i, r := range d.resources {
		r.Close()
		d.resources[i] = nil
	}
	d.resources = d.resources[:0]
	time.Sleep(exitingDelay)
	d.atomicRes.Store(make(Resources, 0))
}

func (d *Discover) update(slice []string, chosen int) {
	if d.resources.update(d.Plugins[chosen], slice, d.healthCheck) {
		r := d.resources.clone()
		d.atomicRes.Store(r)
		atomic.StoreInt64(&d.count, int64(len(r)))
		statsResources.WithLabelValues(d.Label).Set(float64(len(r)))
		d.resources.clean()
	}
}
