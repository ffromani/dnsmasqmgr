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
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
)

var (
	HWAddrNotFound   error = errors.New("hardware address not found")
	IPAddrNotFound   error = errors.New("IP address not found")
	BadHWAddrFormat  error = errors.New("Malformed hardware address address")
	BadIPFormat      error = errors.New("Malformed IP address")
	BadBindingFormat error = errors.New("Malformed Binding Pair")
)

// Host represents a single entry in the /etc/hosts file
type Host struct {
	Address           net.IP
	CanonicalHostname string
	Aliases           []string
}

func ParseHostString(s string) (Host, error) {
	return Host{}, nil
}

func ParseHost(addr, name string, aliases []string) (Host, error) {
	return Host{}, nil
}

func (h Host) String() string {
	return ""
}

func (h Host) Equal(x Host) bool {
	return h.Address.Equal(x.Address)
}

func (h Host) Duplicate(x Host) bool {
	if h.CanonicalHostname == x.CanonicalHostname {
		return true
	}
	if h.Address.Equal(x.Address) {
		return true
	}
	numAliases := len(h.Aliases)
	if len(x.Aliases) < len(h.Aliases) {
		numAliases = len(x.Aliases)
	}
	for ix := 0; ix < numAliases; ix++ {
		if h.Aliases[ix] == x.Aliases[ix] {
			return true
		}
	}
	return false
}

// Conf represents the configured Bindings
type Conf struct {
	lock sync.RWMutex
	// linear search isn't O(1), but it is more than enough for the kind of load we expect
	hosts []Host
}

// Len returns the number of configured Bindings
func (m *Conf) Len() int {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return len(m.hosts)
}

// String converts all the registered bindings in the Conf in content in etchosts (man 8 dnsmasq) representation
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
		if h.Duplicate(x) {
			return &h
		}
	}
	return nil
}

func (m *Conf) add(b Binding) error {
	if x := m.duplicate(b); x != nil {
		return fmt.Errorf("%s: %s", DuplicateFound, x)
	}
	m.bindings = append(m.bindings, b)
	return nil
}

func (m *Conf) GetByAddress(addr string) (Host, error) {
	return Host{}, nil
}

func (m *Conf) GetByHostname(name string) (Host, error) {
	return Host{}, nil
}

func (m *Conf) GetByAlias(alias string) (Host, error) {
	return Host{}, nil
}

// Parse creates a Conf from a reader, which must return content in etchosts (man 5 hosts) format
func Parse(r io.Reader) (*Conf, error) {
	m := &Conf{}
	s := bufio.NewScanner(r)
	for s.Scan() {
		b, err := ParseHostString(s.Text())
		if err != nil {
			return nil, err
		}
		m.add(b)
	}
	return m, nil
}
