package main

import (
	"fmt"
	"log"
	"sort"

	"github.com/docopt/docopt-go"
	"gogs.deanishe.net/deanishe/awgo"
)

const (
	Version = "0.1"
)

var (
	usage = `alfssh [<query>]

Display a list of know SSH hosts in Alfred. If <query>
is specified, the hostnames will be filtered against it.

Usage:
	alfssh [<query>]
	alfssh -h|--version

Options:
	-h, --help  Show this message and exit.
	--version   Show version information and exit.
`
)

func init() {
}

// Host is computer that may be connected to.
type Host struct {
	Hostname string
	Port     int
	// Name of the source, e.g. "known_hosts"
	Source string
}

// GetURL returns the ssh:// URL for the host.
func (h *Host) GetURL() string {
	url := fmt.Sprintf("ssh://%s", h.Hostname)
	return url
}

func GetHostSearchText(h *Host) string {
	return h.Hostname
}

type Hosts []Host

func (s Hosts) Len() int {
	return len(s)
}

func (s Hosts) Less(i, j int) bool {
	return s[i].Hostname < s[j].Hostname
}

func (s Hosts) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// loadHosts returns a sequence of hostnames.
func loadHosts() []Host {
	hostnames := []string{
		"toot.home.lan",
		"mclovin.home.lan",
		"poon.deanishe.net",
		"snarf.deanishe.com",
		"krieger.deanishe.net",
		"youse.cnblw.me",
		"cheese.momama.net",
		"vla.deanjackson.de",
	}
	hosts := make(Hosts, len(hostnames))
	for i, hostname := range hostnames {
		hosts[i] = Host{hostname, 22, "hardcoded"}
	}
	sort.Sort(hosts)
	return hosts
}

// run executes the workflow.
func run() {
	// Parse options --------------------------------------------------
	vstr := fmt.Sprintf("%s/%v (awgo/%v)", workflow.GetName(), Version,
		workflow.Version)

	args, err := docopt.Parse(usage, nil, true, vstr, false)
	if err != nil {
		log.Fatalf("Error parsing CLI options : %v", err)
	}
	log.Printf("args=%v", args)

	query := args["<query>"]
	// Alfred will pass an empty string, so normalise value for shell
	// and Alfred.
	if query == nil {
		query = ""
	}
	log.Printf("query=%v", query)

	// Load and filter hosts ------------------------------------------
	hosts := loadHosts()
	log.Printf("%d known hosts.", len(hosts))

	if query != "" {
		log.Printf("%d hosts match '%s'.", len(hosts), query)
	}

	// Send results to Alfred -----------------------------------------
	var url string
	for _, host := range hosts {
		url = host.GetURL()
		it := workflow.NewItem()
		it.Title = host.Hostname
		it.Subtitle = url
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
