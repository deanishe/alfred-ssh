// Copyright (c) 2019 Dean Jackson <deanishe@deanishe.net>
// MIT Licence applies http://opensource.org/licenses/MIT

// +build mage

package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"howett.net/plist"
)

var (
	// Home    = os.ExpandEnv("$HOME")
	Library = os.ExpandEnv("$HOME/Library")

	// Workflow configuration
	// Read from info.plist
	BundleID string
	Version  string
	Name     string

	PrefsFile   string
	SyncFolder  string
	DataDir     string
	CacheDir    string
	WorkflowDir string
	AppVersion  string

	defaultSyncFolder string
)

func init() {
	ip := readInfo()

	BundleID = ip.BundleID
	if BundleID == "" {
		panic("Bundle ID is unset")
	}

	Version = ip.Version
	Name = ip.Name

	PrefsFile = filepath.Join(Library, "Preferences/com.runningwithcrayons.Alfred-Preferences.plist")
	if _, err := os.Stat(PrefsFile); err == nil {
		CacheDir = filepath.Join(Library, "Caches/com.runningwithcrayons.Alfred/Workflow Data", ip.BundleID)
		DataDir = filepath.Join(Library, "Application Support/Alfred/Workflow Data", ip.BundleID)
		defaultSyncFolder = filepath.Join(Library, "Application Support/Alfred")

	} else {
		AppVersion = "3.8.1"
		PrefsFile = filepath.Join(Library, "Preferences/com.runningwithcrayons.Alfred-Preferences-3.plist")
		CacheDir = filepath.Join(Library, "Caches/com.runningwithcrayons.Alfred-3/Workflow Data", ip.BundleID)
		DataDir = filepath.Join(Library, "Application Support/Alfred 3/Workflow Data", ip.BundleID)
		defaultSyncFolder = filepath.Join(Library, "Application Support/Alfred 3")

	}
	SyncFolder = syncFolder()
	WorkflowDir = filepath.Join(SyncFolder, "Alfred.alfredpreferences/workflows")
}

type infoPlist struct {
	BundleID string `plist:"bundleid"`
	Version  string `plist:"version"`
	Name     string `plist:"name"`
}

func syncFolder() string {

	var (
		dirs = []string{defaultSyncFolder}
		data []byte
		err  error
	)

	if data, err = ioutil.ReadFile(PrefsFile); err != nil {
		panic(err)
	}

	p := struct {
		SyncFolder string `plist:"syncfolder"`
	}{}

	if _, err = plist.Unmarshal(data, &p); err != nil {
		panic(err)
	}

	if p.SyncFolder != "" {
		dirs = append([]string{p.SyncFolder}, dirs...)
	}

	for _, p := range dirs {
		p = expandPath(p)
		if exists(p) {

			return p
		}
	}

	panic("syncfolder not found")
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
		"alfred_version":           AppVersion,
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
