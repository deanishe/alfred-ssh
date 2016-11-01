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

	"strconv"

	"github.com/docopt/docopt-go"
	"gogs.deanishe.net/deanishe/alfred-ssh"
	"gogs.deanishe.net/deanishe/awgo"
)

var (
	minScore = 30.0 // Default cut-off for search results
	usage    = `alfssh [options] [<query>]

Display a list of know SSH hosts in Alfred 3. If <query>
is specified, the hostnames will be filtered against it.

Usage:
    alfssh search [-d] [<query>]
    alfssh (remember|forget) <url>
    alfssh print (datadir|cachedir|distname|logfile)
    alfssh --help|--version

Options:
    -h, --help        Show this message and exit.
    --version         Show version information and exit.
    -d, --demo        Use fake test data instead of real data from the computer.
                      Useful for testing, otherwise pointless. Demo mode can also
                      turned on by setting the environment variable DEMO_MODE=1
`
	wfopts *aw.Options
	sopts  *aw.SortOptions
	wf     *aw.Workflow
)

func init() {
	sopts = aw.NewSortOptions()
	sopts.SeparatorBonus = 10.0
	wfopts = &aw.Options{
		SortOptions: sopts,
	}
	wf = aw.NewWorkflow(nil)
}

// Hosts is a collection of Host objects that supports aw.Sortable.
// (and therefore sort.Interface).
type Hosts []*assh.Host

// Len etc. implement sort.Interface.
func (s Hosts) Len() int           { return len(s) }
func (s Hosts) Less(i, j int) bool { return s[i].Hostname < s[j].Hostname }
func (s Hosts) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// SortKey implements aw.Sortable.
func (s Hosts) SortKey(i int) string { return s[i].Name }

// --------------------------------------------------------------------
// Execute Script Filter
// --------------------------------------------------------------------

type options struct {
	port        int    // SSH port. Added later by query parser.
	printVar    string // Set to print the corresponding variable
	query       string // User query. User input is parsed into query and username
	rawInput    string // The full, unparsed query
	remember    bool   // Where to remember or forget url
	url         string // URL to add to history
	username    string // SSH username. Added later by query parser.
	useTestData bool   // Whether to load test data instead of user data
}

// runOptions constructs the program options from command-line arguments and
// environment variables.
func runOptions() *options {

	o := &options{}

	// Parse options --------------------------------------------------
	vstr := fmt.Sprintf("%s/%v (awgo/%v)", wf.Name(),
		wf.Version(), aw.AwgoVersion)

	args, err := docopt.Parse(usage, nil, true, vstr, false)
	if err != nil {
		panic(fmt.Sprintf("Error parsing CLI options : %v", err))
	}
	// log.Printf("args=%v", args)

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
	} else if args["remember"] == true || args["forget"] == true {
		if s, ok := args["<url>"].(string); ok {
			o.url = s
		} else {
			panic("Can't convert <url> to string.")
		}
		if args["remember"] == true {
			o.remember = true
		}
	}

	if args["--demo"] == true || os.Getenv("DEMO_MODE") == "1" {
		o.useTestData = true
	}

	if args["<query>"] != nil {
		if s, ok := args["<query>"].(string); ok {
			o.query = s
			o.rawInput = s
		} else {
			panic("Can't convert query to string.")
		}
	}

	return o
}

