//
// Copyright (c) 2016 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2016-12-11
//

package ssh

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"
)

// KnownSource implements Source for a known_hosts-formatted file.
type KnownSource struct {
	baseSource
}

// NewKnownSource creates a new HostsSource for a hosts-formatted file.
func NewKnownSource(path, name string, priority int) *KnownSource {
	s := &KnownSource{}
	s.Filepath = path
	s.name = name
	s.priority = priority
	return s
}

// Hosts implements Source.
func (s *KnownSource) Hosts() []Host {
	if s.hosts == nil {
		hosts := readKnownHostsFile(s.Filepath)
		log.Printf("[source/load/known_hosts] %d host(s) in '%s'", len(hosts), s.Name())
		s.hosts = make([]Host, len(hosts))
		for i, h := range hosts {
			h.source = s.Name()
			s.hosts[i] = Host(h)
		}
	}
	return s.hosts
}

// readKnownHostsFile reads hostnames from ~/.ssh/known_hosts.
func readKnownHostsFile(path string) []*BaseHost {
	var hosts []*BaseHost

	fp, err := os.Open(path)
	if err != nil {
		log.Printf("[known_hosts/%s] Error opening file: %v", path, err)
		return hosts
	}
	defer fp.Close()

	scanner := bufio.NewScanner(fp)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		line := scanner.Text()
		for _, host := range parseKnownHostsLine(line, path) {
			hosts = append(hosts, host)
		}
	}
	return hosts
}

// parseKnownHostsLine extracts the host(s) from a single line in
// ~/.ssh/know_hosts.
func parseKnownHostsLine(line, path string) []*BaseHost {
	var hosts []*BaseHost
	var hostnames []string

	// Split line on first whitespace. First element is hostname(s),
	// second is the key.
	i := strings.Index(line, " ")
	if i < 0 {
		return hosts
	}

	line = line[:i]

	// Split hostname on comma. Some entries are of format hostname,ip.
	hostnames = append(hostnames, strings.Split(line, ",")...)

	// Parse the found hostnames to see if any specify a non-default
	// port. Such entries look like [host.name.here]:NNNN instead of
	// host.name.only
	var port int

	for _, hostname := range hostnames {

		port = 22

		if strings.HasPrefix(hostname, "[") {
			// Assume [ip.addr.goes.here]:NNNN
			i = strings.Index(hostname, "]:")
			if i < 0 {
				log.Printf("[known_hosts/%s] Don't understand hostname : %s", path, hostname)
				continue
			}

			p, err := strconv.Atoi(hostname[i+2:])
			if err != nil {
				log.Printf("[known_hosts/%s] Error parsing hostname `%v` : %v", path, hostname, err)
				continue
			}

			port = p
			hostname = hostname[1:i]
		}

		if !IsValidHostname(hostname) {
			log.Printf("[known_host] Invalid hostname: %s", hostname)
			continue
		}

		hosts = append(hosts, &BaseHost{name: hostname, hostname: hostname, port: port})
	}

	return hosts
}
