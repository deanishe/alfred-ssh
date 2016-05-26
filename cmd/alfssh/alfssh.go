//
// Copyright (c) 2016 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2016-05-26
//

/*
alfssh
======

A Script Filter for Alfred 3 for opening SSH connections. Auto-suggests
hosts from ~/.ssh/known_hosts and from /etc/hosts.

The script filter is implemented as a command-line program (that outputs
JSON).
*/
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/docopt/docopt-go"
	"gogs.deanishe.net/deanishe/alfred-ssh"
	"gogs.deanishe.net/deanishe/awgo"
	"gogs.deanishe.net/deanishe/awgo/fuzzy"
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
	wf *workflow.Workflow
)

func init() {
	wf = workflow.NewWorkflow(nil)
}

// Hosts is a collection of Host objects that supports workflow.Fuzzy
// (and therefore sort.Interface).
type Hosts []*assh.Host

// Len etc. implement sort.Interface.
func (s Hosts) Len() int           { return len(s) }
func (s Hosts) Less(i, j int) bool { return s[i].Hostname < s[j].Hostname }
func (s Hosts) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// Keywords implements workflow.Fuzzy.
func (s Hosts) Keywords(i int) string { return s[i].Name() }

// --------------------------------------------------------------------
// Execute Script Filter
// --------------------------------------------------------------------

type options struct {
	printVar    string // Set to print the corresponding variable
	query       string // User query
	url         string // URL to add to history
	username    string // SSH username. Added later by query parser.
	useTestData bool   // Whether to load test data instead of user data
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
	log.Printf("args=%v", args)

	// Alternate Actions
	if args["print"] == true {
		if args["datadir"] == true {
			o.printVar = "data"
		} else if args["cachedir"] == true {
			o.printVar = "cache"
		} else if args["logfile"] == true {
			o.printVar = "log"
		} else if args["distname"] == true {
			o.printVar = "dist"
		}
	} else if args["remember"] == true {
		if s, ok := args["<url>"].(string); ok {
			o.url = s
		} else {
			panic("Can't convert <url> to string.")
		}

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
	var h *assh.History

	historyPath := filepath.Join(wf.DataDir(), "history.json")

	log.Printf("historyPath=%s", historyPath)

	o := runOptions()
	log.Printf("options=%+v", o)

	// ===================== Alternate actions ========================
	if o.printVar == "data" {

		fmt.Println(wf.DataDir())
		return

	} else if o.printVar == "cache" {

		fmt.Println(wf.CacheDir())
		return

	} else if o.printVar == "log" {

		fmt.Println(wf.LogFile())
		return

	} else if o.printVar == "dist" {

		name := strings.Replace(
			fmt.Sprintf("%s-%s.alfredworkflow", wf.Name(), wf.Version()),
			" ", "-", -1)
		fmt.Println(name)

		return

	} else if o.url != "" { // Add host to history

		h := assh.NewHistory(historyPath)
		if err := h.Load(); err != nil {
			log.Printf("Error loading history : %v", err)
			panic(err)
		}
		if err := h.Add(o.url); err != nil {
			log.Printf("Error adding URL : %v", err)
			panic(err)
		}

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
	h = assh.NewHistory(historyPath)
	if err := h.Load(); err != nil {
		log.Printf("Error loading history : %v", err)
	} else {
		log.Printf("Loaded %d items from history", len(h.Hosts()))
	}

	if o.useTestData {
		log.Println("**** Using test data ****")
		hosts = assh.TestHosts()
	} else {
		hosts = append(hosts, h.Hosts()...) // History
		hosts = assh.Hosts()
	}

	totalHosts := len(hosts)
	log.Printf("%d hosts found", totalHosts)

	// Filter hosts ---------------------------------------------------
	if o.query != "" {
		// q := strings.TrimSpace(fmt.Sprintf("%s %s", o.username, o.query))
		for i, score := range fuzzy.Sort(hosts, o.query) {
			if score <= minFuzzyScore { // Cutoff
				hosts = hosts[:i]
				break
			}
			log.Printf("[%f] %+v", score, hosts[i])
		}
		log.Printf("%d/%d hosts match `%s`", len(hosts), totalHosts, o.query)
	}

	// Add Host for query if it makes sense
	if o.query != "" {
		host := &assh.Host{o.query, 22, "user input", ""}
		hosts = append(hosts, host)
	}

	// Send results to Alfred -----------------------------------------
	// Show warning if no matches found
	if len(hosts) == 0 {
		wf.Warn("No matching hosts found", "Try another query")
		return
	}

	// Alfred feedback
	var title, subtitle, url string

	urls := map[string]bool{}
	for _, host := range hosts {

		if o.username != "" &&
			host.Username != "" &&
			o.username != host.Username {
			log.Printf("Ignoring mismatched username: %+v", host)
			continue
		}

		if o.username != "" && host.Username == "" {
			host.Username = o.username
		}

		title = host.Name()
		url = host.URL()
		subtitle = fmt.Sprintf("%s (from %s)", url, host.Source)

		if dupe := urls[url]; dupe {
			log.Printf("Ignoring duplicate result: %v", url)
			continue
		}

		urls[url] = true

		// Create and configure feedback item
		it := wf.NewItem(title)
		it.Subtitle = subtitle
		it.Autocomplete = title
		it.UID = url
		it.Arg = url
		it.Valid = true
		it.SetIcon("icon.png", "")

		// Modifiers
		m, err := workflow.NewModifier("cmd")
		if err != nil {
			panic(err)
		}
		m.SetArg(host.SFTP())
		m.SetSubtitle(fmt.Sprintf("Open as SFTP connection (%s)", host.SFTP()))
		it.SetModifier(m)

	}
	wf.SendFeedback()
}

// main calls run() via Workflow.Run().
func main() {
	wf.Run(run)
}
