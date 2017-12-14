//
// Copyright (c) 2016 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2016-05-26
//

/*
assh
====

A Script Filter for Alfred 3 for opening SSH connections. Auto-suggests
hosts from ~/.ssh/known_hosts and from /etc/hosts.

The script filter is implemented as a command-line program (that outputs
JSON).
*/
package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"strconv"

	"os/exec"

	"github.com/deanishe/alfred-ssh"
	"github.com/deanishe/awgo"
	"github.com/deanishe/awgo/fuzzy"
	"github.com/deanishe/awgo/update"
	"github.com/deanishe/awgo/util"

	"github.com/docopt/docopt-go"
)

// Name of background job that checks for updates
const updateJobName = "checkForUpdate"

// GitHub repo
const repo = "deanishe/alfred-ssh"

// Paths to built-in sources
var (
	SSHUserConfigPath   = os.ExpandEnv("$HOME/.ssh/config")
	SSHGlobalConfigPath = "/etc/ssh/ssh_config"
	SSHKnownHostsPath   = os.ExpandEnv("$HOME/.ssh/known_hosts")
	EtcHostsPath        = "/etc/hosts"
	// HistoryVersion      = 2
)

// Priorities for sources
var (
	PriorityUserConfig   = 1
	PriorityKnownHosts   = 2
	PriorityHistory      = 3
	PriorityGlobalConfig = 4
	PriorityEtcHosts     = 5
)

var (
	iconUpdate = &aw.Icon{Value: "update.png"}
	minScore   = 30.0 // Default cut-off for search results
	usage      = `assh [options] [<query>]

Display a list of know SSH hosts in Alfred 3. If <query>
is specified, the hostnames will be filtered against it.

Usage:
    assh open <url>
    assh search [-d] [<query>]
    assh remember <url>
    assh forget <url>
    assh print (datadir|cachedir|distname|logfile)
    assh check
    assh --help|--version

Options:
    -h, --help        Show this message and exit.
    --version         Show version information and exit.
    -d, --demo        Use fake test data instead of real data from the
                      computer.
                      Useful for testing, otherwise pointless. Demo
                      mode can also turned on by setting the
                      environment variable DEMO_MODE=1
`
	// wfopts *aw.Options
	// sopts  *aw.SortOptions
	sopts  []fuzzy.Option
	wfopts []aw.Option
	wf     *aw.Workflow
)

func init() {
	// sopts = aw.NewSortOptions()
	sopts = append(sopts, fuzzy.SeparatorBonus(10.0))
	wf = aw.New(aw.SortOptions(sopts...), update.GitHub(repo))
}

// Hosts is a collection of Host objects that supports aw.Sortable.
// (and therefore sort.Interface).
type Hosts []ssh.Host

// Len etc. implement sort.Interface.
func (s Hosts) Len() int           { return len(s) }
func (s Hosts) Less(i, j int) bool { return s[i].Hostname() < s[j].Hostname() }
func (s Hosts) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// SortKey implements aw.Sortable.
func (s Hosts) SortKey(i int) string { return s[i].Name() }

// --------------------------------------------------------------------
// Execute Script Filter
// --------------------------------------------------------------------

type options struct {
	checkForUpdate bool     // Download list of available releases
	forget         bool     // Whether to forget URL
	open           bool     // Whether to open URL
	print          bool     // Whether to print a variable
	remember       bool     // Whether to remember URL
	printVar       string   // Set to print the corresponding variable
	query          string   // User query. User input is parsed into query and username
	rawInput       string   // The full, unparsed query
	historyPath    string   // Path to history cache file
	url            *url.URL // URL to add to history
	username       string   // SSH username. Added later by query parser.
	port           int      // SSH port. Added later by query parser.
	useTestData    bool     // Whether to load test data instead of user data
	exitOnSuccess  bool     // Append && exit to shell commands
}

