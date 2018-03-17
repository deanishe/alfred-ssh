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

const (
	// Name of background job that checks for updates
	updateJobName = "checkForUpdate"
	// GitHub repo
	repo = "deanishe/alfred-ssh"
	// Doc & help URLs
	docsURL  = "https://github.com/deanishe/alfred-ssh/blob/master/README.md"
	issueURL = "https://github.com/deanishe/alfred-ssh/issues"
	forumURL = "https://www.alfredforum.com/topic/8956-secure-shell-for-alfred-3-ssh-plus-sftp-mosh-ping-with-autosuggest/"
)

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

// Workflow icons
var (
	IconWorkflow        = &aw.Icon{Value: "icon.png"}
	IconConfig          = &aw.Icon{Value: "icons/config.png"}
	IconDocs            = &aw.Icon{Value: "icons/docs.png"}
	IconHelp            = &aw.Icon{Value: "icons/help.png"}
	IconIssue           = &aw.Icon{Value: "icons/issue.png"}
	IconUpdateAvailable = &aw.Icon{Value: "icons/update-available.png"}
	IconUpdateOK        = &aw.Icon{Value: "icons/update-ok.png"}
	IconURL             = &aw.Icon{Value: "icons/url.png"}
	IconOn              = &aw.Icon{Value: "icons/on.png"}
	IconOff             = &aw.Icon{Value: "icons/off.png"}
	IconLog             = &aw.Icon{Value: "icons/log.png"}
)

