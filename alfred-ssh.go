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
	"path/filepath"
	"strconv"
	"strings"

	"gogs.deanishe.net/deanishe/awgo"
)

var (
	// minFuzzyScore is the default cut-off for search results
	minFuzzyScore = 30.0
	usage         = `alfssh [options] [<query>]

Display a list of know SSH hosts in Alfred 3. If <query>
is specified, the hostnames will be filtered against it.

Usage:
    alfssh search [-d] [<query>]
    alfssh remember <url>
    alfssh print (datadir|cachedir|distname|logfile)
    alfssh --help|--version

Options:
    -h, --help        Show this message and exit.
    --version         Show version information and exit.
    -d, --demo        Use fake test data instead of real data from the computer.
                      Useful for testing, otherwise pointless. Demo mode can also
                      turned on by setting the environment variable DEMO_MODE=1
`
	// knownHostsPath string
	knownHostsPath = os.ExpandEnv("$HOME/.ssh/known_hosts")
	etcHostsPath   = "/etc/hosts"
	wf             *workflow.Workflow
)

func init() {
	wf = workflow.NewWorkflow(nil)
}

// --------------------------------------------------------------------
// Data models
// --------------------------------------------------------------------

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
		url = fmt.Sprintf("%s[%s]:%d", prefix, h.Hostname, h.Port)
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
			log.Printf("Ignoring duplicate: %s", URL)
			return nil
		}
	}
	host, err := NewHostFromURL(URL)
	if err != nil {
		return err
	}
	if host.Username == "" {
		log.Printf("Not saving connection without username: %v", URL)
		return nil
	}
	host.Source = "history"
	h.hosts = append(h.hosts, host)

	log.Printf("Saving %s ...", host.Name())

	return h.Save()
}

// Hosts returns all the Hosts in History.
func (h *History) Hosts() []*Host {
	return h.hosts
}

// Load loads the history from disk.
func (h *History) Load() error {
	if !workflow.PathExists(h.Path) {
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

	log.Printf("%d item(s) in history", len(h.hosts))
	return nil
}

// --------------------------------------------------------------------
// Load data
// --------------------------------------------------------------------

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
	i = strings.Index(line, ",")

	if i > -1 {
		hostnames = append(hostnames, strings.TrimSpace(line[0:i]))
		hostnames = append(hostnames, strings.TrimSpace(line[i+1:]))
	} else {
		hostnames = append(hostnames, strings.TrimSpace(line))
	}

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

		hosts = append(hosts, &Host{hostname, port, "~/.ssh/known_hosts", ""})
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
			hosts = append(hosts, host)
		}
	}

	log.Printf("%d hosts in ~/.ssh/known_hosts", len(hosts))
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

	log.Printf("%d hosts in /etc/hosts", len(hosts))
	return hosts
}

// Hosts loads hosts from all sources. Duplicates are removed.
func Hosts() []*Host {
	hosts := []*Host{}

	seen := make(map[string]bool, 10)

	// ~/.ssh/known_hosts
	for _, h := range readKnownHosts() {

		url := h.URL()

		if _, dupe := seen[url]; !dupe {
			hosts = append(hosts, h)
			seen[url] = true
		}
	}

	// /etc/hosts
	for _, h := range readEtcHosts() {

		url := h.URL()

		if _, dupe := seen[url]; !dupe {
			hosts = append(hosts, h)
			seen[url] = true
		}
	}

	// History
	h := NewHistory(filepath.Join(wf.DataDir(), "history.json"))
	if err := h.Load(); err != nil {
		log.Printf("Error loading history: %v", err)
	} else {
		for _, h := range h.hosts {

			url := h.URL()

			if _, dupe := seen[url]; !dupe {
				hosts = append(hosts, h)
				seen[url] = true
			}
		}
	}
	// sort.Sort(hosts)
	return hosts
}
