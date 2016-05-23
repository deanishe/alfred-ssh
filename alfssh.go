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

A Script Filter for Alfred 3 for opening SSH connections. Auto-suggests
hosts from ~/.ssh/known_hosts and from /etc/hosts.

The script filter is implemented as a command-line program (that outputs
JSON).
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
// Version is the current version of the workflow
// Version = "0.2.0"
)

var (
	usage = `alfssh [options] [<query>]

Display a list of know SSH hosts in Alfred 3. If <query>
is specified, the hostnames will be filtered against it.

Usage:
    alfssh [-t] [<query>]
    alfssh --help|--version
    alfssh --datadir|--cachedir|--distname|--logfile

Options:
    --datadir   Print path to workflow's data directory and exit.
    --cachedir  Print path to workflow's cache directory and exit.
    --logfile   Print path to workflow's logfile and exit.
    -h, --help  Show this message and exit.
    --version   Show version information and exit.
    -d, --demo  Use fake test data instead of real data from the computer.
                Useful for testing, otherwise pointless.
    --distname  Print filename of distributable .alfredworkflow file (for
                the build script).
`

	// knownHostsPath string
	knownHostsPath = os.ExpandEnv("$HOME/.ssh/known_hosts")
	etcHostsPath   = "/etc/hosts"
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

// Hosts is a collection of Host objects that supports workflow.Fuzzy
// (and therefore sort.Interface).
type Hosts []*Host

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
		hosts = append(hosts, &Host{hostname, port, "~/.ssh/known_hosts"})
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
	log.Printf("%d hosts in ~/.ssh/known_hosts.", len(hosts))
	return hosts
}

// readEtcHosts reads hostnames from /etc/hosts.
func readEtcHosts() []*Host {
	var hosts []*Host
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
			hosts = append(hosts, &Host{s, 22, "/etc/hosts"})
		}
	}
	log.Printf("%d hosts in /etc/hosts.", len(hosts))
	return hosts
}

// loadTestHosts loads fake test data instead of real hosts.
func loadTestHosts() Hosts {
	hosts := make(Hosts, len(testHostnames))
	for i, name := range testHostnames {
		hosts[i] = &Host{name, 22, "test data"}
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

	// sort.Sort(hosts)
	return hosts
}

// --------------------------------------------------------------------
// Execute Script Filter
// --------------------------------------------------------------------

type options struct {
	printVar string

	useTestData bool

	query    string
	username string
}

// runOptions constructs the program options from command-line arguments and
// environment variables.
func runOptions() *options {
	o := &options{}

	// Parse options --------------------------------------------------
	vstr := fmt.Sprintf("%s/%v (awgo/%v)", workflow.Name(),
		workflow.Version(), workflow.LibVersion)

	args, err := docopt.Parse(usage, nil, true, vstr, false)
	if err != nil {
		panic(fmt.Sprintf("Error parsing CLI options : %v", err))
	}
	// log.Printf("args=%v", args)

	// Alternate Actions
	if args["--datadir"] == true {
		o.printVar = "data"
	} else if args["--cachedir"] == true {
		o.printVar = "cache"
	} else if args["--logfile"] == true {
		o.printVar = "log"
	} else if args["--distname"] == true {
		o.printVar = "dist"
	}

	if args["--demo"] == true || os.Getenv("DEMO_MODE") != "" {
		o.useTestData = true
	}

	if args["<query>"] != nil {
		if s, ok := args["<query>"].(string); ok {
			o.query = s
		} else {
			panic("Can't convert query to string.")
		}
	}

	return o
}

// run executes the workflow.
func run() {

	var hosts Hosts

	o := runOptions()
	log.Printf("options=%v", o)

	// ===================== Alternate actions ========================
	if o.printVar == "data" {
		fmt.Println(workflow.DataDir())
		return
	} else if o.printVar == "cache" {
		fmt.Println(workflow.CacheDir())
		return
	} else if o.printVar == "log" {
		fmt.Println(workflow.LogFile())
		return
	} else if o.printVar == "dist" {
		name := strings.Replace(
			fmt.Sprintf("%s-%s.alfredworkflow", workflow.Name(), workflow.DefaultWorkflow().Version),
			" ", "-", -1)
		fmt.Println(name)
		return
	}

	// ====================== Script Filter ===========================

	// Parse query ----------------------------------------------------

	// Extract username if there is one
	if i := strings.Index(o.query, "@"); i > -1 {
		o.username, o.query = o.query[:i], o.query[i+1:]
		log.Printf("username=%v", o.username)
	}
	log.Printf("query=%v", o.query)

	// Load hosts -----------------------------------------------------
	if o.useTestData {
		log.Println("**** Using test data ****")
		hosts = loadTestHosts()
	} else {
		hosts = loadHosts()
	}

	totalHosts := len(hosts)
	log.Printf("%d hosts found.", totalHosts)

	// Filter hosts ---------------------------------------------------
	if o.query != "" {
		// var matches Hosts
		for i, score := range workflow.SortFuzzy(hosts, o.query) {
			if score == 0.0 { // Cutoff
				hosts = hosts[:i]
				break
			}
		}
		log.Printf("%d/%d hosts match `%s`.", len(hosts), totalHosts, o.query)
	}

	// Add Host for query if it makes sense
	if o.query != "" {
		qhost := &Host{o.query, 22, "user input"}
		dupe := false
		for _, h := range hosts {
			if h.GetURL(o.username) == qhost.GetURL(o.username) {
				dupe = true
				break
			}
		}
		if !dupe {
			hosts = append(hosts, qhost)
		}
	}

	// Send results to Alfred -----------------------------------------
	// Show warning if no matches found
	if len(hosts) == 0 {
		workflow.Warn("No matching hosts found", "Try another query")
		return
	}

	// Alfred feedback
	var title, subtitle, url string

	for _, host := range hosts {

		// Prefix title with username@ to match URL
		if o.username != "" {
			title = fmt.Sprintf("%s@%s", o.username, host.Hostname)
		} else {
			title = host.Hostname
		}

		url = host.GetURL(o.username)
		subtitle = fmt.Sprintf("%s (from %s)", url, host.Source)

		// Create and configure feedback item
		it := workflow.NewItem()
		it.Title = title
		it.Subtitle = subtitle
		it.Autocomplete = title
		it.UID = fmt.Sprintf("%s:%d", host.Hostname, host.Port)
		it.Arg = url
		it.Valid = true
		it.SetIcon("icon.png", "")
	}
	workflow.SendFeedback()
}

// main calls run() via workflow.Run().
func main() {
	// workflow.DefaultWorkflow().Version = Version
	workflow.Run(run)
}
