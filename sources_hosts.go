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
	"net"
	"os"
	"strings"
)

// HostsSource implements Source for a hosts-formatted file.
type HostsSource struct {
	baseSource
}

// NewHostsSource creates a new HostsSource for a hosts-formatted file.
func NewHostsSource(path, name string, priority int) *HostsSource {
	s := &HostsSource{}
	s.Filepath = path
	s.name = name
	s.priority = priority
	return s
}

// Hosts implements Source.
func (s *HostsSource) Hosts() []Host {
	if s.hosts == nil {
		hosts := readHostsFile(s.Filepath)
		log.Printf("[source/load/hosts] %d host(s) in '%s'", len(hosts), s.Name())
		s.hosts = make([]Host, len(hosts))
		for i, h := range hosts {
			h.source = s.Name()
			s.hosts[i] = Host(h)
		}
	}
	return s.hosts
}

// readHostsFile reads hostnames from hosts-formatted path.
func readHostsFile(path string) []*BaseHost {
	var hosts []*BaseHost

	fp, err := os.Open(path)
	if err != nil {
		log.Printf("[hosts/%s] Error reading file : %v", path, err)
		return hosts
	}

	scanner := bufio.NewScanner(fp)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {

		line := scanner.Text()

		// Strip comments
		if i := strings.Index(line, "#"); i > -1 {
			line = line[:i]
		}

		// Ignore empty lines
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse fields
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if net.ParseIP(fields[0]) == nil {
			log.Printf("[hosts/%s] Invalid IP address : %v", path, fields[0])
			continue
		}

		// All other fields are hostnames
		for _, s := range fields[1:] {
			if s == "broadcasthost" {
				continue
			}
			h := &BaseHost{name: s, hostname: s}
			hosts = append(hosts, h)
		}
	}

	return hosts
}
