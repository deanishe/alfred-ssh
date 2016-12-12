//
// Copyright (c) 2016 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2016-12-11
//

package ssh

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/url"
	"os"
)

type historyEntry struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// History is a list of previously opened URLs.
type History struct {
	baseSource
	d *Deduplicator
}

// NewHistory initialises a new History struct. You must call History.Load()
// to load cached data.
func NewHistory(path, name string, priority int) *History {
	h := &History{}
	h.Filepath = path
	h.name = name
	h.priority = priority
	h.d = &Deduplicator{}
	return h
}

// Add adds an item to the History.
func (h *History) Add(host Host) error {
	if h.d.IsDuplicate(host) {
		log.Printf("[history/%s] Ignoring duplicate: %v", h.Filepath, host)
		return nil
	}

	h.hosts = append(h.hosts, host)
	h.d.Add(host)

	log.Printf("Adding %s to history ...", host.Name())

	return h.Save()
}

// Remove removes an item from the History.
func (h *History) Remove(host Host) error {
	for i, xh := range h.hosts {
		if xh.Name() != host.Name() {
			continue
		}
		if xh.SSHURL().String() == host.SSHURL().String() {
			h.hosts = append(h.hosts[0:i], h.hosts[i+1:]...)
			log.Printf("Removed '%s' from history", host.Name())
			return h.Save()
		}
	}
	log.Printf("Item not in history: %v", host)
	return nil
}

// Hosts returns all the Hosts in History.
func (h *History) Hosts() []Host {
	if h.hosts == nil {
		h.Load()
		log.Printf("[source/load/history] %d host(s) in '%s'", len(h.hosts), h.Name())
	}
	return h.hosts
}

// Load loads the history from disk.
func (h *History) Load() error {
	if _, err := os.Stat(h.Filepath); err != nil {
		return nil
	}

	urls := []string{}
	data, err := ioutil.ReadFile(h.Filepath)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(data, &urls); err != nil {
		return err
	}
	h.hosts = make([]Host, len(urls))
	for i, s := range urls {
		u, err := url.Parse(s)
		if err != nil {
			return err
		}
		host := NewBaseHostFromURL(u)
		host.source = h.Name()
		if !h.d.IsDuplicate(host) {
			h.hosts[i] = host
			h.d.Add(host)
		}
	}

	return nil
}

// Save saves the History to disk.
func (h *History) Save() error {

	urls := make([]string, len(h.hosts))

	for i, host := range h.hosts {
		urls[i] = host.SSHURL().String()
	}

	data, err := json.MarshalIndent(urls, "", "  ")
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(h.Filepath, data, 0600); err != nil {
		return err
	}

	log.Printf("Saved %d host(s) to history", len(h.hosts))
	return nil
}
