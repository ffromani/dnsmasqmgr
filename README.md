# dnsmasqmgr
dnsmasqmgr provides a HTTP endpoint to manage dnsmasqd lease

## Description
`dnsmasqmgr` is a package that helps you query and manage the [dnsmasq](http://www.thekelleys.org.uk/dnsmasq/doc.html) dhcp host configuration
- see the options `--dhcp-host` and `--dhcp-hostsfile` of `dnsmasq`.

The key components are:
- `dnsmasqmgrd` is the cornerstone server which providess a HTTP endpoint to query and modify the lease configuration file.
- `dnsmasqreloadd` is an utility service that makes `dnsmasq` reload the configuration once changes. It's the only componet that requires system privileges
- `dnsmasqmgr` is both a `golang` package and a command line utility to interact with `dnsmasqmgrd` across a LAN

## License/Copyright
(C) 2019 Francesco Romani - write me @gmail
License: MIT

## DISCLAIMER AND USAGE WARNING
I -the author- use this package to manage my own `dnsmasq` instances in my LAN setup.
While I try to avoid obvious security missteps, this package is *not* meant to be use
in setups with any vaguely important, yet alone sensitive information, or in public-facing
networks. *Use at your own risk*.

## Configuration and setup
The suggested setup is:
1. create user and group for the `dnsmasqmgrd` service.
```bash
```
2. create a directory like `/var/lib/dnsmasqmgr` which is *readable* by dnsmasqd and *writable only* by the user/group set previously.
```bash
```
3. configure `dnsmasq` to to use `/var/lib/dnsmasqmgr/hostinfo` as `hostsfile`
```bash
```
4. let `dnsmasqmgrd` run, using the provided systemd unit or any other mean
5. let `dnsmasqreloadd` run, using the provided systemd unit or any other mean
6. interact with `dnsmasqmgrd` using the HTTP API or using `dnsmasqmgr` go package or command line tool

## HTTP API
see `doc/api.md`

## Docker image
WRITEME