// run executes the workflow.
func run() {

	var hosts Hosts
	var host *assh.Host
	// var h *assh.History
	var historyPath string

	o := runOptions()

	if o.useTestData {
		historyPath = filepath.Join(wf.DataDir(), "history.test.json")
	} else {
		historyPath = filepath.Join(wf.DataDir(), "history.json")
	}
	// log.Printf("options=%+v", o)

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

	} else if o.url != "" { // Remember or forget URL

		if os.Getenv("DISABLE_HISTORY") == "1" {
			log.Println("History disabled. Ignoring.")
			return
		}

		h := assh.NewHistory(historyPath)
		if err := h.Load(); err != nil {
			log.Printf("Error loading history : %v", err)
			panic(err)
		}

		if o.remember { // Add URL to history

			if err := h.Add(o.url); err != nil {
				log.Printf("Error adding URL : %v", err)
				panic(err)
			}
		} else { // Remove URL from history
			if err := h.Remove(o.url); err != nil {
				log.Printf("Error removing URL : %v", err)
				panic(err)
			}
			log.Printf("Removed %s from history", o.url)
		}

		return
	}

	// ====================== Script Filter ===========================

	// Parse query ----------------------------------------------------

	// Extract username if there is one
	if i := strings.Index(o.query, "@"); i > -1 {
		o.username, o.query = o.query[:i], o.query[i+1:]
	}
	if i := strings.Index(o.query, ":"); i > -1 {
		var port string
		o.query, port = o.query[:i], o.query[i+1:]
		if v, err := strconv.Atoi(port); err == nil {
			o.port = v
		}
	}

	log.Printf("query=%v, username=%v, port=%v", o.query, o.username, o.port)

	// Load hosts -----------------------------------------------------

	// History
	_, err := assh.RegisterHistory(historyPath)
	if err != nil {
		log.Printf("Error loading history : %v", err)
	}

	// Disable sources user doesn't want
	if os.Getenv("DISABLE_HISTORY") == "1" {
		assh.Disable("history")
	}
	if os.Getenv("DISABLE_ETC_HOSTS") == "1" {
		assh.Disable("/etc/hosts")
	}
	if os.Getenv("DISABLE_KNOWN_HOSTS") == "1" {
		assh.Disable("known_hosts")
	}
	if os.Getenv("DISABLE_CONFIG") == "1" {
		assh.Disable("config")
	}
	if os.Getenv("DISABLE_ETC_CONFIG") == "1" {
		assh.Disable("/etc/config")
	}

	// Main dataset
	if o.useTestData {
		log.Println("**** Using test data ****")
		hosts = append(hosts, assh.TestHosts()...)
	} else {
		hosts = append(hosts, assh.Hosts()...)
	}

	totalHosts := len(hosts)
	log.Printf("%d hosts found", totalHosts)

	// Add Host for query if it makes sense
	if o.query != "" {
		host = &assh.Host{
			Name:     o.rawInput,
			Hostname: o.query,
			Port:     22,
			Source:   "user input",
			Username: o.username,
		}
		hosts = append(hosts, host)
	}

	// Send results to Alfred -----------------------------------------
	// Show warning if no matches found
	if len(hosts) == 0 {
		wf.Warn("No matching hosts found", "Try another query")
		return
	}

	// Alfred feedback
	var cmd, comp, key, title, subtitle, uid, url string
	var exitOnSuccess bool

	if os.Getenv("EXIT_ON_SUCCESS") == "1" {
		exitOnSuccess = true
	}

	seen := map[string]bool{}
	for _, host := range hosts {

		// Ignore hosts with usernames that don't match user's input
		if o.username != "" &&
			host.Username != "" &&
			o.username != host.Username {
			// log.Printf("Ignoring mismatched username: %+v", host)
			continue
		}

		title = host.Name
		comp = host.Name // Autocomplete
		key = host.Name  // Sort key

		if o.username != "" && host.Username == "" {
			host.Username = o.username
			comp = fmt.Sprintf("%s@%s", o.username, host.Name)
			title = comp
		}

		if o.port != 0 && o.port != host.Port {
			host.Port = o.port
			comp = fmt.Sprintf("%s:%d", comp, o.port)
			title = comp
		}

		url = host.URL()
		uid = host.UID()
		subtitle = fmt.Sprintf("%s (from %s)", url, host.Source)

		if dupe := seen[uid]; dupe {
			log.Printf("Ignoring duplicate result: %v", uid)
			continue
		}

		seen[uid] = true

		// Feedback item -------------------------------------------------
		it := wf.NewItem(title).
			Subtitle(subtitle).
			Autocomplete(comp).
			UID(uid).
			Arg(url).
			Copytext(url).
			Valid(true).
			Icon(&aw.Icon{Value: "icon.png"}).
			SortKey(key)

		// Variables -----------------------------------------------------
		it.Var("query", o.rawInput).
			Var("host", host.Hostname).
			Var("source", host.Source).
			Var("shell_cmd", "0").
			Var("url", url)

		// Modifiers -----------------------------------------------------

		// Open SFTP connection instead
		it.NewModifier("cmd").
			Arg(host.SFTP()).
			Subtitle(fmt.Sprintf("Connect with SFTP (%s)", host.SFTP()))

		// Open mosh connection instead
		cmd = host.Mosh(os.Getenv("MOSH_CMD"))
		if exitOnSuccess {
			cmd += " && exit"
		}
		if cmd != "" {
			it.NewModifier("alt").
				Subtitle(fmt.Sprintf("Connect with mosh (%s)", cmd)).
				Arg(cmd).
				Var("shell_cmd", "1")
		}

		// Ping host
		cmd = "ping " + host.Hostname
		if exitOnSuccess {
			cmd += " && exit"
		}
		it.NewModifier("shift").
			Subtitle(fmt.Sprintf("Ping %s", host.Hostname)).
			Arg(cmd).
			Var("shell_cmd", "1")

		// Delete connection from history
		m := it.NewModifier("ctrl")
		if host.Source == "history" {
			m.Subtitle("Delete connection from history").Arg(url).Valid(true)
		} else {
			m.Subtitle("Connection not from history").Valid(false)
		}
	}

	// Filter hosts ---------------------------------------------------
	if o.query != "" {
		// q := strings.TrimSpace(fmt.Sprintf("%s %s", o.username, o.query))
		res := wf.Filter(o.query)
		for i, r := range res {
			log.Printf("%3d. %5.2f %s", i+1, r.Score, r.SortKey)
		}
		log.Printf("%d/%d hosts match `%s`", len(res), totalHosts, o.query)
	}

	wf.SendFeedback()
}

// main calls run() via Workflow.Run().
func main() {
	wf.Run(run)
}