// runOptions constructs the program options from command-line arguments and
// environment variables.
func runOptions() *options {

	o := &options{}

	// Parse options --------------------------------------------------
	vstr := fmt.Sprintf("%s/%v (awgo/%v)", wf.Name(),
		wf.Version(), aw.AwGoVersion)

	args, err := docopt.Parse(usage, wf.Args(), true, vstr, false)
	if err != nil {
		panic(fmt.Sprintf("Error parsing CLI options : %v", err))
	}
	// log.Printf("args=%+v", args)

	// Alternate Actions
	if args["check"] == true {
		o.checkForUpdate = true
	}
	if args["remember"] == true {
		o.remember = true
	}
	if args["forget"] == true {
		o.forget = true
	}
	if args["open"] == true {
		o.open = true
	}
	if args["print"] == true {
		o.print = true
	}

	if args["<url>"] != nil {
		if s, ok := args["<url>"].(string); ok {
			o.url, err = url.Parse(s)
			if err != nil || !o.url.IsAbs() {
				wf.Fatalf("Invalid URL: %s", s)
			}
		} else {
			wf.Fatal("Can't convert <url> to string.")
		}
	}

	if o.print {
		if args["datadir"] == true {
			o.printVar = "data"
		} else if args["cachedir"] == true {
			o.printVar = "cache"
		} else if args["logfile"] == true {
			o.printVar = "log"
		} else if args["distname"] == true {
			o.printVar = "dist"
		}
	}

	if args["--demo"] == true || optionSet("DEMO_MODE") {
		o.useTestData = true
		o.historyPath = filepath.Join(wf.DataDir(), "history.test.json")
	} else {
		o.historyPath = filepath.Join(wf.DataDir(), "history.json")
	}

	if args["<query>"] != nil {
		if s, ok := args["<query>"].(string); ok {
			s = strings.TrimSpace(s)
			o.query = s
			o.rawInput = s
		} else {
			wf.Fatal("Can't convert query to string.")
		}
	}

	if optionSet("EXIT_ON_SUCCESS") {
		o.exitOnSuccess = true
	}

	return o
}

// Print a variable to STDOUT
func runPrint(o *options) {
	if o.printVar == "data" {

		fmt.Print(wf.DataDir())
		return

	} else if o.printVar == "cache" {

		fmt.Print(wf.CacheDir())
		return

	} else if o.printVar == "log" {

		fmt.Print(wf.LogFile())
		return

	} else if o.printVar == "dist" {

		name := strings.Replace(
			fmt.Sprintf("%s-%s.alfredworkflow", wf.Name(), wf.Version()),
			" ", "-", -1)
		fmt.Print(name)

		return

	}
}

// Open a URL in the default or custom application.
func runOpen(o *options) {

	wf.TextErrors = true

	var (
		argv     = []string{}
		sshHdlr  = os.Getenv("SSH_APP")
		sftpHdlr = os.Getenv("SFTP_APP")
	)
	log.Printf("Opening URL %s", o.url)
	if o.url.Scheme == "ssh" && sshHdlr != "" {
		argv = append(argv, "-a", sshHdlr)
	} else if o.url.Scheme == "sftp" && sftpHdlr != "" {
		argv = append(argv, "-a", sftpHdlr)
	}
	argv = append(argv, o.url.String())
	cmd := exec.Command("open", argv...)
	log.Printf("Command: %s %+v", cmd.Path, cmd.Args)
	out, err := cmd.CombinedOutput()
	if err != nil {
		wf.Fatal(string(out))
	}
	return

}

// Check for an update to the workflow
func runUpdate(o *options) {
	wf.TextErrors = true

	if err := wf.CheckForUpdate(); err != nil {
		wf.FatalError(err)
	}

	if wf.UpdateAvailable() {
		log.Printf("[update] An update is available")
	} else {
		log.Printf("[update] Workflow is up to date")
	}
}

// Add host or remove host from history
func runHistory(o *options) {
	if os.Getenv("DISABLE_HISTORY") == "1" {
		log.Println("History disabled. Ignoring.")
		return
	}

	h := ssh.NewHistory(o.historyPath, "history", 1)
	if err := h.Load(); err != nil {
		log.Printf("Error loading history : %v", err)
		panic(err)
	}

	host := ssh.NewBaseHostFromURL(o.url)
	if o.remember { // Add URL to history
		if err := h.Add(host); err != nil {
			log.Printf("Error adding host %v : %v", host, err)
			panic(err)
		}
		log.Printf("Saved host '%s' to history", host.Name())
	} else { // Remove URL from history
		if err := h.Remove(host); err != nil {
			log.Printf("Error removing host %v : %v", host, err)
			panic(err)
		}
		log.Printf("Removed '%s' from history", host.Name())
	}
	h.Save()
	return
}

