package arping

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHostsList(t *testing.T) {
	ips, count, err := getCidrIpAddresses("10.25.0.0/23")
	assert.NoError(t, err, "cidr block should not error")
	assert.Equal(t, 510, len(ips), "expect ips to contain 510 entries")
	assert.Equal(t, count, len(ips), "expect that ips should be the same length as count")
	assert.Equal(t, "10.25.0.10", ips[9].To4().String(), "10th element should be 10.25.0.10")

	ips, count, _ = getCidrIpAddresses("127.0.0.1/32")
	assert.Equal(t, 1, len(ips), "expect to get a single IP address")
	assert.Equal(t, count, len(ips), "expect lenght and count to be the same")
	assert.Equal(t, "127.0.0.1", ips[0].To4().String())
}
