#!/usr/bin/env python
# encoding: utf-8
#
# Copyright (c) 2016 Dean Jackson <deanishe@deanishe.net>
#
# MIT Licence. See http://opensource.org/licenses/MIT
#
# Created on 2016-05-26
#

"""Generate some random host."""

from __future__ import print_function, unicode_literals, absolute_import

from random import choice, randint, shuffle

COUNT = 30

domains = """\
bartell.com
fay-king.com
gierschner.org
johann.com
junitz.com
kassulke-spencer.com
kessler.com
kostolzin.de
kuhlman-wolf.info
kulas-douglas.biz
larson-schumm.info
lind-sipes.com
lockman.com
losekann.com
maelzer.org
mayer.biz
reinger.info
roemer.com
scholz.net
sipes.com
trapp.com
wesack.com
zorbach.com
""".strip().split('\n')

subdomains = """\
mail
www
vpn
ftp
gateway
""".strip().split('\n')

servers = """\
antonetta
balduin
elias
ermanno
ethelyn
fechner
froehlich
gene
heinz
heser
holsten
hornich
iliana
kadeem
kurt
leslee
meyer
moesha
monja
reiner
roland
russel
rust
sandy
teobaldo
valerius
wulff
zaida
""".strip().split('\n')


def domain(host):
    """Return domain of `host`."""
    return host.split('.', 1)[1]


def main():
    """Run script."""
    shuffle(servers)

    hosts = []
    for i in range(COUNT):
        d = choice(domains)
        if randint(0, 1):
            s = servers.pop()
        else:
            s = choice(subdomains)
        hosts.append('{}.{}'.format(s, d))

    for h in sorted(hosts, key=domain):
        print('"{}",'.format(h))

if __name__ == '__main__':
    main()
