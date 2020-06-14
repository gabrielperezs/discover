package resource

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	maxErrors = 3
)

var (
	// ErrNoHostAvailable when the backend don't have hosts
	ErrNoHostAvailable = errors.New("No host available")
)

// CustomDialer implements a roundrobin connection pool based on
// the resources in the backend. Could be IP or hosts. If is hosts
// it will refresh the DNS resolution every 5s and will all the returned
// IPs in the slice of hosts
type CustomDialer struct {
	servername string
	d          *net.Dialer
	tls        *tls.Conn
	addr       string
}

func (cd *CustomDialer) Addr() string {
	return cd.addr
}

// Create a new custom Dialer with specific hosts
func newCustomDialer(addr string) *CustomDialer {
	cd := &CustomDialer{
		addr: addr,
		d: &net.Dialer{
			Timeout:       5 * time.Second,
			KeepAlive:     10 * time.Second,
			FallbackDelay: -1,
		},
	}
	return cd
}

// DialContext will use one IP from the resources using the round-robin
// and call to the original net.DialContext
func (cd *CustomDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	conn, err := cd.d.DialContext(ctx, network, cd.addr)
	if err == nil {
		statConn.WithLabelValues(cd.addr).Inc()
		return conn, err
	}
	switch v := err.(type) {
	case *net.OpError:
		statConnErr.WithLabelValues(cd.addr, v.Error()).Inc()
	default:
		statConnErr.WithLabelValues(cd.addr, err.Error()).Inc()
	}
	return conn, err
}

func (cd *CustomDialer) DialTLSContext(ctx context.Context, network, address string) (net.Conn, error) {
	conn, err := tls.Dial(network, cd.addr, &tls.Config{
		ServerName: cd.servername,
	})
	if err == nil {
		statConn.WithLabelValues(cd.addr).Inc()
		return conn, err
	}
	statConnErr.WithLabelValues(cd.addr, err.Error()).Inc()
	return conn, err
}

var (
	statConn = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "wbrouter_backend_conns",
		Help: "Connections to the backend",
	}, []string{"Host"})

	statConnErr = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "wbrouter_backend_conns_errors",
		Help: "Connections errors to the backend",
	}, []string{"Host", "Error"})

	statAddrs = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "wbrouter_backend_addrs",
		Help: "Address in the backends",
	}, []string{"Host"})

	statUnhealthyNodes = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "wbrouter_backend_nodes_unhealthy",
		Help: "Unhealthy backend nodes",
	}, []string{"Host"})
)
