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

// The dhcphosts package provide utilities to work files in dhcphosts format
// (see man 8 dnsmasq)
// This package use naive linear search and no lookup optimizations (e.g. maps, skipslists).
// Rationale:
// - we expect to work with maximum ~1000 entries (*one* thousand, not thousand*S*)
// - everything is in memory anyway
// - linear search is simpler to implement/maintain

package dhcphosts

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
	DuplicateFound   error = errors.New("Entry already found")
)

// Binding represents the binding between a MAC and an IP
type Binding struct {
	HW net.HardwareAddr
	IP net.IP
}

// ParseBindingString parses a string in the dhcphosts format (man 8 dnsmasq) and returns a Binding
func ParseBindingString(s string) (Binding, error) {
	a := strings.Split(s, ",")
	if len(a) != 2 {
		return Binding{}, BadBindingFormat
	}
	return ParseBinding(a[0], a[1])
}

// ParseBinding creates a Binding between a MAC and a IP, expressed as strings
func ParseBinding(hw, ip string) (Binding, error) {
	hwAddr, err := net.ParseMAC(hw)
	if err != nil {
		return Binding{}, BadHWAddrFormat
	}
	ipAddr := net.ParseIP(ip)
	if ipAddr == nil {
		return Binding{}, BadIPFormat
	}
	return Binding{
		HW: hwAddr,
		IP: ipAddr,
	}, nil
}

// EqualHW returns true if the Binding has the MAC part equal to the argument, false otherwise
func (b Binding) EqualHW(x net.HardwareAddr) bool {
	return bytes.Equal(b.HW, x)
}

// EqualIP returns true if the Binding has the IP part equal to the argument, false otherwise
func (b Binding) EqualIP(x net.IP) bool {
	return b.IP.Equal(x)
}

// Equal returns true if the Binding is equal to the argument, false otherwise
func (b Binding) Equal(x Binding) bool {
	return b.EqualHW(x.HW) && b.EqualIP(x.IP)
}

func (b Binding) Duplicate(x Binding) bool {
	return b.Equal(x)
}

// String converts the binding in its dhcphosts (man 8 dnsmasq) representation
func (b Binding) String() string {
	return fmt.Sprintf("%s,%s", b.HW.String(), b.IP.String())
}

// Conf represents the configured Bindings
type Conf struct {
	lock sync.RWMutex
	// linear search isn't O(1), but it is more than enough for the kind of load we expect
	bindings []Binding
}

// Len returns the number of configured Bindings
func (m *Conf) Len() int {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return len(m.bindings)
}

// String converts all the registered bindings in the Conf in content in dhcphosts (man 8 dnsmasq) representation
func (m *Conf) String() string {
	var sb strings.Builder
	m.lock.RLock()
	defer m.lock.RUnlock()
	for _, bi := range m.bindings {
		sb.WriteString(fmt.Sprintf("%s\n", bi.String()))
	}
	return sb.String()
}

// Add registers a new Binding
func (m *Conf) Add(b Binding) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.add(b)
}

func (m *Conf) duplicate(x Binding) *Binding {
	for _, b := range m.bindings {
		if b.Duplicate(x) {
			return &b
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

// Add forgets a Binding previously Add()ed
func (m *Conf) Delete(b Binding) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	var bindings []Binding
	for _, bi := range m.bindings {
		if bi.Equal(b) {
			continue
		}
		bindings = append(bindings, bi)
	}
	m.bindings = bindings
	return nil
}

func find(data findable, x interface{}, err error) (interface{}, error) {
	for i := 0; i < data.Len(); i++ {
		bi := data.Get(i)
		if data.Equal(bi, x) {
			return bi, nil
		}
	}
	return data.Empty(), err
}

// inspired by the sort package
type findable interface {
	Len() int
	Get(i int) interface{}
	Equal(a, b interface{}) bool
	Empty() interface{}
}

type byHW []Binding

func (a byHW) Len() int                    { return len(a) }
func (a byHW) Get(i int) interface{}       { return a[i] }
func (a byHW) Empty() interface{}          { return Binding{} }
func (a byHW) Equal(x, y interface{}) bool { return x.(Binding).EqualHW(y.(Binding).HW) }

type byIP []Binding

func (a byIP) Len() int                    { return len(a) }
func (a byIP) Get(i int) interface{}       { return a[i] }
func (a byIP) Empty() interface{}          { return Binding{} }
func (a byIP) Equal(x, y interface{}) bool { return x.(Binding).EqualIP(y.(Binding).IP) }

func (m *Conf) GetByHWAddr(hw string) (Binding, error) {
	x, err := net.ParseMAC(hw)
	if err != nil {
		return Binding{}, err
	}
	m.lock.RLock()
	defer m.lock.RUnlock()
	ret, err := find(byHW(m.bindings), Binding{HW: x}, HWAddrNotFound)
	return ret.(Binding), err
}

func (m *Conf) GetByIP(ip string) (Binding, error) {
	x := net.ParseIP(ip)
	if x == nil {
		return Binding{}, BadIPFormat
	}
	m.lock.RLock()
	defer m.lock.RUnlock()
	ret, err := find(byIP(m.bindings), Binding{IP: x}, IPAddrNotFound)
	return ret.(Binding), err
}

// Parse creates a Conf from a reader, which must return content in dhcphosts (man 8 dnsmasq) format
func Parse(r io.Reader) (*Conf, error) {
	m := &Conf{}
	s := bufio.NewScanner(r)
	for s.Scan() {
		b, err := ParseBindingString(s.Text())
		if err != nil {
			return nil, err
		}
		m.add(b)
	}
	return m, nil
}
