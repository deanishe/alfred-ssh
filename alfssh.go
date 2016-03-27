//
// Copyright (c) 2016 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2016-03-25
//

/*
alfssh.go
=========

A Script Filter for Alfred 2 for opening SSH connections. Auto-suggests
hosts from ~/.ssh/known_hosts and from /etc/hosts.

The script filter is implemented as a command-line program (that outputs
XML).
*/
package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/docopt/docopt-go"
	"gogs.deanishe.net/deanishe/awgo"
)

const (
	Version = "0.1"
)

var (
	usage = `alfssh [options] [<query>]

Display a list of know SSH hosts in Alfred. If <query>
is specified, the hostnames will be filtered against it.

Usage:
	alfssh [-t] [<query>]
	alfssh --datadir
	alfssh --cachedir
	alfssh --distname
	alfssh --help|--version

Options:
	-t, --test  Use fake test data instead of real data from the computer.
				Useful for testing, otherwise pointless.
	--datadir   Print path to workflow's data directory and exit.
	--cachedir  Print path to workflow's cache directory and exit.
	--distname  Print filename of distributable .alfredworkflow file (for
				the build script).
	-h, --help  Show this message and exit.
	--version   Show version information and exit.
`

	// knownHostsPath string
	knownHostsPath = os.ExpandEnv("$HOME/.ssh/known_hosts")
	etcHostsPath   = "/etc/hosts"
	// Useful for screenshots
	testHostnames = []string{
		"one.example.com",
		"two.example.com",
		"alpha.deanishe.net",
		"beta.deanishe.net",
		"charlie.deanishe.net",
		"delta.deanishe.net",
		"imap.example.com",
		"mail.example.com",
		"www.example.com",
		"ftp.example.com",
	}
)

// --------------------------------------------------------------------
// Data models
// --------------------------------------------------------------------

// Host is computer that may be connected to.
type Host struct {
	Hostname string
	Port     int
	// Name of the source, e.g. "known_hosts"
	Source string
}

// GetURL returns the ssh:// URL for the host.
func (h Host) GetURL(username string) string {
	var url, prefix string
	if username != "" {
		prefix = fmt.Sprintf("ssh://%s@", username)
	} else {
		prefix = "ssh://"
	}
	if h.Port == 22 {
		url = fmt.Sprintf("%s%s", prefix, h.Hostname)
	} else {
		url = fmt.Sprintf("%s[%s]:%d", prefix, h.Hostname, h.Port)
	}
	return url
}

// Hosts is a collection of Host objects that supports sort.Interface and
// workflow.Fuzzy
type Hosts []Host

// Len implements sort.Interface.
func (s Hosts) Len() int {
	return len(s)
}

// Less implements sort.Interface.
func (s Hosts) Less(i, j int) bool {
	return s[i].Hostname < s[j].Hostname
}

// Swap implements sort.Interface.
func (s Hosts) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Keywords implements workflow.Fuzzy.
func (s Hosts) Keywords(i int) string {
	return s[i].Hostname
}

// --------------------------------------------------------------------
// Load data
// --------------------------------------------------------------------

// parseKnownHostsLine extracts the host(s) from a single line in
// ~/.ssh/know_hosts.
func parseKnownHostsLine(line string) []Host {
	var hosts []Host
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
		hosts = append(hosts, Host{hostname, port, "~/.ssh/known_hosts"})
	}
	return hosts
}

// readKnowHosts reads hostnames from ~/.ssh/know_hosts.
func readKnownHosts() []Host {
	var hosts []Host
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
	log.Printf("%d hosts in ~/.ssh/known_hosts.", len(hosts))
	return hosts
}

// The next few functions are copied from the source of net/parse.go.
// Count occurrences in s of any bytes in t.
func countAnyByte(s string, t string) int {
	n := 0
	for i := 0; i < len(s); i++ {
		if strings.IndexByte(t, s[i]) >= 0 {
			n++
		}
	}
	return n
}

// Split s at any bytes in t.
func splitAtBytes(s string, t string) []string {
	a := make([]string, 1+countAnyByte(s, t))
	n := 0
	last := 0
	for i := 0; i < len(s); i++ {
		if strings.IndexByte(t, s[i]) >= 0 {
			if last < i {
				a[n] = string(s[last:i])
				n++
			}
			last = i + 1
		}
	}
	if last < len(s) {
		a[n] = string(s[last:])
		n++
	}
	return a[0:n]
}

// Split s on whitespace.
func getFields(s string) []string {
	return splitAtBytes(s, " \r\t\n")
}

// readEtcHosts reads hostnames from /etc/hosts.
func readEtcHosts() []Host {
	var hosts []Host
	// TODO: Parse /etc/hosts
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
		fields := getFields(line)
		if len(fields) < 2 {
			continue
		}
		if net.ParseIP(fields[0]) == nil {
			log.Printf("Bad IP address : %v", fields[0])
			continue
		}
		// All other fields are hostnames
		for _, s := range fields[1:] {
			hosts = append(hosts, Host{s, 22, "/etc/hosts"})
		}
	}
	log.Printf("%d hosts in /etc/hosts.", len(hosts))
	return hosts
}

