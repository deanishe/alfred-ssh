//
// Copyright (c) 2016 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2016-05-27
//

package assh

import "testing"

var knownHostsTests = []struct {
	Line     string
	Expected []*Host
}{
	// Empty line
	{"", []*Host{}},
	// Invalid line
	{"nowhitespace", []*Host{}},
	// Simple hostname
	{"localhost ssh-rsa AAAA",
		[]*Host{&Host{Hostname: "localhost", Port: 22}}},
	// Domain
	{"github.com ssh-rsa AAAA",
		[]*Host{&Host{Hostname: "github.com", Port: 22}}},
	// Subdomain
	{"gist.github.com ssh-rsa AAAA",
		[]*Host{&Host{Hostname: "gist.github.com", Port: 22}}},
	// IP address
	{"127.0.0.1 ssh-rsa AAAA",
		[]*Host{&Host{Hostname: "127.0.0.1", Port: 22}}},
	// IP address with port
	{"[8.8.8.8]:1234 ssh-rsa AAAA",
		[]*Host{&Host{Hostname: "8.8.8.8", Port: 1234}}},
	// Hostname with port
	{"[printer.clintonmail.com]:1234 ssh-rsa AAAA",
		[]*Host{&Host{Hostname: "printer.clintonmail.com", Port: 1234}}},
	// Hostname and IP
	{"machine.example.com,10.0.0.1 ecdsa-sha2-nistp256 AAAA",
		[]*Host{
			&Host{Hostname: "machine.example.com", Port: 22},
			&Host{Hostname: "10.0.0.1", Port: 22},
		}},
	// Hostname, IPv4 and IPv6
	{"::1,127.0.0.1,localhost ecdsa-sha2-nistp256 AAAA",
		[]*Host{
			&Host{Hostname: "::1", Port: 22},
			&Host{Hostname: "127.0.0.1", Port: 22},
			&Host{Hostname: "localhost", Port: 22},
		}},
}

// TestParseKnownHosts tests parsing of known_hosts lines
func TestParseKnownHosts(t *testing.T) {
	for i, kh := range knownHostsTests {
		hosts := parseKnownHostsLine(kh.Line)
		if len(hosts) != len(kh.Expected) {
			t.Errorf("[%d] Expected %d hosts, got %d: %s", i+1, len(kh.Expected), len(hosts), kh.Line)
			continue
		}

		// Test individual Hosts
		for j, h := range hosts {
			x := kh.Expected[j]
			if h.Hostname != x.Hostname {
				t.Errorf("[%d.%d] Expected=%v, Got=%v: %s", i+1, j+1, x.Hostname, h.Hostname, kh.Line)
			}
		}
	}
}

var hostnameTests = []struct {
	Hostname string
	Expected bool
}{
	// Plain old hostnames and IPs
	{"localhost", true},
	{"::1", true},
	{"127.0.0.1", true},
	{"google.com", true},
	{"host.google.com", true},
	// Invalid
	// With port
	{"host.google.com:22", false},
	{"127.0.0.1:22", false},
	{"[::1]:22", false},
	// Bad hostnames
	{"host_google_com", false},
	{"host google com", false},
	{"host google com:22", false},
}

// TestValidHostname tests validHostname
func TestValidHostname(t *testing.T) {
	for i, ht := range hostnameTests {
		v := validHostname(ht.Hostname)
		if v != ht.Expected {
			t.Errorf("[%d] Expected=%v, Got=%v: %s", i+1, ht.Expected, v, ht.Hostname)
		}
	}
}
