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
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/havoc-io/ssh_config"
)

// ConfigHost is Host parsed from SSH config-format files.
type ConfigHost struct {
	BaseHost
	forcePort     bool
	forceUsername bool
}

// UID implements Host.
func (h *ConfigHost) UID() string { return UIDForHost(h) }

// SetPort implements Host.
func (h *ConfigHost) SetPort(i int) {
	h.port = i
	h.forcePort = true
}

// SetUsername implemeents Host.
func (h *ConfigHost) SetUsername(n string) {
	h.username = n
	h.forceUsername = true
}

// SSHURL returns a URL based on the Host value from the config file,
// *not* the Hostname.
func (h *ConfigHost) SSHURL() *url.URL {
	u := &url.URL{
		Scheme: "ssh",
		Host:   h.Name(),
	}
	if h.forcePort {
		u.Host = fmt.Sprintf("%s:%d", u.Host, h.Port())
	}
	if h.forceUsername {
		u.User = url.User(h.Username())
	}
	return u
}

// MoshCmd implements Host.
func (h *ConfigHost) MoshCmd(path string) string {
	if path == "" {
		path = "mosh"
	}
	cmd := path + " "
	if h.forcePort {
		cmd += fmt.Sprintf("--ssh 'ssh -p %d' ", h.Port())
	}
	if h.forceUsername && h.Username() != "" {
		cmd += h.Username() + "@"
	}
	cmd += h.Name()
	return cmd
}

// ConfigSource implements Source for ssh config-formatted files.
type ConfigSource struct {
	baseSource
}

// NewConfigSource creates a new ConfigSource from an ssh configuration file.
func NewConfigSource(path, name string, priority int) *ConfigSource {
	s := &ConfigSource{}
	s.Filepath = path
	s.name = name
	s.priority = priority
	return s
}

// Hosts implements Source.
func (s *ConfigSource) Hosts() []Host {
	if s.hosts == nil {
		hosts := parseConfigFile(s.Filepath)
		log.Printf("[source/load/config] %d host(s) in '%s'", len(hosts), s.Name())
		s.hosts = make([]Host, len(hosts))
		for i, h := range hosts {
			h.source = s.Name()
			s.hosts[i] = Host(h)
		}
	}
	return s.hosts
}

// maxIncludeDepth is OpenSSH's limit on nested Include directives.
const maxIncludeDepth = 16

// parseConfigFile parse an SSH config file, following Include directives.
func parseConfigFile(path string) []*ConfigHost {
	// OpenSSH resolves relative Include paths against ~/.ssh for the user
	// config and /etc/ssh for the system config, i.e. the top-level file's dir.
	return parseConfigFileDepth(path, filepath.Dir(path), 0, map[string]bool{})
}

// seen holds files already parsed, so each file's hosts are listed once and
// Include cycles (or several globs matching the same files) stay linear.
func parseConfigFileDepth(path, baseDir string, depth int, seen map[string]bool) []*ConfigHost {
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	if seen[path] {
		return nil
	}
	seen[path] = true

	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("[config/%s] Error opening file: %s", path, err)
		return nil
	}
	body, includes := splitIncludes(string(data))
	hosts := parseConfig(path, body)
	if len(includes) > 0 && depth >= maxIncludeDepth {
		log.Printf("[config/%s] Include nested too deeply, ignoring", path)
		return hosts
	}
	for _, pattern := range includes {
		for _, p := range expandInclude(pattern, baseDir) {
			hosts = append(hosts, parseConfigFileDepth(p, baseDir, depth+1, seen)...)
		}
	}
	return hosts
}

// splitIncludes removes Include directives from an SSH config, returning
// the remaining config and the Include arguments (paths or glob patterns).
func splitIncludes(config string) (string, []string) {
	var includes []string
	lines := strings.Split(config, "\n")
	for i, line := range lines {
		// Keyword and arguments are separated by whitespace and/or one "=".
		line = strings.TrimSpace(line)
		end := strings.IndexAny(line, " \t=")
		if end < 0 || !strings.EqualFold(line[:end], "include") {
			continue
		}
		rest := strings.TrimLeft(line[end:], " \t")
		rest = strings.TrimPrefix(rest, "=")
		includes = append(includes, splitArgs(rest)...)
		lines[i] = ""
	}
	return strings.Join(lines, "\n"), includes
}

