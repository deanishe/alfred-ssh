//
// Copyright (c) 2016 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2016-05-23
//

package ssh

// Useful for screenshots
var testHostnames = []string{
	"kurt.bartell.com",
	"mail.gierschner.org",
	"vpn.johann.com",
	"hornich.junitz.com",
	"gene.kassulke-spencer.com",
	"roland.kassulke-spencer.com",
	"ethelyn.kassulke-spencer.com",
	"mail.kostolzin.de",
	"gateway.kuhlman-wolf.info",
	"www.kuhlman-wolf.info",
	"monja.kuhlman-wolf.info",
	"ftp.kuhlman-wolf.info",
	"ermanno.kulas-douglas.biz",
	"www.lind-sipes.com",
	"zaida.lind-sipes.com",
	"antonetta.lockman.com",
	"valerius.lockman.com",
	"gateway.losekann.com",
	"leslee.losekann.com",
	"ftp.mayer.biz",
	"reiner.roemer.com",
	"mail.roemer.com",
	"gateway.scholz.net",
	"vpn.sipes.com",
	"mail.sipes.com",
	"wulff.sipes.com",
	"elias.wesack.com",
	"gateway.wesack.com",
	"heinz.zorbach.com",
}

// TestHosts loads fake test data instead of real hosts.
func TestHosts() []Host {
	hosts := make([]Host, len(testHostnames))

	for i, name := range testHostnames {
		hosts[i] = &BaseHost{
			name:     name,
			hostname: name,
			source:   "test data",
			port:     22,
		}
	}

	return hosts
}
