package resource

import (
	"bytes"
	"crypto/tls"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/gabrielperezs/discover/discoverlib"
)

const (
	limitRange                   = 1024 * 1024
	defaultTLSHandshakeTimeout   = time.Second * 10
	defaultResponseHeaderTimeout = time.Second * 60
	defaultExpectContinueTimeout = time.Second * 1
	defaultIdleConnTimeout       = time.Second * 30
	defaultRateLimit             = 8
)

type Resource struct {
	Protocol     string
	Host         string
	useTLS       bool
	Transport    *http.Transport
	HealthCheck  HealthCheck
	lastUpdate   time.Time
	healthStatus int64
	close        bool
}

func New(p discoverlib.Plugin, host string, useTLS bool, Healthcheck HealthCheck) *Resource {
	customDialer := newCustomDialer(host)
	r := &Resource{
		Host:        host,
		Protocol:    p.Protocol(),
		HealthCheck: Healthcheck,
		Transport: &http.Transport{
			MaxIdleConns:          100,
			IdleConnTimeout:       defaultIdleConnTimeout,
			TLSHandshakeTimeout:   defaultTLSHandshakeTimeout,
			ExpectContinueTimeout: defaultExpectContinueTimeout,
			ResponseHeaderTimeout: p.Timeout(),
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
			ForceAttemptHTTP2: false,
			DialContext:       customDialer.DialContext,
			DialTLSContext:    customDialer.DialTLSContext,
		},
		lastUpdate: time.Now(),
	}
	if r.HealthCheck.URL != "" {
		go r.runHealthCheck()
	} else {
		r.healthStatus = 1
	}
	return r
}

func (r *Resource) Before(t time.Time) bool {
	return r.lastUpdate.Add(1 * time.Minute).Before(t)
}

func (r *Resource) Update() {
	r.lastUpdate = time.Now()
}

func (r *Resource) IsHealthy() bool {
	return atomic.LoadInt64(&r.healthStatus) == 1
}

func (r *Resource) Close() {
	r.close = true
	r.Transport.CloseIdleConnections()
}

func (r *Resource) IsClose() bool {
	return r.close
}

func (r *Resource) runHealthCheck() {
	interval, _ := time.ParseDuration(r.HealthCheck.Interval)
	if interval.Nanoseconds() == 0 {
		interval = defaultInterval
	}
	for {
		if r.close {
			return
		}
		r.doHealthCheck()
		time.Sleep(interval)
	}
}

func (r *Resource) doHealthCheck() bool {
	if r.HealthCheck.URL != "" && !r.isNodeHealthy(r.HealthCheck.URL) {
		atomic.CompareAndSwapInt64(&r.healthStatus, 1, 0)
		statUnhealthyNodes.WithLabelValues(r.Host).Add(1)
		return false
	}
	atomic.CompareAndSwapInt64(&r.healthStatus, 0, 1)
	return true
}

func (r *Resource) isNodeHealthy(orgurl string) bool {
	req, err := http.NewRequest(http.MethodGet, orgurl, nil)
	if err != nil {
		log.Printf("Discover: error health check url: %s - %s", orgurl, err.Error())
		return false
	}

	_, p, _ := net.SplitHostPort(req.Host)
	if p == "" {
		p = "80"
	}

	h, _, _ := net.SplitHostPort(r.Host)
	customDialer := newCustomDialer(h + ":" + p)
	client := &http.Client{
		Transport: &http.Transport{
			DialContext:    customDialer.DialContext,
			DialTLSContext: customDialer.DialTLSContext,
		},
		Timeout: 2 * time.Second,
	}
	res, err := client.Do(req)
	if err != nil {
		return false
	}
	defer res.Body.Close()

	if res.StatusCode != r.HealthCheck.RespCode {
		io.Copy(ioutil.Discard, res.Body)
		return false
	}

	if r.HealthCheck.RespContent != "" {
		b, _ := ioutil.ReadAll(res.Body)
		return bytes.Contains(b, *(*[]byte)(unsafe.Pointer(&r.HealthCheck.RespContent)))
	}

	io.Copy(ioutil.Discard, res.Body)
	return true
}
