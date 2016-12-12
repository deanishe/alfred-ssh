//
// Copyright (c) 2016 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2016-12-11
//

package ssh

import (
	"log"
	"os"
	"sort"
)

// Source provides Hosts.
type Source interface {
	Name() string  // Display name of the source
	Hosts() []Host // Hosts contained by source
	Priority() int // Priority (lower number = higher priority)
}

// Sources is a priority-sorted list of Sources.
type Sources []Source

// Hosts returns all hosts from all sources.
func (sl Sources) Hosts() []Host {
	var hosts = []Host{}
	// var seen = map[string]bool{}
	sort.Sort(sl)
	for _, s := range sl {
		hosts = append(hosts, s.Hosts()...)
	}
	i := len(hosts)
	hosts = FilterDuplicateHosts(hosts)
	dupes := i - len(hosts)
	if dupes > 0 {
		log.Printf("%d duplicate(s) ignored", dupes)
	}
	return hosts
}

// Len implements sort.Interface.
func (sl Sources) Len() int { return len(sl) }

// Less implements sort.Interface.
func (sl Sources) Less(i, j int) bool { return sl[i].Priority() < sl[j].Priority() }

// Swap implements sort.Interface.
func (sl Sources) Swap(i, j int) { sl[i], sl[j] = sl[j], sl[i] }

// DefaultSources returns
func DefaultSources() Sources {
	s := Sources{
		Source(NewConfigSource(os.ExpandEnv("$HOME/.ssh/config"), "~/.ssh/config", 1)),
		Source(NewConfigSource("/etc/ssh/ssh_config", "/etc/ssh", 5)),
		Source(NewHostsSource("/etc/hosts", "/etc/hosts", 4)),
	}
	sort.Sort(s)
	return s
}

type baseSource struct {
	Filepath string
	name     string
	hosts    []Host
	priority int
}

// Name implements Source.
func (s *baseSource) Name() string { return s.name }

// Priority implements Source.
func (s *baseSource) Priority() int { return s.priority }
