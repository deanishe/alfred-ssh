//
// Copyright (c) 2016 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2016-05-23
//

package assh

// Useful for screenshots
var testHostnames = []string{
	"api.dehmel.de",
	"api.gleichner.info",
	"api.hartung.de",
	"api.hayes-padberg.com",
	"api.littel-zieme.com",
	"api.rohleder.org",
	"api.rosemann.de",
	"api.senger-marquardt.com",
	"api.streich.com",
	"api.tintzmann.com",
	"api.weitzel.org",
	"imap.hartung.de",
	"imap.littel-zieme.com",
	"imap.roberts-collins.org",
	"imap.roehrdanz.de",
	"imap.rosemann.de",
	"imap.senger-beier.com",
	"imap.senger-marquardt.com",
	"imap.vogt.de",
	"imap.weitzel.org",
	"mail.dehmel.de",
	"mail.hartung.de",
	"mail.hayes-padberg.com",
	"mail.holzapfel.de",
	"mail.rohleder.org",
	"mail.schmitt.info",
	"mail.tintzmann.com",
	"mail.ziemann.info",
	"smtp.considine-johnston.com",
	"smtp.dehmel.de",
	"smtp.gleichner.info",
	"smtp.hartung.de",
	"smtp.hayes-padberg.com",
	"smtp.holzapfel.de",
	"smtp.littel-zieme.com",
	"smtp.senger-marquardt.com",
	"smtp.tintzmann.com",
	"smtp.vogt.de",
	"smtp.ziemann.info",
	"www.carsten.org",
	"www.considine-johnston.com",
	"www.hayes-padberg.com",
	"www.littel-zieme.com",
	"www.roehrdanz.de",
	"www.senger-beier.com",
	"www.tintzmann.com",
	"www.weitzel.org",
	"www.ziemann.info",
}

// TestHosts loads fake test data instead of real hosts.
func TestHosts() []*Host {
	hosts := make([]*Host, len(testHostnames))

	for i, name := range testHostnames {
		hosts[i] = &Host{name, 22, "test data", ""}
	}

	return hosts
}
