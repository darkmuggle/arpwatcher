package metrics

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/darkmuggle/arptracker/pkg/arping"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

const (
	gratitousArp = "ff:ff:ff:ff:ff:ff"

	defaultPort = 2112
	metricsPath = "/metrics"
)

type macSeen struct {
	mac       net.HardwareAddr
	count     int64
	firstSeen time.Time
	lastSeen  time.Time
	network   *net.IPNet
}

type ipTable struct {
	mu    sync.Mutex
	log   *logrus.Entry
	table map[string][]macSeen

	expiredMacs []macSeen
}

// record adds a new arping result to the
func (i *ipTable) record(r arping.ArpPingResult) {
	i.mu.Lock()
	defer i.mu.Unlock()

	promCounterMacReplies.Add(1)

	if r.IP == nil || r.Mac == nil || r.Mac.String() == gratitousArp {
		return
	}

	if _, ok := i.table[r.IP.String()]; !ok {
		i.table[r.IP.String()] = []macSeen{}
	}

	found := false
	for idx, v := range i.table[r.IP.String()] {
		if strings.EqualFold(r.Mac.String(), v.mac.String()) {
			found = true
			i.table[r.IP.String()][idx].lastSeen = time.Now()
			i.table[r.IP.String()][idx].count++
			break
		}
	}

	if !found {
		i.table[r.IP.String()] = append(
			i.table[r.IP.String()],
			macSeen{
				mac:       *r.Mac,
				network:   r.Network,
				count:     1,
				firstSeen: time.Now(),
				lastSeen:  time.Now(),
			})
	}
}

// updateMetrics checks whether or not the entries have been seen recently
func (i *ipTable) updateMetrics() {
	i.mu.Lock()
	defer i.mu.Unlock()

	activeIps := 0
	macIps := make(map[string][]string)

	expireTime := time.Now().Add(-20 * time.Minute)
	for ip, v := range i.table {
		newV := []macSeen{}

		for _, val := range v {
			// If the last seen time is after the "expire time", then include it
			// in the results.
			if val.lastSeen.After(expireTime) {
				newV = append(newV, val)
				if _, ok := macIps[val.mac.String()]; !ok {
					macIps[val.mac.String()] = []string{}
				}

				// add the IP to the indivual MAC address table
				addIpToMac := true
				for _, v := range macIps[val.mac.String()] {
					if v == val.network.IP.String() {
						addIpToMac = false
					}
				}
				if addIpToMac {
					macIps[val.mac.String()] = append(
						macIps[val.mac.String()],
						ip,
					)
				}
			} else {
				i.expiredMacs = append(i.expiredMacs, val)
			}
		}

		i.table[ip] = newV

		if len(v) != 0 {
			activeIps++
			promGaugeIpMacAdddresses.With(
				prometheus.Labels{
					promLabelIPAddress: ip,
				},
			).Set(float64(len(v)))
		} else {
			// expire the IP
			promGaugeIpMacAdddresses.With(
				prometheus.Labels{
					promLabelIPAddress: ip,
				},
			).Set(0)
		}
	}

	// Report the number of IPs seen
	promGaugeIPAddresses.Set(float64(activeIps))

	// Set the the count IP addresses per MAC address
	for k, v := range macIps {
		promGaugeMacAddressIPs.With(
			prometheus.Labels{
				promLabelMacAddress: k,
			},
		).Set(float64(len(v)))
	}

	// Report the number of MacAddresses seen
	promGaugeUniqueMacAddresses.Set(float64(len(macIps)))

	// Set expired macs to 0
	for _, v := range i.expiredMacs {
		promGaugeMacAddressIPs.With(
			prometheus.Labels{
				promLabelMacAddress: v.mac.String(),
			},
		).Set(0)
	}
}

// Watch takes the results from an arping.ResultChan and generates Prometheus metrics.
func Watch(l *logrus.Entry, port int, results arping.ResultChan, termChan chan struct{}) error {
	if l == nil {
		return errors.New("logrus.Entry is nil")
	}

	mt := ipTable{
		table: make(map[string][]macSeen),
		mu:    sync.Mutex{},
		log:   l.WithField("func", "metrics"),
	}

	// setup a metrics port
	go func() {
		if port == 0 {
			port = defaultPort
		}
		for {
			http.Handle(metricsPath, promhttp.Handler())
			_ = http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
		}
	}()

	// expire stuff
	go func(mt *ipTable) {
		for {
			select {
			case <-termChan:
				return
			case <-time.After(1 * time.Second):
				mt.updateMetrics()
			}
		}
	}(&mt)

	// watch the results channel
	go func(mt *ipTable, results arping.ResultChan) {
		for {
			select {
			case r := <-results:
				mt.record(r)
			case <-termChan:
				return
			}
		}
	}(&mt, results)

	return nil
}