// Alfred Script Filter to search hosts
func runSearch(o *options) {
	var hosts Hosts
	var host ssh.Host

	// Parse query ----------------------------------------------------
	// Extract username if present
	if i := strings.Index(o.query, "@"); i > -1 {
		o.username, o.query = o.query[:i], o.query[i+1:]
	}
	// Extract port if present
	if i := strings.Index(o.query, ":"); i > -1 {
		var port string
		o.query, port = o.query[:i], o.query[i+1:]
		if v, err := strconv.Atoi(port); err == nil {
			o.port = v
		}
	}

	log.Printf("query=%v, username=%v, port=%v", o.query, o.username, o.port)

	// Show update status if there's no query
	if o.query == "" && wf.UpdateAvailable() {
		// noUIDs = true
		wf.NewItem("An update is available!").
			Subtitle("↩ or ⇥ to install").
			Valid(false).
			Autocomplete("workflow:update").
			Icon(iconUpdate)
	}

	// Load hosts from sources ----------------------------------------
	hosts = loadHosts(o)
	totalHosts := len(hosts)
	// log.Printf("%d total host(s)", totalHosts)

	// Prepare results for Alfred -------------------------------------
	// seen := map[string]bool{}
	d := ssh.Deduplicator{}
	for _, host := range hosts {

		// Force use of username/port parsed from input
		if o.username != "" {
			host.SetUsername(o.username)
		}
		if o.port != 0 && o.port != 22 {
			host.SetPort(o.port)
		}

		// Check again if it's a dupe
		if !d.IsDuplicate(host) {
			itemForHost(host, o)
			d.Add(host)
		}
	}

	// Filter hosts and/or add host from query ------------------------
	if o.query != "" {
		// Filter hosts
		res := wf.Filter(o.query)
		for i, r := range res {
			log.Printf("%3d. %5.2f %s", i+1, r.Score, r.SortKey)
		}
		log.Printf("%d/%d hosts match `%s`", len(res), totalHosts, o.query)

		// Add Host for query if it makes sense
		if ssh.IsValidHostname(o.query) {
			host = ssh.NewBaseHost(o.rawInput, o.query, "user input", o.username, o.port)
			if !d.IsDuplicate(host) {
				itemForHost(host, o)
			}
		} else {
			wf.WarnEmpty(fmt.Sprintf("Invalid hostname: %s", o.query), "Enter a different value")
		}
	}

	wf.WarnEmpty("No matching hosts", "Try different input")

	wf.SendFeedback()
}

// run executes the workflow. Calls other run* functions based on command-line options.
func run() {
	o := runOptions()
	// log.Printf("options=%+v", o)

	if o.checkForUpdate {
		runUpdate(o)
		return
	}

	// Run update check
	if wf.UpdateCheckDue() && !aw.IsRunning(updateJobName) {
		log.Println("Checking for update...")
		cmd := exec.Command("./assh", "check")
		if err := aw.RunInBackground(updateJobName, cmd); err != nil {
			log.Printf("Error running update check: %s", err)
		}
	}

	if o.print {
		runPrint(o)
		return
	} else if o.open {
		runOpen(o)
		return
	} else if o.remember || o.forget {
		runHistory(o)
		return
	}
	runSearch(o)

}

