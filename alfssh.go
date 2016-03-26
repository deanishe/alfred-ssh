package main

import (
	"fmt"
	"log"

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
	alfssh [<query>]
	alfssh --datadir
	alfssh --cachedir
	alfssh --help|--version

Options:
	--datadir   Print path to workflow's data directory and exit.
	--cachedir  Print path to workflow's cache directory and exit.
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
func (h Host) GetURL() string {
	url := fmt.Sprintf("ssh://%s", h.Hostname)
	return url
}
func (h Host) Keywords() string {
	return h.Hostname
}

// Hosts is a sortable list of Hosts
// type Hosts []*Host

// func (s Hosts) Len() int {
// 	return len(s)
// }

// func (s Hosts) Less(i, j int) bool {
// 	return s[i].Hostname < s[j].Hostname
// }

// func (s Hosts) Swap(i, j int) {
// 	s[i], s[j] = s[j], s[i]
// }

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
	hosts := make([]Host, len(hostnames))
	for i, hostname := range hostnames {
		hosts[i] = Host{hostname, 22, "hardcoded"}
	}
	// sort.Sort(hosts)
	return hosts
}

// run executes the workflow.
func run() {
	var query string
	var hosts []Host
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

	// ====================== Script Filter ===========================
	if args["<query>"] == nil {
		query = ""
	} else {
		query = fmt.Sprintf("%v", args["<query>"])
	}
	log.Printf("query=%v", query)

	// Load and filter hosts ------------------------------------------
	hosts = loadHosts()
	log.Printf("%d known hosts.", len(hosts))

	if query != "" {
		// TODO: Filter hosts
		s := make([]workflow.Filterable, len(hosts))
		for i, h := range hosts {
			s[i] = h
		}
		// Cast to []Filterable
		matches := workflow.Filter(query, s, 0.0)
		log.Printf("%d/%d hosts match '%s'.", len(matches), len(matches), query)
		var h Host
		for i, f := range matches {
			h, _ = f.(Host)
			log.Printf("%3d  %s", i+1, h.Hostname)
		}
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
