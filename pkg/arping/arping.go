package arping

import (
	"errors"
	"net"
	"net/netip"
	"time"

	"github.com/mdlayher/arp"
	"github.com/sirupsen/logrus"
)

// termChan is used to terminate the process
type termChan chan struct{}

type ArpPingResult struct {
	Mac     *net.HardwareAddr
	IP      *net.IP
	Network *net.IPNet
}

// ResultChan is a channel for responses
type ResultChan chan ArpPingResult

type arpPinger struct {
	net    *net.IPNet
	netIp  net.IP
	iface  *net.Interface
	ipList []net.IP

	log *logrus.Entry

	resultChan ResultChan
	termChan   termChan
}

type Arpinger interface {
	Run() (ResultChan, error)
	Stop()
}

type ArpResponse struct{}

func NewApringer(l *logrus.Entry, network, ifaceName string) (Arpinger, error) {
	if l == nil {
		return nil, errors.New("log must not be nil")
	}
	netIp, netw, err := net.ParseCIDR(network)
	if err != nil {
		return nil, err
	}

	ipList, count, err := getCidrIpAddresses(network)
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, errors.New("network has no hosts")
	}

	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return nil, err
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return nil, err
	} else if len(addrs) == 0 {
		return nil, errors.New("unable to determine the ip address interface")
	}

	ll := l.WithFields(logrus.Fields{
		"func":      "pinger",
		"cidr":      network,
		"interface": ifaceName,
	})

	ll.WithField("ip addresses", addrs).Info("host IP")

	return &arpPinger{
		net:        netw,
		netIp:      netIp,
		iface:      iface,
		ipList:     ipList,
		log:        ll,
		resultChan: make(ResultChan, count),
		termChan:   make(termChan, 1),
	}, nil
}

func (ap *arpPinger) Run() (ResultChan, error) {
	ap.log.WithField("count", len(ap.ipList)).Info("starting pinger")

	doneChan := make(chan struct{}, len(ap.ipList))

	recordPing := func(index int) {
		i := ap.ipList[index]
		responses := make(map[string]int)

		// arping the target 10 times. If there is no response to the ping,
		// move on.
		for x := 0; x <= 10; x++ {
			if r, err := ap.arpPing(&i); err == nil && r != nil {
				if _, ok := responses[r.String()]; ok {
					responses[r.String()]++
				} else {
					responses[r.String()] = 1
				}

				ap.resultChan <- ArpPingResult{
					IP:      &i,
					Mac:     r,
					Network: ap.net,
				}

			} else {
				break
			}
		}

		// Show where the duplicate are
		if len(responses) > 1 {
			ap.log.WithFields(logrus.Fields{
				"ip":         i,
				"duplicates": len(responses),
			}).Warn("duplicate mac addresss responded")

			for k, v := range responses {
				ap.log.WithFields(logrus.Fields{
					"ip":      i,
					"hw addr": k,
					"count":   v,
				}).Warn()
			}
		}
		doneChan <- struct{}{}

	}

	go func() {
		doneCount := 0
		var cycle int64 = 0
		for {
			select {
			case <-ap.termChan:
				return
			case <-doneChan:
				doneCount++
			case <-time.After(5 * time.Second):
				if doneCount == len(ap.ipList) {
					doneCount = 0
					cycle++
					ap.log.WithField("cycle", cycle).Info("starting cycle")
				} else {
					ap.log.WithFields(logrus.Fields{
						"remaining": (len(ap.ipList) - doneCount),
						"total":     len(ap.ipList),
						"cylce":     cycle,
					}).Info()
				}
			}
		}
	}()

	go func() {
		index := 0
		for {
			select {
			case <-ap.termChan:
				return
			default:
				if index == len(ap.ipList) {
					index = 0
				}
				recordPing(index)
				index++
			}
		}
	}()

	return ap.resultChan, nil
}

func (ap *arpPinger) Stop() {
	ap.termChan <- struct{}{}
}

func (ap *arpPinger) arpPing(ip *net.IP) (*net.HardwareAddr, error) {
	if ip == nil {
		return nil, nil
	}
	client, err := arp.Dial(ap.iface)
	if err != nil {
		return nil, err
	}
	ipAddress, err := netip.ParseAddr(ip.String())
	if err != nil {
		return nil, err
	}

	client.SetReadDeadline(time.Now().Add(1 * time.Second))
	hw, err := client.Resolve(ipAddress)
	return &hw, err
}

// getCidrIpAddresses returns a list of net.IP's and count for
// a given CIDR block.
func getCidrIpAddresses(cidr string) ([]net.IP, int, error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, 0, err
	}

	inc := func(ip net.IP) {
		for j := len(ip) - 1; j >= 0; j-- {
			ip[j]++
			if ip[j] > 0 {
				break
			}
		}
	}

	var ips []net.IP
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
		newIp := net.ParseIP(ip.String())
		ips = append(ips, newIp)
	}

	// remove network address and broadcast address
	lenIPs := len(ips)
	switch {
	case lenIPs < 2:
		return ips, lenIPs, nil

	default:
		return ips[1 : len(ips)-1], lenIPs - 2, nil
	}
}