// splitArgs splits on whitespace, keeping "double quoted" arguments whole.
func splitArgs(s string) []string {
	var (
		args    []string
		current strings.Builder
		quoted  bool
		inArg   bool
	)
	for _, r := range s {
		switch {
		case r == '"':
			quoted = !quoted
			inArg = true
		case (r == ' ' || r == '\t') && !quoted:
			if inArg {
				args = append(args, current.String())
				current.Reset()
				inArg = false
			}
		default:
			current.WriteRune(r)
			inArg = true
		}
	}
	if inArg {
		args = append(args, current.String())
	}
	return args
}

// expandInclude resolves an Include argument to the files it matches.
func expandInclude(pattern, baseDir string) []string {
	if strings.HasPrefix(pattern, "~/") {
		pattern = filepath.Join(os.Getenv("HOME"), pattern[2:])
	} else if !filepath.IsAbs(pattern) {
		pattern = filepath.Join(baseDir, pattern)
	}
	// Glob returns matches in lexical order, which is the order OpenSSH uses.
	matches, err := filepath.Glob(pattern)
	if err != nil {
		log.Printf("[config] Bad Include pattern %q: %s", pattern, err)
	}
	var files []string
	for _, m := range matches {
		if !hiddenByGlob(pattern, m) {
			files = append(files, m)
		}
	}
	return files
}

// hiddenByGlob reports whether match has a dot-file component that pattern
// did not spell out with a leading ".". glob(3), which OpenSSH uses, never
// lets a wildcard match a leading dot; filepath.Glob does.
func hiddenByGlob(pattern, match string) bool {
	pp := strings.Split(filepath.Clean(pattern), string(filepath.Separator))
	mp := strings.Split(filepath.Clean(match), string(filepath.Separator))
	if len(pp) != len(mp) {
		return false
	}
	for i := range mp {
		if strings.HasPrefix(mp[i], ".") && !strings.HasPrefix(pp[i], ".") {
			return true
		}
	}
	return false
}

// parseConfig parses the hosts in an SSH config (without Include lines).
func parseConfig(path, body string) []*ConfigHost {
	var hosts []*ConfigHost
	cfg, err := ssh_config.Parse(strings.NewReader(body))
	if err != nil {
		log.Printf("[config/%s] Parse error: %s", path, err)
		return hosts
	}

	for _, e := range cfg.Hosts {
		var (
			p    *ssh_config.Param
			port = 22
			hn   string // hostname
			user string
		)

		p = e.GetParam(ssh_config.HostKeyword)
		if p != nil {
			hn = p.Value()
		}

		// log.Println(e.String())
		// log.Printf("hostnames=%v", e.Hostnames)

		p = e.GetParam(ssh_config.HostNameKeyword)
		if p != nil {
			hn = p.Value()
		}

		p = e.GetParam(ssh_config.PortKeyword)
		if p != nil {
			port, err = strconv.Atoi(p.Value())
			if err != nil {
				log.Printf("Bad port: %s", err)
				port = 22
			}
		}
		// log.Printf("port=%v", port)

		p = e.GetParam(ssh_config.UserKeyword)
		if p != nil {
			user = p.Value()
		}

		for _, n := range e.Hostnames {
			if strings.Contains(n, "*") || strings.Contains(n, "!") || strings.Contains(n, "?") {
				continue
			}

			h := &ConfigHost{}
			h.name = n
			h.hostname = n
			h.port = port
			h.username = user

			if hn != "" {
				h.hostname = hn
			}
			// log.Printf("%+v", host)
			hosts = append(hosts, h)
		}
	}
	return hosts
}
