//
// Copyright (c) 2016 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2016-03-25
//

// Package assh reads SSH hosts from ~/.ssh/known_hosts and /etc/hosts.
package assh

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var (
	knownHostsPath = os.ExpandEnv("$HOME/.ssh/known_hosts")
	etcHostsPath   = "/etc/hosts"
	// Providers contains all registered providers of Hosts
	Providers map[string]Provider
	// disabled contains the names of disabled providers
	disabled map[string]bool
)

func init() {
	Providers = make(map[string]Provider, 2)
	disabled = map[string]bool{}
	Register(&providerWrapper{name: "/etc/hosts", fn: readEtcHosts})
	Register(&providerWrapper{name: "known_hosts", fn: readKnownHosts})
}

// --------------------------------------------------------------------
// Data models
// --------------------------------------------------------------------

// Provider is a provider of Hosts.
type Provider interface {
	Name() string
	Hosts() []*Host
}

// Disable disables a Provider.
func Disable(name string) {
	if p := Providers[name]; p != nil {
		disabled[name] = true
	} else {
		log.Printf("Unknown provider: %s", name)
	}
}

// Disabled returns true if a Provider is disabled.
func Disabled(name string) bool {
	return disabled[name]
}

// Register registers a Provider.
func Register(p Provider) {
	Providers[p.Name()] = p
}

type providerWrapper struct {
	name   string
	fn     func() []*Host
	hosts  []*Host
	called bool
}

// Name implements Provider.
func (pw *providerWrapper) Name() string {
	return pw.name
}

// Hosts implements Provider.
func (pw *providerWrapper) Hosts() []*Host {
	if pw.called {
		return pw.hosts
	}
	pw.hosts = pw.fn()
	pw.called = true
	return pw.hosts
}

// Host is computer that may be connected to.
type Host struct {
	Hostname string `json:"host"`
	Port     int    `json:"port"`
	// Name of the source, e.g. "known_hosts"
	Source   string `json:"source"`
	Username string `json:"username"`
}

// NewHost initialises a Host with port set to 22.
func NewHost() *Host {
	return &Host{Port: 22}
}

// NewHostFromURL creates a Host for an ssh:// URL.
func NewHostFromURL(URL string) (*Host, error) {
	u, err := url.Parse(URL)
	if err != nil {
		return nil, err
	}
	h := &Host{Source: "URL", Port: 22}
	if i := strings.Index(u.Host, ":"); i > -1 { // URL contains port
		h.Hostname = u.Host[:i]
		if j, err := strconv.Atoi(u.Host[i+1:]); err == nil {
			h.Port = j
		}
	} else {
		h.Hostname = u.Host
	}
	if u.User != nil {
		h.Username = u.User.Username()
	}
	return h, nil
}

// Name is the name of the connection, i.e. [user@]hostname[:port]
func (h *Host) Name() string {
	n := ""
	if h.Username != "" {
		n = fmt.Sprintf("%s@", h.Username)
	}
	n = n + h.Hostname
	if h.Port != 22 {
		n = n + fmt.Sprintf(":%d", h.Port)
	}
	return n
}

// URL returns the ssh:// URL for the host.
func (h *Host) URL() string {
	return h.URLForProtocol("ssh")
}

// URLForProtocol returns <proto>:// URL for the host.
func (h *Host) URLForProtocol(proto string) string {
	var url, prefix string
	if h.Username != "" {
		prefix = fmt.Sprintf("%s://%s@", proto, h.Username)
	} else {
		prefix = fmt.Sprintf("%s://", proto)
	}
	if h.Port == 22 {
		url = fmt.Sprintf("%s%s", prefix, h.Hostname)
	} else {
		// url = fmt.Sprintf("%s[%s]:%d", prefix, h.Hostname, h.Port)
		url = fmt.Sprintf("%s%s:%d", prefix, h.Hostname, h.Port)
	}
	return url
}

// SFTP returns the sftp:// URL for the host.
func (h *Host) SFTP() string {
	return h.URLForProtocol("sftp")
}

// History is a list of previously opened URLs.
type History struct {
	Path  string
	hosts []*Host
}

// NewHistory initialises a new History struct. You must call History.Load()
// to load cached data.
func NewHistory(path string) *History {
	return &History{Path: path, hosts: []*Host{}}
}

// Add adds an item to the History.
func (h *History) Add(URL string) error {
	for _, host := range h.hosts {
		if host.URL() == URL {
			log.Printf("[History] Ignoring duplicate: %s", URL)
			return nil
		}
	}
	host, err := NewHostFromURL(URL)
	if err != nil {
		return err
	}
	if host.Username == "" {
		log.Printf("Not adding connection without username to history: %v", URL)
		return nil
	}
	host.Source = h.Name()
	h.hosts = append(h.hosts, host)

	log.Printf("Adding %s to history ...", host.Name())

	return h.Save()
}

