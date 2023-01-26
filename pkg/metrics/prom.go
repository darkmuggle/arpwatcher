package metrics

import "github.com/prometheus/client_golang/prometheus"

const (
	// promLabelIPAddress is a label for ip addresses
	promLabelIPAddress = "ip_address"
	// promLabelMacAddress is a label for mac addresses
	promLabelMacAddress = "mac_address"
)

var (
	// promGuageIpMacAddresses is a counter to count the number of Mac addresess
	// associated with a single IP in a Subnet.
	promGaugeIpMacAdddresses = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "network_ip_mac_address_current",
			Help: "network_ip_mac_address_current is the number of mac addresses seen with a current IP address",
		},
		[]string{
			promLabelIPAddress,
		},
	)

	// promGaugeMacAddressIps is a gauge of how many IP's a single mac address has
	promGaugeMacAddressIPs = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "network_arping_mac_address_ip_current",
			Help: "network_arping_mac_address_ip_current is the number of ip address associate with a single mac address",
		},
		[]string{
			promLabelMacAddress,
		},
	)

	// promGaugeIPAddresses is a guage of the number of free ips in the cidr block.
	promGaugeIPAddresses = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "network_arping_ip_addresses",
			Help: "network_arping_ip_addresses_unique is a guage of the number of ip adddresses",
		},
	)

	// promGaugeUniqueMacAddresses is a guage of the number of free ips in the cidr block.
	promGaugeUniqueMacAddresses = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "network_arping_mac_addresses_unique",
			Help: "network_arping_mac_addresses_unique is a guage of the unique mac addresses seen",
		},
	)

	promCounterMacReplies = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "network_arping_replies",
			Help: "network_arping_replies is a counter of the number of replies",
		},
	)
)

func init() {
	prometheus.MustRegister(
		promCounterMacReplies,

		promGaugeIPAddresses,
		promGaugeUniqueMacAddresses,
		promGaugeMacAddressIPs,
		promGaugeIpMacAdddresses,
	)
}