// itemForHost adds a feedback Item to Workflow wf for Host.
func itemForHost(host ssh.Host, o *options) *aw.Item {
	var (
		cmd      string
		title    = host.Name()
		comp     = host.Name() // Autocomplete
		key      = host.Name() // Sort key
		url      = host.SSHURL().String()
		uid      = host.UID()
		subtitle = fmt.Sprintf("%s (from %s)", url, host.Source())
	)

	if o.username != "" && host.Username() == "" {
		host.SetUsername(o.username)
		comp = fmt.Sprintf("%s@%s", o.username, host.Name())
		title = comp
	}

	if o.port != 0 && o.port != host.Port() {
		host.SetPort(o.port)
		comp = fmt.Sprintf("%s:%d", comp, o.port)
		title = comp
	}

	// Feedback item
	it := wf.NewItem(title).
		Subtitle(subtitle).
		Autocomplete(comp).
		Arg(url).
		Copytext(url).
		Largetype(host.CanonicalURL().String()).
		UID(uid).
		Valid(true).
		Icon(&aw.Icon{Value: "icon.png"}).
		Match(key)

	// Variables
	it.Var("query", o.rawInput).
		Var("name", host.Name()).
		Var("hostname", host.Hostname()).
		Var("source", host.Source()).
		Var("port", fmt.Sprintf("%d", host.Port())).
		Var("shell_cmd", "0").
		Var("url", url)

	// Modifiers

	// Open SFTP connection instead
	url = host.SFTPURL().String()
	it.NewModifier("cmd").
		Arg(url).
		Subtitle(fmt.Sprintf("Connect with SFTP (%s)", url))

	// Open mosh connection instead
	if os.Getenv("MOSH_CMD") != "" {
		cmd = host.MoshCmd(os.Getenv("MOSH_CMD"))
		if cmd != "" {
			if o.exitOnSuccess {
				cmd += " && exit"
			}
			it.NewModifier("alt").
				Subtitle(fmt.Sprintf("Connect with mosh (%s)", cmd)).
				Arg(cmd).
				Var("shell_cmd", "1")
		}
	}

	// Ping host
	cmd = "ping " + host.Hostname()
	if o.exitOnSuccess {
		cmd += " && exit"
	}
	it.NewModifier("shift").
		Subtitle(fmt.Sprintf("Ping %s", host.Hostname())).
		Arg(cmd).
		Var("shell_cmd", "1")

	// Delete connection from history
	m := it.NewModifier("ctrl")
	if host.Source() == "history" {
		m.Subtitle("Delete connection from history").Arg(url).Valid(true)
	} else {
		m.Subtitle("Connection not from history").Valid(false)
	}
	return it
}

// loadHosts loads Hosts from all active sources.
func loadHosts(o *options) []ssh.Host {
	var start = time.Now()
	var hosts Hosts

	if o.useTestData {
		log.Println("**** Using test data ****")
		hosts = append(hosts, ssh.TestHosts()...)
		return hosts
	}

	sources := ssh.Sources{}

	if !optionSet("DISABLE_HISTORY") {
		sources = append(sources, ssh.NewHistory(o.historyPath, "history", PriorityHistory))
		// log.Printf("[source/new/history] %s", aw.ShortenPath(o.historyPath))
	}
	if !optionSet("DISABLE_ETC_HOSTS") {
		sources = append(sources, ssh.NewHostsSource(EtcHostsPath, "/etc/hosts", PriorityEtcHosts))
		// log.Printf("[source/new/hosts] %s", EtcHostsPath)
	}
	if !optionSet("DISABLE_KNOWN_HOSTS") {
		sources = append(sources, ssh.NewKnownSource(SSHKnownHostsPath, "known_hosts", PriorityKnownHosts))
		// log.Printf("[source/new/known_hosts] %s", aw.ShortenPath(SSHKnownHostsPath))
	}
	if !optionSet("DISABLE_CONFIG") {
		sources = append(sources, ssh.NewConfigSource(SSHUserConfigPath, "~/.ssh/config", PriorityUserConfig))
		// log.Printf("[source/new/config] %s", aw.ShortenPath(SSHUserConfigPath))
	}
	if !optionSet("DISABLE_ETC_CONFIG") {
		sources = append(sources, ssh.NewConfigSource(SSHGlobalConfigPath, "/etc/ssh", PriorityGlobalConfig))
		// log.Printf("[source/new/config] %s", SSHGlobalConfigPath)
	}
	hosts = append(hosts, sources.Hosts()...)

	log.Printf("%d host(s) loaded in %s", len(hosts), util.HumanDuration(time.Since(start)))
	return hosts
}

// optionSet returns true if environment variable key is set to 1, Y, yes etc.
func optionSet(key string) bool {
	v := strings.ToLower(os.Getenv(key))
	if v == "" {
		return false
	}
	if v == "1" || v == "y" || v == "yes" {
		return true
	}
	return false
}

// main calls run() via Workflow.Run().
func main() {
	wf.Run(run)
}
