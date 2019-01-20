# dnsmasqmgr
dnsmasqmgr provides a HTTP endpoint to manage dnsmasqd lease

## DISCLAIMER AND USAGE WARNING
I -the author- use this package to manage my own `dnsmasq` instances in my LAN setup.
While I try to avoid obvious security missteps, this package is *not* meant to be use
in setups with any vaguely important, yet alone sensitive information, or in public-facing
networks. *Use at your own risk*.

## Description
`dnsmasqmgr` is a package that helps you query and manage the [dnsmasq](http://www.thekelleys.org.uk/dnsmasq/doc.html) lease configuration.
The key components are:
- `dnsmasqmgrd` is the cornerstone server which providess a HTTP endpoint to query and modify the lease configuration file.
- `dnsmasqreloadd` is an utility server that restarts `dnsmasqd` when the configuration changes.
- `dnsmasqmgr` is both a `golang` package and a command line utility to interact with `dnsmasqmgrd` across a LAN

## Configuration and setup
WRITEME

## Usage tips
WRITEME

## Docker image
WRITEME

## License/Copyright
(C) 2019 Francesco Romani - write me @gmail
License: MIT
