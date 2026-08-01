package gencfg

import (
	"errors"
	"net"
	"reflect"
	"testing"
)

func TestGetIPv4UsesHostnameLookup(t *testing.T) {
	var lookups []string
	stubNetwork(t,
		func(host string) ([]net.IP, error) {
			lookups = append(lookups, host)
			return []net.IP{net.ParseIP("192.0.2.10")}, nil
		},
		func() ([]net.Interface, error) {
			t.Fatal("network interfaces should not be enumerated after hostname lookup succeeds")
			return nil, nil
		},
		nil,
	)

	got, err := getIPv4("example-host")
	if err != nil {
		t.Fatalf("getIPv4() error: %v", err)
	}
	if got != "192.0.2.10" {
		t.Fatalf("getIPv4() = %q, want %q", got, "192.0.2.10")
	}
	if want := []string{"example-host"}; !reflect.DeepEqual(lookups, want) {
		t.Fatalf("lookups = %v, want %v", lookups, want)
	}
}

func TestGetIPv4FallsBackToLocalHostname(t *testing.T) {
	var lookups []string
	stubNetwork(t,
		func(host string) ([]net.IP, error) {
			lookups = append(lookups, host)
			if host == "example-host.local" {
				return []net.IP{net.ParseIP("192.0.2.20")}, nil
			}
			return nil, errors.New("lookup failed")
		},
		func() ([]net.Interface, error) {
			t.Fatal("network interfaces should not be enumerated after .local lookup succeeds")
			return nil, nil
		},
		nil,
	)

	got, err := getIPv4("example-host")
	if err != nil {
		t.Fatalf("getIPv4() error: %v", err)
	}
	if got != "192.0.2.20" {
		t.Fatalf("getIPv4() = %q, want %q", got, "192.0.2.20")
	}
	if want := []string{"example-host", "example-host.local"}; !reflect.DeepEqual(lookups, want) {
		t.Fatalf("lookups = %v, want %v", lookups, want)
	}
}

func TestGetIPv4FallsBackToLocalInterface(t *testing.T) {
	stubNetwork(t,
		func(string) ([]net.IP, error) {
			return nil, errors.New("lookup failed")
		},
		func() ([]net.Interface, error) {
			return []net.Interface{
				{Name: "lo0", Flags: net.FlagUp | net.FlagLoopback},
				{Name: "en0", Flags: net.FlagUp},
			}, nil
		},
		func(iface net.Interface) ([]net.Addr, error) {
			if iface.Name == "en0" {
				return []net.Addr{mustCIDR(t, "192.0.2.30/24")}, nil
			}
			return []net.Addr{mustCIDR(t, "127.0.0.1/8")}, nil
		},
	)

	got, err := getIPv4("example-host")
	if err != nil {
		t.Fatalf("getIPv4() error: %v", err)
	}
	if got != "192.0.2.30" {
		t.Fatalf("getIPv4() = %q, want %q", got, "192.0.2.30")
	}
}

func stubNetwork(
	t *testing.T,
	lookup func(string) ([]net.IP, error),
	interfaces func() ([]net.Interface, error),
	addrs func(net.Interface) ([]net.Addr, error),
) {
	t.Helper()

	originalLookupIP := lookupIP
	originalNetworkInterfaces := networkInterfaces
	originalInterfaceAddrs := interfaceAddrs

	lookupIP = lookup
	networkInterfaces = interfaces
	if addrs != nil {
		interfaceAddrs = addrs
	}

	t.Cleanup(func() {
		lookupIP = originalLookupIP
		networkInterfaces = originalNetworkInterfaces
		interfaceAddrs = originalInterfaceAddrs
	})
}

func mustCIDR(t *testing.T, cidr string) *net.IPNet {
	t.Helper()

	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		t.Fatalf("net.ParseCIDR(%q) error: %v", cidr, err)
	}
	ipNet.IP = ip

	return ipNet
}
