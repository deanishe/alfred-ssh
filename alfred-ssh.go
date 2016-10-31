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
	"sort"
	"strconv"
	"strings"

	"github.com/havoc-io/ssh_config"
)

var (
	knownHostsPath = os.ExpandEnv("$HOME/.ssh/known_hosts")
	etcHostsPath   = "/etc/hosts"
	// Providers contains all registered providers of Hosts
	prov *Providers
	// providers map[string]Provider
	// disabled contains the names of disabled providers
	disabled map[string]bool
)

func init() {
	// providers = make(map[string]Provider, 4)
	prov = NewProviders()
	// disabled = map[string]bool{}
	Register(&providerWrapper{name: "config", fn: readConfig, priority: 1})
	Register(&providerWrapper{name: "known_hosts", fn: readKnownHosts, priority: 3})
	Register(&providerWrapper{name: "/etc/config", fn: readEtcConfig, priority: 5})
	Register(&providerWrapper{name: "/etc/hosts", fn: readEtcHosts, priority: 4})
}

// --------------------------------------------------------------------
// Data models
// --------------------------------------------------------------------

// Providers is a prioritised list of providers
type Providers struct {
	disabled  map[string]bool
	providers []Provider
	dirty     bool
}

// NewProviders returns an initialised Providers
func NewProviders() *Providers {
	return &Providers{
		disabled:  map[string]bool{},
		providers: []Provider{},
	}
}

// Len implements sort.Interface.
func (p Providers) Len() int {
	return len(p.providers)
}

// Less implements sort.Interface.
func (p Providers) Less(i, j int) bool {
	return p.providers[i].Priority() < p.providers[j].Priority()
}

// Swap implements sort.Interface.
func (p Providers) Swap(i, j int) {
	p.providers[i], p.providers[j] = p.providers[j], p.providers[i]
}

// Register adds a Provider.
func (p *Providers) Register(pv Provider) {
	p.providers = append(p.providers, pv)
	p.dirty = true
}

// Disable turns a Provider off.
func (p *Providers) Disable(name string) {
	for _, pv := range p.providers {
		if pv.Name() == name {
			p.disabled[name] = true
			return
		}
	}
	log.Printf("Unknown provider: %s", name)
}

// Get returns Providers ordered by priority.
func (p *Providers) Get() []Provider {
	if p.dirty {
		sort.Sort(p)
		p.dirty = false
	}
	pvs := []Provider{}
	for _, pv := range p.providers {
		if off := p.disabled[pv.Name()]; !off {
			pvs = append(pvs, pv)
		}
	}
	return pvs
}

// Provider is a provider of Hosts.
type Provider interface {
	Name() string
	Hosts() []*Host
	Priority() int
}

// Disable disables a Provider.
func Disable(name string) {
	prov.Disable(name)
}

// Disabled returns true if a Provider is disabled.
func Disabled(name string) bool {
	return prov.disabled[name]
}

// Register registers a Provider.
func Register(p Provider) {
	prov.Register(p)
}

// providerWrapper implements Provider. It allows construction of
// providers from other objects. fn() is called once to fetch Hosts,
// which are cached in hosts.
type providerWrapper struct {
	name     string
	fn       func() []*Host
	hosts    []*Host
	called   bool
	priority int
}

// Name implements Provider.
func (pw *providerWrapper) Name() string { return pw.name }

// Hosts implements Provider.
func (pw *providerWrapper) Hosts() []*Host {
	if pw.called {
		return pw.hosts
	}
	pw.hosts = pw.fn()
	pw.called = true
	return pw.hosts
}

func (pw *providerWrapper) Priority() int { return pw.priority }

// Host is computer that may be connected to.
type Host struct {
	Name     string `json:"name"`
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

// NewHostFromURL creates a Host from an ssh:// URL.
func NewHostFromURL(URL string) (*Host, error) {
	u, err := url.Parse(URL)
	if err != nil {
		return nil, err
	}
	name := u.Host
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
		name = u.User.Username() + "@" + name
		h.Username = u.User.Username()
	}
	h.Name = name
	return h, nil
}