var (
	minScore = 30.0 // Default cut-off for search results
	usage    = `assh [options] [<query>]

Display a list of know SSH hosts in Alfred 3. If <query>
is specified, the hostnames will be filtered against it.

Usage:
    assh open <url>
    assh search [-d] [<query>]
    assh remember <url>
    assh forget <url>
    assh print (datadir|cachedir|distname|logfile)
    assh check
	assh config [<query>]
	assh toggle <var>
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
	wf *aw.Workflow
)

func init() {
	wf = aw.New(
		aw.SortOptions(
			fuzzy.SeparatorBonus(10.0),
		),
		aw.AddMagic(
			urlMagic{"docs", "Open workflow documentation in your browser", docsURL},
			urlMagic{"forum", "Visit the workflow thread on alfredforum.com", forumURL},
		),
		update.GitHub(repo),
		aw.HelpURL(issueURL),
	)
}

// main calls run() via Workflow.Run().
func main() { wf.Run(run) }

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
	// Command-line options
	Check         bool   // Download list of available releases
	Config        bool   // Whether to show configuration options
	Demo          bool   `env:"DEMO_MODE"` // Whether to load test data instead of user data
	Forget        bool   // Whether to forget URL
	Open          bool   // Whether to open URL
	Print         bool   // Whether to print a variable
	PrintDataDir  bool   `docopt:"datadir"`
	PrintCacheDir bool   `docopt:"cachedir"`
	PrintDistName bool   `docopt:"distname"`
	PrintLogFile  bool   `docopt:"logfile"`
	Remember      bool   // Whether to remember URL
	Search        bool   // Whether to search hosts
	Toggle        bool   // Whether to toggle a setting on/off
	RawInput      string `docopt:"<query>"` // The full, unparsed query
	RawURL        string `docopt:"<url>"`   // Input URL
	VarName       string `docopt:"<var>"`   // Name of variable to toggle

	// Workflow configuration (environment variables)
	DisableConfig     bool
	DisableEtcConfig  bool
	DisableEtcHosts   bool
	DisableHistory    bool
	DisableKnownHosts bool
	ExitOnSuccess     bool // Append " && exit" to shell commands
	MoshCmd           string
	SFTPApp           string `env:"SFTP_APP"`
	SSHApp            string `env:"SSH_APP"`
	SSHCmd            string `env:"SSH_CMD"`

	// Derived configuration
	query       string   // User query. User input is parsed into query and username
	url         *url.URL // URL to add to history
	username    string   // SSH username. Added later by query parser.
	port        int      // SSH port. Added later by query parser.
	historyPath string   // Path to history cache file
}

// MagicAction that opens a given URL.
type urlMagic struct {
	keyword     string
	description string
	URL         string
}

func (ma urlMagic) Keyword() string     { return ma.keyword }
func (ma urlMagic) Description() string { return ma.description }
func (ma urlMagic) RunText() string     { return fmt.Sprintf("Opening %s ...", ma.URL) }
func (ma urlMagic) Run() error {
	cmd := exec.Command("/usr/bin/open", ma.URL)
	_, err := util.RunCmd(cmd)
	return err
}

// parseArgs constructs the program options from command-line arguments and
// environment variables.
func parseArgs() *options {

	var (
		o    = &options{}
		vstr = fmt.Sprintf("%s/%v (awgo/%v)", wf.Name(), wf.Version(), aw.AwGoVersion)
		err  error
	)

	// Parse options --------------------------------------------------

	args, err := docopt.ParseArgs(usage, wf.Args(), vstr)
	if err != nil {
		panic(fmt.Sprintf("Error parsing CLI options: %v", err))
	}

	if err = args.Bind(o); err != nil {
		panic(fmt.Sprintf("Error parsing CLI options: %v", err))
	}

	if err = wf.Alfred.To(o); err != nil {
		panic(fmt.Sprintf("Error loading workflow configuration: %v", err))
	}

	if o.RawURL != "" {
		o.url, err = url.Parse(o.RawURL)
		if err != nil || !o.url.IsAbs() {
			wf.Fatalf("Invalid URL: %s", o.RawURL)
		}
	}

	if o.Demo {
		o.historyPath = filepath.Join(wf.DataDir(), "history.test.json")
	} else {
		o.historyPath = filepath.Join(wf.DataDir(), "history.json")
	}

	if o.RawInput != "" {
		o.RawInput = strings.TrimSpace(o.RawInput)
		o.query = o.RawInput
	}

	return o
}

// Print a variable to STDOUT
func runPrint(o *options) {

	if o.PrintDataDir {
		fmt.Print(wf.DataDir())
		return

	} else if o.PrintCacheDir {
		fmt.Print(wf.CacheDir())
		return

	} else if o.PrintLogFile {
		fmt.Print(wf.LogFile())
		return

	} else if o.PrintDistName {
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

	if o.DisableHistory {
		log.Println("History disabled. Ignoring.")
		return
	}

	h := ssh.NewHistory(o.historyPath, "history", 1)
	if err := h.Load(); err != nil {
		log.Printf("Error loading history : %v", err)
		panic(err)
	}

	host := ssh.NewBaseHostFromURL(o.url)

	if o.Remember { // Add URL to history
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

// Alfred Script Filter to view configuration
func runConfig(o *options) {

	sources := []struct {
		title, file, varName string
		disabled             bool
	}{
		{"SSH Config", "~/.ssh/config", "DISABLE_CONFIG", o.DisableConfig},
		{"SSH Config (system)", "/etc/ssh/ssh_config", "DISABLE_ETC_CONFIG", o.DisableEtcConfig},
		{"/etc/hosts", "/etc/hosts", "DISABLE_ETC_HOSTS", o.DisableEtcHosts},
		{"History", "workflow history", "DISABLE_HISTORY", o.DisableHistory},
		{"Known Hosts", "~/.ssh/known_hosts", "DISABLE_KNOWN_HOSTS", o.DisableKnownHosts},
	}

	wf.Var("query", o.query)

	if wf.UpdateAvailable() {
		wf.NewItem("An Update is Available!").
			Subtitle("↩ or ⇥ to install").
			Autocomplete("workflow:update").
			Icon(IconUpdateAvailable).
			Valid(false)
	} else {
		wf.NewItem("Workflow is Up To Date").
			Subtitle("↩ or ⇥ to check for update now").
			Autocomplete("workflow:update").
			Icon(IconUpdateOK).
			Valid(false)
	}

	for _, src := range sources {

		icon := IconOn
		if src.disabled {
			icon = IconOff
		}

		wf.NewItem("Source: " + src.title).
			Subtitle(src.file).
			Arg(src.varName).
			Valid(true).
			Icon(icon)

	}

	wf.NewItem("Log File").
		Subtitle("Open workflow log file").
		Autocomplete("workflow:log").
		Icon(IconLog)

	// Docs & help URLs
	wf.NewItem("Documentation").
		Subtitle("Read the workflow docs in your browser").
		Autocomplete("workflow:docs").
		Icon(IconDocs)

	wf.NewItem("Report Issue").
		Subtitle("Open the workflow's issue tracker on GitHub").
		Autocomplete("workflow:help").
		Icon(IconIssue)

	wf.NewItem("Visit Forum").
		Subtitle("Open the workflow's thread on alfredforum.com").
		Autocomplete("workflow:forum").
		Icon(IconURL)

	if o.query != "" {
		wf.Filter(o.query)
	}

	wf.WarnEmpty("No matches found", "Try a different query?")
	wf.SendFeedback()
}

// Toggle a setting on/off
func runToggle(o *options) {

	wf.Configure(aw.TextErrors(true))

	var s = "1"

	if wf.Alfred.GetBool(o.VarName) {
		s = "0"
	}

	log.Printf("[toggle] %s ->  %q", o.VarName, s)

	if err := wf.Alfred.SetConfig(o.VarName, s, true).Do(); err != nil {
		wf.FatalError(err)
	}

}

// Alfred Script Filter to search hosts
func runSearch(o *options) {

	var (
		hosts Hosts
		host  ssh.Host
	)

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

		// Ensure update notification is top results
		wf.Configure(aw.SuppressUIDs(true))

		wf.NewItem("An Update is Available!").
			Subtitle("↩ or ⇥ to install").
			Valid(false).
			Autocomplete("workflow:update").
			Icon(IconUpdateAvailable)
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
			host = ssh.NewBaseHost(o.RawInput, o.query, "user input", o.username, o.port)
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

	o := parseArgs()
	log.Printf("options=\n%+v\n", o)

	if o.Check {
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

	if o.Print {
		runPrint(o)
		return
	} else if o.Open {
		runOpen(o)
		return
	} else if o.Remember || o.Forget {
		runHistory(o)
		return
	} else if o.Toggle {
		runToggle(o)
		return
	} else if o.Config {
		runConfig(o)
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
		Icon(IconWorkflow).
		Match(key)

	// Variables
	it.Var("query", o.RawInput).
		Var("name", host.Name()).
		Var("hostname", host.Hostname()).
		Var("source", host.Source()).
		Var("port", fmt.Sprintf("%d", host.Port())).
		Var("shell_cmd", "0").
		Var("url", url)

	// Send ssh command via Terminal Command instead of opening URL
	if os.Getenv("SSH_CMD") != "" {
		cmd = host.SSHCmd(os.Getenv("SSH_CMD"))
		if cmd != "" {
			if o.ExitOnSuccess {
				cmd += " && exit"
			}
			it.Arg(cmd)
			it.Subtitle(fmt.Sprintf("%s (from %s)", cmd, host.Source()))
			it.Var("shell_cmd", "1")
		}
	}

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
			if o.ExitOnSuccess {
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
	if o.ExitOnSuccess {
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

	if o.Demo {
		log.Println("**** Using test data ****")
		hosts = append(hosts, ssh.TestHosts()...)
		return hosts
	}

	sources := ssh.Sources{}

	if !o.DisableHistory {
		sources = append(sources, ssh.NewHistory(o.historyPath, "history", PriorityHistory))
		// log.Printf("[source/new/history] %s", aw.ShortenPath(o.historyPath))
	}
	if !o.DisableEtcHosts {
		sources = append(sources, ssh.NewHostsSource(EtcHostsPath, "/etc/hosts", PriorityEtcHosts))
		// log.Printf("[source/new/hosts] %s", EtcHostsPath)
	}
	if !o.DisableKnownHosts {
		sources = append(sources, ssh.NewKnownSource(SSHKnownHostsPath, "known_hosts", PriorityKnownHosts))
		// log.Printf("[source/new/known_hosts] %s", aw.ShortenPath(SSHKnownHostsPath))
	}
	if !o.DisableConfig {
		sources = append(sources, ssh.NewConfigSource(SSHUserConfigPath, "~/.ssh/config", PriorityUserConfig))
		// log.Printf("[source/new/config] %s", aw.ShortenPath(SSHUserConfigPath))
	}
	if !o.DisableEtcConfig {
		sources = append(sources, ssh.NewConfigSource(SSHGlobalConfigPath, "/etc/ssh", PriorityGlobalConfig))
		// log.Printf("[source/new/config] %s", SSHGlobalConfigPath)
	}
	hosts = append(hosts, sources.Hosts()...)

	log.Printf("%d host(s) loaded in %s", len(hosts), time.Since(start))
	return hosts
}

/*
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
*/
