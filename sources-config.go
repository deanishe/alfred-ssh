//
// Copyright (c) 2016 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2016-12-11
//

package ssh

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/danieljimenez/ssh_config"
)

// ConfigHost is Host parsed from SSH config-format files.
type ConfigHost struct {
	BaseHost
	forcePort     bool
	forceUsername bool
}

// UID implements Host.
func (h *ConfigHost) UID() string { return UIDForHost(h) }

// SetPort implements Host.
func (h *ConfigHost) SetPort(i int) {
	h.port = i
	h.forcePort = true
}

// SetUsername implemeents Host.
func (h *ConfigHost) SetUsername(n string) {
	h.username = n
	h.forceUsername = true
}

// SSHURL returns a URL based on the Host value from the config file,
// *not* the Hostname.
func (h *ConfigHost) SSHURL() *url.URL {
	u := &url.URL{
		Scheme: "ssh",
		Host:   h.Name(),
	}
	if h.forcePort {
		u.Host = fmt.Sprintf("%s:%d", u.Host, h.Port())
	}
	if h.forceUsername {
		u.User = url.User(h.Username())
	}
	return u
}

// MoshCmd implements Host.
func (h *ConfigHost) MoshCmd(path string) string {
	if path == "" {
		path = "mosh"
	}
	cmd := path + " "
	if h.forcePort {
		cmd += fmt.Sprintf("--ssh 'ssh -p %d' ", h.Port())
	}
	if h.forceUsername && h.Username() != "" {
		cmd += h.Username() + "@"
	}
	cmd += h.Name()
	return cmd
}

// ConfigSource implements Source for ssh config-formatted files.
type ConfigSource struct {
	baseSource
}

// NewConfigSource creates a new ConfigSource from an ssh configuration file.
func NewConfigSource(path, name string, priority int) *ConfigSource {
	s := &ConfigSource{}
	s.Filepath = path
	s.name = name
	s.priority = priority
	return s
}

// Hosts implements Source.
func (s *ConfigSource) Hosts() []Host {
	if s.hosts == nil {
		hosts := parseConfigFile(s.Filepath)
		log.Printf("[source/load/config] %d host(s) in '%s'", len(hosts), s.Name())
		s.hosts = make([]Host, len(hosts))
		for i, h := range hosts {
			h.source = s.Name()
			s.hosts[i] = Host(h)
		}
	}
	return s.hosts
}

// parseConfigFile parse an SSH config file.
func parseConfigFile(path string) []*ConfigHost {
	var hosts []*ConfigHost
	r, err := os.Open(path)
	if err != nil {
		log.Printf("[config/%s] Error opening file: %s", path, err)
		return hosts
	}
	cfg, err := ssh_config.Parse(r)
	if err != nil {
		log.Printf("[config/%s] Parse error: %s", path, err)
		return hosts
	}

	for _, e := range cfg.Hosts {
		var (
			p    *ssh_config.Param
			port = 22
			hn   string // hostname
			user string
		)

		p = e.GetParam(ssh_config.HostKeyword)
		if p != nil {
			hn = p.Value()
		}

		// log.Println(e.String())
		// log.Printf("hostnames=%v", e.Hostnames)

		p = e.GetParam(ssh_config.HostNameKeyword)
		if p != nil {
			hn = p.Value()
		}

		p = e.GetParam(ssh_config.PortKeyword)
		if p != nil {
			port, err = strconv.Atoi(p.Value())
			if err != nil {
				log.Printf("Bad port: %s", err)
				port = 22
			}
		}
		// log.Printf("port=%v", port)

		p = e.GetParam(ssh_config.UserKeyword)
		if p != nil {
			user = p.Value()
		}

		for _, n := range e.Hostnames {
			if strings.Contains(n, "*") || strings.Contains(n, "!") || strings.Contains(n, "?") {
				continue
			}

			h := &ConfigHost{}
			h.name = n
			h.hostname = n
			h.port = port
			h.username = user

			if hn != "" {
				h.hostname = hn
			}
			// log.Printf("%+v", host)
			hosts = append(hosts, h)
		}
	}
	return hosts
}
