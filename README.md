# dnsmasqmgr
dnsmasqmgr provides an endpoint to manage dnsmasq's DNS and DHCP lease services

## Description
`dnsmasqmgr` is a package that helps you query and manage the [dnsmasq](http://www.thekelleys.org.uk/dnsmasq/doc.html) dhcp host configuration
- see the options `--dhcp-host` and `--dhcp-hostsfile` of `dnsmasq`.

The key components are:
- `dnsmasqmgrd` is the cornerstone server which providess a HTTP endpoint to query and modify the lease configuration file.
- `dnsmasqmgr` is both a `golang` package and a command line utility to interact with `dnsmasqmgrd` across a LAN
- `dnsmasqreloadd` is an utility service that makes `dnsmasq` reload the configuration once changes. It's the only component that requires system privileges

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
useradd -c "dnsmasqmgr" -d /var/lib/dnsmasqmgr/ -M -r -s /sbin/nologin dnsmasqmgr
```
`helpers/mkuser.sh` automates this step for you.

2. create a directory like `/var/lib/dnsmasqmgr` which is *readable* by dnsmasqd and *writable only* by the user/group set previously.
```bash
mkdir -p /var/lib/dnsmasqmgr/conf.d/hosts.d
mkdir -p /var/lib/dnsmasqmgr/conf.d/dhcp.d
chown -R dnsmasqmgr:dnsmasqmgr /var/lib/dnsmasqmgr
find /var/lib/dnsmasqmgr -type d | xargs chmod 0755
find /var/lib/dnsmasqmgr -type f | xargs chmod 0644
```
`helpers/mktree.sh` automates this step for you.

2.1. fix SELinux permissions (optional)

TODO

3. configure `dnsmasq` to integrate with `dnsmasqmgrd`. Highlight of the needed options
```bash
addn-hosts=/etc/hosts.d
addn-hosts=/var/lib/dnsmasqmgr/conf.d/hosts.d
dhcp-hostsfile=/var/lib/dnsmasqmgr/conf.d/dhcp.d/leases
dhcp-range=192.168.1.2,192.168.1.128,12h
```
Note the difference: we use `addn-hosts` but `dhcp-hostsfile`

4. let `dnsmasqmgrd` run, using the provided systemd unit or any other mean
5. let `dnsmasqreloadd` run, using the provided systemd unit or any other mean
6. interact with `dnsmasqmgrd` using the API or using `dnsmasqmgr` go package or command line tool

## API
see `pkg/dnsmasqmgr/dnsmasqmgr.proto`

## Container image
Not supported. Patches welcome.