// FullName is the name of the connection, i.e. [user@]hostname[:port]
func (h *Host) FullName() string {
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

// Mosh returns a mosh command for the host.
func (h *Host) Mosh(path string) string {
	if path == "" {
		path = "mosh"
	}
	cmd := path + " "
	if h.Port != 22 {
		cmd += fmt.Sprintf("--ssh 'ssh -p %d' ", h.Port)
	}
	if h.Username != "" {
		cmd += h.Username + "@"
	}
	cmd += h.Hostname
	return cmd
}

// UID returns a unique ID for Host.
func (h *Host) UID() string { return h.Name + " | " + h.URL() }

// History is a list of previously opened URLs.
type History struct {
	Path     string
	hosts    []*Host
	priority int
}

// NewHistory initialises a new History struct. You must call History.Load()
// to load cached data.
func NewHistory(path string) *History {
	return &History{Path: path, hosts: []*Host{}, priority: 2}
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

	log.Printf("Adding %s to history ...", host.FullName())

	return h.Save()
}

// Remove removes an item from the History.
func (h *History) Remove(URL string) error {
	for i, host := range h.hosts {
		if host.URL() == URL {
			h.hosts = append(h.hosts[0:i], h.hosts[i+1:]...)
			log.Printf("Removed %s from history", host.FullName())
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

// Priority implements Provider.
func (h *History) Priority() int { return h.priority }

// Load loads the history from disk.
func (h *History) Load() error {
	if _, err := os.Stat(h.Path); err != nil {
		// log.Println("0 hosts in history")
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

		hosts = append(hosts, &Host{Name: hostname, Hostname: hostname, Port: port})
	}

	return hosts
}

// readConfig reads hostnames from ~/.ssh/config.
func readConfig() []*Host {
	return parseConfig(os.ExpandEnv("$HOME/.ssh/config"), "~/.ssh/config")
}

// readEtcConfig reads hostnames from /etc/ssh/ssh_config.
func readEtcConfig() []*Host {
	return parseConfig(os.ExpandEnv("/etc/ssh/ssh_config"), "/etc/ssh/ssh_config")
}

// parseConfig parse an SSH config file.
func parseConfig(path, source string) []*Host {
	var host *Host
	var hosts []*Host
	r, err := os.Open(path)
	if err != nil {
		log.Printf("Error opening file `%s`: %s", path, err)
		return hosts
	}
	cfg, err := ssh_config.Parse(r)
	if err != nil {
		log.Printf("Error parsing `%s`: %s", path, err)
		return hosts
	}

	for _, e := range cfg.Hosts {
		var port = 22
		var p *ssh_config.Param
		var name string
		var user string

		p = e.GetParam(ssh_config.HostKeyword)
		if p != nil {
			name = p.Value()
		}

		// log.Println(e.String())
		// log.Printf("hostnames=%v", e.Hostnames)

		p = e.GetParam(ssh_config.HostNameKeyword)
		if p != nil {
			name = p.Value()
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

			host = NewHost()
			host.Name = n
			if name != "" {
				host.Hostname = name
			} else {
				host.Hostname = n
			}
			host.Port = port
			host.Username = user
			host.Source = source
			// log.Printf("%+v", host)
			hosts = append(hosts, host)
		}
	}
	return hosts
}

// readKnowHosts reads hostnames from ~/.ssh/known_hosts.
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
			hosts = append(hosts, &Host{s, s, 22, "/etc/hosts", ""})
		}
	}

	// log.Printf("%d host(s) in /etc/hosts", len(hosts))
	return hosts
}

// Hosts loads Hosts from active providers. Duplicates are removed.
func Hosts() []*Host {
	hosts := []*Host{}
	seen := map[string]bool{}

	for _, p := range prov.Get() {

		i := 0
		j := 0
		for _, h := range p.Hosts() {

			uid := h.UID()

			if _, dupe := seen[uid]; !dupe {
				hosts = append(hosts, h)
				seen[uid] = true
				i++
			} else {
				// log.Printf("Dupe : %s", uid)
				j++
			}
		}
		log.Printf("Loaded %d host(s) from '%s', ignored %d dupe(s)", i, p.Name(), j)
	}

	return hosts
}