// Remove removes an item from the History.
func (h *History) Remove(URL string) error {
	for i, host := range h.hosts {
		if host.URL() == URL {
			h.hosts = append(h.hosts[0:i], h.hosts[i+1:]...)
			log.Printf("Removed %s from history", host.Name())
			return h.Save()
		}
	}
	log.Printf("Item not in history: %s", URL)
	return nil
}

// Hosts returns all the Hosts in History.
func (h *History) Hosts() []*Host {
	return h.hosts
}

// Name implements Provider.
func (h *History) Name() string {
	return "history"
}

// Load loads the history from disk.
func (h *History) Load() error {
	if _, err := os.Stat(h.Path); err != nil {
		log.Println("0 hosts in history")
		return nil
	}

	urls := []string{}
	data, err := ioutil.ReadFile(h.Path)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, &urls)
	if err != nil {
		return err
	}
	for _, u := range urls {
		if host, err := NewHostFromURL(u); err != nil {
			log.Printf("Error loading URL %s: %s", u, err)
		} else {
			host.Source = "history"
			h.hosts = append(h.hosts, host)
		}
	}
	// log.Printf("%d host(s) in history", len(h.hosts))
	return nil
}

// Save saves the History to disk.
func (h *History) Save() error {
	urls := []string{}

	for _, host := range h.hosts {
		urls = append(urls, host.URL())
	}

	data, err := json.MarshalIndent(urls, "", "  ")
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(h.Path, data, 0600); err != nil {
		return err
	}

	log.Printf("Saved %d host(s) to history", len(h.hosts))
	return nil
}

// RegisterHistory is a convenience method to create and register a History.
func RegisterHistory(path string) (*History, error) {
	h := NewHistory(path)
	if err := h.Load(); err != nil {
		return nil, err
	}
	Register(h)
	return h, nil
}

// --------------------------------------------------------------------
// Load data
// --------------------------------------------------------------------

var (
	hostnameRegex = regexp.MustCompile("^[a-zA-Z0-9.-]+$")
)

// validHostname returns true if n is an IP address or hostname.
func validHostname(n string) bool {
	if ip := net.ParseIP(n); ip != nil {
		return true
	}
	return hostnameRegex.MatchString(n)
}

// parseKnownHostsLine extracts the host(s) from a single line in
// ~/.ssh/know_hosts.
func parseKnownHostsLine(line string) []*Host {
	var hosts []*Host
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
				log.Printf("Don't understand hostname : %s", hostname)
				continue
			}

			p, err := strconv.Atoi(hostname[i+2:])
			if err != nil {
				log.Printf("Error parsing hostname `%v` : %v", hostname, err)
				continue
			}

			port = p
			hostname = hostname[1:i]
		}

		if !validHostname(hostname) {
			log.Printf("Invalid hostname: %s", hostname)
			continue
		}

		hosts = append(hosts, &Host{Hostname: hostname, Port: port})
	}

	return hosts
}

// readKnowHosts reads hostnames from ~/.ssh/know_hosts.
func readKnownHosts() []*Host {
	var hosts []*Host

	fp, err := os.Open(knownHostsPath)
	if err != nil {
		log.Printf("Error opening ~/.ssh/known_hosts : %v", err)
		return hosts
	}
	defer fp.Close()

	scanner := bufio.NewScanner(fp)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		line := scanner.Text()
		for _, host := range parseKnownHostsLine(line) {
			host.Source = "~/.ssh/known_hosts"
			hosts = append(hosts, host)
		}
	}

	// log.Printf("%d host(s) in ~/.ssh/known_hosts", len(hosts))
	return hosts
}

// readEtcHosts reads hostnames from /etc/hosts.
func readEtcHosts() []*Host {
	var hosts []*Host

	fp, err := os.Open(etcHostsPath)
	if err != nil {
		log.Printf("Error reading /etc/hosts : %v", err)
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
			log.Printf("Bad IP address : %v", fields[0])
			continue
		}

		// All other fields are hostnames
		for _, s := range fields[1:] {
			hosts = append(hosts, &Host{s, 22, "/etc/hosts", ""})
		}
	}

	// log.Printf("%d host(s) in /etc/hosts", len(hosts))
	return hosts
}

// Hosts loads Hosts from active providers. Duplicates are removed.
func Hosts() []*Host {
	hosts := []*Host{}
	seen := make(map[string]bool, 10)

	for n, p := range Providers {
		if Disabled(n) {
			log.Printf("Ignoring disabled provider '%s'", n)
			continue
		}

		i := 0
		j := 0
		for _, h := range p.Hosts() {

			u := h.URL()

			if _, dupe := seen[u]; !dupe {
				hosts = append(hosts, h)
				seen[u] = true
				i++
			} else {
				j++
			}

		}
		log.Printf("Loaded %d host(s) from '%s', ignored %d dupe(s)", i, n, j)
	}

	return hosts
}
