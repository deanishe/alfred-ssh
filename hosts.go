//
// Copyright (c) 2016 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2016-12-11
//

package ssh

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var (
	hostnameRegex = regexp.MustCompile("^[a-zA-Z0-9.-]+$")
)

// Host is a host you can connect to.
type Host interface {
	UID() string                // Unique ID of host
	Name() string               // Display name of host
	Hostname() string           // Qualified hostname
	Port() int                  // Port (22 by default)
	SetPort(i int)              // Set the Port
	Source() string             // Display name of source
	Username() string           // Username if not default
	SetUsername(n string)       // Set Username
	CanonicalURL() *url.URL     // Canonical SSH URL
	SSHURL() *url.URL           // ssh:// URL for this host
	SFTPURL() *url.URL          // sftp:// URL for this host
	MoshCmd(path string) string // Command-line mosh command for this host
}

// Deduplicator recognises duplicate Hosts.
type Deduplicator struct {
	tags map[string]bool
}

// Add adds a new Host.
func (d *Deduplicator) Add(h Host) {
	if d.tags == nil {
		d.tags = map[string]bool{}
	}
	d.tags[h.UID()] = true
	// log.Printf("Host: %#v, UID: %s", h, h.UID())
	// d.tags[h.CanonicalURL().String()] = true
	// d.tags[h.SSHURL().String()] = true
}

// IsDuplicate returns true if Host is a duplicate.
func (d *Deduplicator) IsDuplicate(h Host) bool {
	if d.tags == nil {
		d.tags = map[string]bool{}
	}
	if dupe := d.tags[h.UID()]; dupe {
		return true
	}
	return false
}

// FilterDuplicateHosts removes duplicate Hosts.
func FilterDuplicateHosts(hosts []Host) []Host {
	clean := []Host{}
	d := &Deduplicator{}

	for _, h := range hosts {
		if d.IsDuplicate(h) {
			continue
		}
		clean = append(clean, h)
		d.Add(h)
	}
	return clean
}

// type jsonHost struct {
// 	Name     string `json:"name"`
// 	Hostname string `json:"hostname"`
// 	Source   string `json:"source"`
// 	Username string `json:"username"`
// 	Port     int
// }

// newJSONHost creates a jsonHost object for a Host.
// func newJSONHost(h Host) *jsonHost {
// 	return &jsonHost{
// 		h.Name(),
// 		h.Hostname(),
// 		h.Source(),
// 		h.Username(),
// 		h.Port(),
// 	}
// }

// BaseHost implements Host.
type BaseHost struct {
	name     string
	hostname string
	source   string
	username string
	port     int
}

// NewBaseHost creates a new BaseHost object.
func NewBaseHost(name, hostname, source, username string, port int) *BaseHost {
	return &BaseHost{name, hostname, source, username, port}
}

// NewBaseHostFromURL creates a new BaseHost object.
func NewBaseHostFromURL(u *url.URL) *BaseHost {
	h := &BaseHost{
		// name:     name,
		hostname: u.Host,
		source:   "URL",
		port:     22,
	}
	// Extract port from hostname
	if i := strings.Index(u.Host, ":"); i > -1 {
		h.hostname = u.Host[:i]
		if j, err := strconv.Atoi(u.Host[i+1:]); err == nil {
			h.port = j
		}
	}
	if u.User != nil {
		h.username = u.User.Username()
	}
	name := h.hostname
	if h.username != "" {
		name = h.username + "@" + name
	}
	if h.port != 22 {
		name = fmt.Sprintf("%s:%d", name, h.port)
	}
	h.name = name
	return h
}

// MarshalJSON exports BaseHost as JSON.
// func (h *BaseHost) MarshalJSON() ([]byte, error) {
// 	return json.MarshalIndent(newJSONHost(h), "", "  ")
// }

// UnmarshalJSON initialises a BaseHost from JSON.
// func (h *BaseHost) UnmarshalJSON(data []byte) error {
// 	j := jsonHost{}
// 	err := json.Unmarshal(data, &j)
// 	if err != nil {
// 		return err
// 	}
// 	h.name = j.Name
// 	h.hostname = j.Hostname
// 	h.source = j.Source
// 	h.username = j.Username
// 	h.port = j.Port
// 	return nil
// }

// UID implements Host.
func (h *BaseHost) UID() string { return UIDForHost(h) }

// Name implements Host.
func (h *BaseHost) Name() string { return h.name }

// Hostname implements Host.
func (h *BaseHost) Hostname() string { return h.hostname }

// Port implements Host.
func (h *BaseHost) Port() int {
	if h.port == 0 {
		return 22
	}
	return h.port
}

// SetPort implements Host.
func (h *BaseHost) SetPort(i int) { h.port = i }

// Source implements Host.
func (h *BaseHost) Source() string { return h.source }

// Username implements Host.
func (h *BaseHost) Username() string { return h.username }

// SetUsername implemeents Host.
func (h *BaseHost) SetUsername(n string) { h.username = n }

// CanonicalURL implements Host.
func (h *BaseHost) CanonicalURL() *url.URL {
	u := &url.URL{Scheme: "ssh", Host: h.Hostname()}
	if h.Username() != "" {
		u.User = url.User(h.Username())
	}
	if h.Port() == 22 {
		u.Host = h.Hostname()
	} else {
		u.Host = fmt.Sprintf("%s:%d", h.Hostname(), h.Port())
	}
	return u
}

// SSHURL implements Host.
func (h *BaseHost) SSHURL() *url.URL {
	u := h.CanonicalURL()
	u.Scheme = "ssh"
	return u
}

// SFTPURL implements Host.
func (h *BaseHost) SFTPURL() *url.URL {
	u := h.CanonicalURL()
	u.Scheme = "sftp"
	return u
}

// MoshCmd implements Host.
func (h *BaseHost) MoshCmd(path string) string {
	if path == "" {
		path = "mosh"
	}
	cmd := path + " "
	if h.Port() != 22 {
		cmd += fmt.Sprintf("--ssh 'ssh -p %d' ", h.Port())
	}
	if h.Username() != "" {
		cmd += h.Username() + "@"
	}
	cmd += h.Hostname()
	return cmd
}

// UIDForHost returns a UID for a Host.
func UIDForHost(h Host) string {
	uid := h.SSHURL().String()
	if h.Port() != 22 && strings.Index(h.SSHURL().Host, ":") < 0 {
		uid = fmt.Sprintf("%s:%d", uid, h.Port())
	}

	return fmt.Sprintf("%s||%s", h.Name(), uid)
}

// IsValidHostname returns true if n is an IP address or hostname.
func IsValidHostname(n string) bool {
	if ip := net.ParseIP(n); ip != nil {
		return true
	}
	return hostnameRegex.MatchString(n)
}
