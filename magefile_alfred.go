// Copyright (c) 2019 Dean Jackson <deanishe@deanishe.net>
// MIT Licence applies http://opensource.org/licenses/MIT

// +build mage

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"howett.net/plist"
)

var (
	alfredPrefsPath   = os.ExpandEnv("${HOME}/Library/Preferences/com.runningwithcrayons.Alfred-Preferences-3.plist")
	defaultSyncFolder = os.ExpandEnv("~/Library/Application Support/Alfred 3")

	// Read from info.plist
	BundleID string
	Version  string
	Name     string

	DataDir     string
	CacheDir    string
	WorkflowDir string
)

func init() {
	home := os.ExpandEnv("$HOME")
	ip := readInfo()

	BundleID = ip.BundleID
	if BundleID == "" {
		panic("Bundle ID is unset")
	}

	Version = ip.Version
	Name = ip.Name

	CacheDir = filepath.Join(home, "Library/Caches/com.runningwithcrayons.Alfred-3/Workflow Data", ip.BundleID)
	DataDir = filepath.Join(home, "Library/Application Support/Alfred 3/Workflow Data", ip.BundleID)
	WorkflowDir = workflowDirectory()
}

type infoPlist struct {
	BundleID string `plist:"bundleid"`
	Version  string `plist:"version"`
	Name     string `plist:"name"`
}

func syncFolder() string {

	data, err := ioutil.ReadFile(alfredPrefsPath)
	if err != nil {
		panic(err)
	}

	p := struct {
		SyncFolder string `plist:"syncfolder"`
	}{}

	if _, err := plist.Unmarshal(data, &p); err != nil {
		panic(err)
	}

	return p.SyncFolder
}

func workflowDirectory() string {

	dirs := []string{}

	if p := syncFolder(); p != "" {
		dirs = append(dirs, p)
	}
	dirs = append(dirs, defaultSyncFolder)

	for _, p := range dirs {
		p = expandPath(filepath.Join(p, "Alfred.alfredpreferences/workflows"))
		if _, err := os.Stat(p); err != nil {
			fmt.Printf("read %q: %v\n", p, err)
			continue
		}
		return p
	}
	panic("workflow directory not found")
}

func readInfo() infoPlist {
	data, err := ioutil.ReadFile("info.plist")
	if err != nil {
		panic(err)
	}

	ip := infoPlist{}
	if _, err := plist.Unmarshal(data, &ip); err != nil {
		panic(err)
	}

	return ip
}

func alfredEnv() map[string]string {
	return map[string]string{
		"alfred_workflow_bundleid": BundleID,
		"alfred_workflow_version":  Version,
		"alfred_workflow_name":     Name,
		"alfred_workflow_cache":    CacheDir,
		"alfred_workflow_data":     DataDir,
		"GO111MODULE":              "on", // for building
	}
}

// expand ~ and variables in path.
func expandPath(path string) string {
	if strings.HasPrefix(path, "~") {
		path = "${HOME}" + path[1:]
	}

	return os.ExpandEnv(path)
}

func exists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false
		}
		panic(err)
	}

	return true
}
