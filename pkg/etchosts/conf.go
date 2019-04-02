/*
 * Copyright 2019 Francesco Romani - fromani/gmail
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy of this
 * software and associated documentation files (the "Software"), to deal in the Software
 * without restriction, including without limitation the rights to use, copy, modify,
 * merge, publish, distribute, sublicense, and/or sell copies of the Software, and to
 * permit persons to whom the Software is furnished to do so, subject to the following
 * conditions:
 *
 * The above copyright notice and this permission notice shall be included in all copies
 * or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED,
 * INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A
 * PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT
 * HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
 * OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
 * SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
 */

// The etchosts package provide utilities manage /etc/hosts-like file (see man 5 hosts)
// This package use naive linear search and no lookup optimizations (e.g. maps, skipslists).
// Rationale:
// - we expect to work with maximum ~1000 entries (*one* thousand, not thousand*S*)
// - everything is in memory anyway
// - linear search is simpler to implement/maintain

package etchosts

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
)

var (
	ErrBadHWAddrFormat  error = errors.New("Malformed hardware address address")
	ErrBadIPFormat      error = errors.New("Malformed IP address")
	ErrBadEntryFormat   error = errors.New("Malformed entry")
	ErrMissingHostname  error = errors.New("Missing hostname")
	ErrDuplicate        error = errors.New("Duplicated entry")
	ErrNotFoundHostname error = errors.New("Hostname not found in the hostsfile")
	ErrNotFoundAddress  error = errors.New("Address not found in the hostsfile")
)

// Host represents a single entry in the /etc/hosts file
type Host struct {
	Address           net.IP
	CanonicalHostname string
	Aliases           []string
}

func ParseHostString(s string) (Host, error) {
	s = strings.Replace(s, "\t", " ", -1)
	items := strings.Split(s, " ")
	if len(items) < 2 {
		return Host{}, ErrBadEntryFormat
	}
	var aliases []string
	if len(items) > 2 {
		aliases = items[2:]
	}
	return ParseHost(items[0], items[1], aliases)
}

func ParseHost(addr, name string, aliases []string) (Host, error) {
	h := Host{
		Address:           net.ParseIP(addr),
		CanonicalHostname: name,
		Aliases:           aliases,
	}
	var err error
	if h.Address == nil {
		err = ErrBadIPFormat
	}
	return h, err
}

func (h Host) String() string {
	sep := ""
	aliases := ""
	if len(h.Aliases) > 0 {
		sep = "\t"
		aliases = strings.Join(h.Aliases, " ")
	}
	return fmt.Sprintf("%s\t%s%s%s", h.Address, h.CanonicalHostname, sep, aliases)
}

func (h Host) Equal(x Host) bool {
	return h.Address.Equal(x.Address)
}

func (h Host) Duplicate(x Host) bool {
	return h.findDuplicate(x) != ""
}

func (h Host) findDuplicate(x Host) string {
	if h.CanonicalHostname == x.CanonicalHostname {
		return x.CanonicalHostname
	}
	if h.Address.Equal(x.Address) {
		return x.Address.String()
	}
	numAliases := len(h.Aliases)
	if len(x.Aliases) < len(h.Aliases) {
		numAliases = len(x.Aliases)
	}
	for ix := 0; ix < numAliases; ix++ {
		if h.Aliases[ix] == x.Aliases[ix] {
			return x.Aliases[ix]
		}
	}
	return ""
}

// Conf represents the configured Bindings
type Conf struct {
	lock sync.RWMutex
	// this is not really for efficiency, even though it's a nice plus,
	// but rather  because Hostname is the key here.
	hosts map[string]Host
}

func NewConf() *Conf {
	return &Conf{
		hosts: make(map[string]Host),
	}
}

// Len returns the number of configured Bindings
func (m *Conf) Len() int {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return len(m.hosts)
}

// String converts all the registered hosts in the Conf in content in etchosts (man 8 dnsmasq) representation
func (m *Conf) String() string {
	var sb strings.Builder
	m.lock.RLock()
	defer m.lock.RUnlock()
	for _, h := range m.hosts {
		sb.WriteString(fmt.Sprintf("%s\n", h.String()))
	}
	return sb.String()
}

func (m *Conf) duplicate(x Host) *Host {
	for _, h := range m.hosts {
		if what := h.findDuplicate(x); what != "" {
			log.Printf("etchosts: [%s] duplicates [%s] on %s", x, h, what)
			return &h
		}
	}
	return nil
}

func (m *Conf) add(h Host) error {
	if x := m.duplicate(h); x != nil {
		return fmt.Errorf("%s: %s", ErrDuplicate, x)
	}
	m.hosts[h.CanonicalHostname] = h
	log.Printf("etchosts: added [[%s]]", h)
	return nil
}

func (m *Conf) Add(name, addr string, aliases []string) (Host, error, bool) {
	if name == "" {
		return Host{}, ErrMissingHostname, false
	}
	ret := Host{
		CanonicalHostname: name,
		Address:           net.ParseIP(addr),
	}
	if ret.Address == nil {
		return ret, ErrBadIPFormat, false
	}
	for _, alias := range aliases {
		ret.Aliases = append(ret.Aliases, alias)
	}
	m.lock.Lock()
	defer m.lock.Unlock()
	err := m.add(ret)
	return ret, err, err != nil
}

func (m *Conf) GetByAddress(addr string) (Host, error) {
	var ret Host
	var err error = ErrNotFoundAddress
	var ipAddr = net.ParseIP(addr)
	if ipAddr == nil {
		return ret, ErrBadIPFormat
	}

	defer func() {
		log.Printf("etchosts: GetByAddress(%s) -> (%s, %v)", addr, ret, err)
	}()
	for _, h := range m.hosts {
		if h.Address.Equal(ipAddr) {
			ret = h
			err = nil
			break
		}
	}
	return ret, err
}

func (m *Conf) GetByHostname(name string) (Host, error) {
	var ret Host
	var err error = ErrNotFoundHostname
	defer func() {
		log.Printf("etchosts: GetByHostname(%s) -> (%s, %v)", name, ret, err)
	}()
	ret, ok := m.hosts[name]
	if ok {
		err = nil
	}
	return ret, err
}

func (m *Conf) GetByAlias(alias string) (Host, error) {
	return Host{}, nil
}

// Parse creates a Conf from a reader, which must return content in etchosts (man 5 hosts) format
func Parse(r io.Reader) (*Conf, error) {
	m := NewConf()
	s := bufio.NewScanner(r)
	for s.Scan() {
		var err error
		line := s.Text()
		h, err := ParseHostString(line)
		if err != nil {
			log.Printf("etchosts: error parsing '%s': %v", line, err)
			continue
		}

		err = m.add(h)
		if err != nil {
			log.Printf("etchosts: error adding '%s': %v", h, err)
			continue
		}
	}
	return m, nil
}
