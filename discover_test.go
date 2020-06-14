package discover

import (
	"math/rand"
	"testing"
	"time"
)

func TestLoadPlugins(t *testing.T) {
	d := &Discover{}
	c := &Config{
		DiscoverURI: []string{
			"k8s://kubeconfig:80?namespace=wbsearch&refresh=60s&watch=true",
			"dns://www.dotwconnect.com:80?refresh=5s",
		},
	}
	err := d.loadPlugins(c.DiscoverURI)
	if err != nil {
		t.Error(err)
	}
}

func TestStoreAndLoad(t *testing.T) {
	d := &Discover{}
	c := &Config{
		DiscoverURI: []string{
			"dns://1.1.1.1:80?refresh=10s",
			"dns://1.1.1.2:80?refresh=10s",
			"dns://1.1.1.3:80?refresh=10s",
		},
	}
	err := d.loadPlugins(c.DiscoverURI)
	if err != nil {
		t.Error(err)
	}

	hosts := []string{
		"1.1.1.1",
		"1.1.1.2",
		"1.1.1.3",
	}
	d.update(hosts, 0)

	if len(hosts) != len(d.Resources()) {
		t.Error("Invalid number of hosts")
	}

	valid := true
	for i, v := range d.Resources() {
		if v.Host != hosts[i] {
			valid = false
		}
	}
	if !valid {
		t.Error("Invalid hosts store")
	}
}

func TestNextResource(t *testing.T) {
	d := &Discover{}
	c := &Config{
		DiscoverURI: []string{
			"dns://1.1.1.1:80?refresh=10s",
			"dns://1.1.1.2:80?refresh=10s",
			"dns://1.1.1.3:80?refresh=10s",
		},
	}
	err := d.loadPlugins(c.DiscoverURI)
	if err != nil {
		t.Error(err)
	}

	hosts := []string{
		"1.1.1.1",
		"1.1.1.2",
		"1.1.1.3",
	}
	d.update(hosts, 0)

	for i, v := range d.Resources() {
		t.Logf("Resource: %d - %v", i, v.Host)
	}

	for i := 1; i <= 20; i++ {
		r := d.NextHealthy()
		if r.Host != hosts[i%3] {
			t.Errorf("error %v != %v", r.Host, hosts[i%3])
		}
	}

}

func TestTickerPlugins(t *testing.T) {
	d := &Discover{
		resources: make(Resources, 0),
	}
	d.atomicRes.Store(make(Resources, 0))

	c := &Config{
		DiscoverURI: []string{
			//"k8s://kubeconfig:80?namespace=wbsearch&refresh=3s&watch=true",
			"dns://www.dotwconnect.com:80?refresh=1s",
			"dns://1a7f6e769243af4202942ae498c376a51-1588922734.eu-west-1.elb.amazonaws.com:80?refresh=1s",
		},
	}
	err := d.loadPlugins(c.DiscoverURI)
	if err != nil {
		t.Error(err)
	}

	go func() {
		sleep := 10 * time.Second
		rand.Seed(time.Now().UnixNano())
		sleep = sleep + (time.Duration(rand.Int63n(500)) * time.Millisecond)
		time.Sleep(sleep)
		d.Exit()
	}()

	d.listener()
}