// loadTestHosts loads fake test data instead of real hosts.
func loadTestHosts() Hosts {
	hosts := make(Hosts, len(testHostnames))
	for i, name := range testHostnames {
		hosts[i] = Host{name, 22, "test data"}
	}
	return hosts
}

// loadHosts loads hosts from all sources. Duplicates are removed.
func loadHosts() Hosts {
	var hosts Hosts
	seen := make(map[string]bool, 10)

	// ~/.ssh/known_hosts
	for _, h := range readKnownHosts() {
		url := h.GetURL("")
		if _, dupe := seen[url]; !dupe {
			hosts = append(hosts, h)
			seen[url] = true
		}
	}

	// /etc/hosts
	for _, h := range readEtcHosts() {
		url := h.GetURL("")
		if _, dupe := seen[url]; !dupe {
			hosts = append(hosts, h)
			seen[url] = true
		}
	}

	// Remove duplicates
	// log.Printf("%d hosts before dupe check.", len(hosts))
	// var key string
	// m := map[string]bool{}
	// for _, h := range hosts {
	// 	key = fmt.Sprintf("%s:%d", h.Hostname, h.Port)
	// 	if _, dupe := m[key]; !dupe {
	// 		hosts[len(m)] = h
	// 		m[key] = true
	// 	}
	// }
	// log.Printf("Removed %d duplicate hosts.", len(hosts)-len(m))
	// hosts = hosts[:len(m)]

	// sort.Sort(hosts)
	return hosts
}

// --------------------------------------------------------------------
// Execute Script Filter
// --------------------------------------------------------------------

// run executes the workflow.
func run() {
	var query, username string
	var hosts Hosts

	// Parse options --------------------------------------------------
	vstr := fmt.Sprintf("%s/%v (awgo/%v)", workflow.GetName(), Version,
		workflow.Version)

	args, err := docopt.Parse(usage, nil, true, vstr, false)
	if err != nil {
		log.Fatalf("Error parsing CLI options : %v", err)
	}
	log.Printf("args=%v", args)

	// ===================== Alternate actions ========================
	if args["--datadir"] == true {
		fmt.Println(workflow.GetDataDir())
		return
	}

	if args["--cachedir"] == true {
		fmt.Println(workflow.GetCacheDir())
		return
	}

	if args["--distname"] == true {
		name := strings.Replace(
			fmt.Sprintf("%s-%s.alfredworkflow", workflow.GetName(), Version),
			" ", "-", -1)
		fmt.Println(name)
		return
	}

	// ====================== Script Filter ===========================

	// Parse query ----------------------------------------------------
	if args["<query>"] == nil {
		query = ""
	} else {
		query = fmt.Sprintf("%v", args["<query>"])
	}

	// Extract username if there is one
	if i := strings.Index(query, "@"); i > -1 {
		username, query = query[:i], query[i+1:]
		log.Printf("username=%v", username)
	}
	log.Printf("query=%v", query)

	// Load hosts -----------------------------------------------------
	if args["--test"] == true {
		hosts = loadTestHosts()
	} else {
		hosts = loadHosts()
	}

	// Add Host for query if it makes sense
	if query != "" {
		hosts = append(hosts, Host{query, 22, "user input"})
	}

	totalHosts := len(hosts)
	log.Printf("Loaded %d hosts.", totalHosts)

	// Filter hosts ---------------------------------------------------
	if query != "" {
		// var matches Hosts
		for i, score := range workflow.SortFuzzy(hosts, query) {
			if score == 0.0 { // Cutoff
				hosts = hosts[:i]
				break
			}
		}
		log.Printf("%d/%d hosts match `%s`.", len(hosts), totalHosts, query)
	}

	// Send results to Alfred -----------------------------------------
	// Show warning if no matches found
	// TODO: Allow ad-hoc entry of hosts
	if len(hosts) == 0 {
		it := workflow.NewItem()
		it.Title = "No matching hosts found"
		it.Subtitle = "Try another query"
		it.Icon = workflow.ICON_WARNING
		workflow.SendFeedback()
		return
	}

	// Alfred feedback
	var title, subtitle, url string
	for _, host := range hosts {
		// Prefix title with username@ to match URL
		if username != "" {
			title = fmt.Sprintf("%s@%s", username, host.Hostname)
		} else {
			title = host.Hostname
		}
		url = host.GetURL(username)
		subtitle = fmt.Sprintf("%s (from %s)", url, host.Source)

		// Create and configure feedback item
		it := workflow.NewItem()
		it.Title = title
		it.Subtitle = subtitle
		it.UID = fmt.Sprintf("%s:%d", host.Hostname, host.Port)
		it.Arg = url
		it.SetValid(true)
		it.SetIcon("icon.png", "")
	}
	workflow.SendFeedback()
}

// main calls run() via workflow.Run().
func main() {
	workflow.GetDefaultWorkflow().Version = Version
	workflow.Run(run)
}
