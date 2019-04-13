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
	"log"
	"net"
	"strings"
)

var (
	ErrHWAddrNotFound   error = errors.New("hardware address not found")
	ErrIPAddrNotFound   error = errors.New("IP address not found")
	ErrBadHWAddrFormat  error = errors.New("Malformed hardware address address")
	ErrBadIPFormat      error = errors.New("Malformed IP address")
	ErrBadBindingFormat error = errors.New("Malformed Binding Pair")
	ErrDuplicateFound   error = errors.New("Entry already found")
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
		return Binding{}, ErrBadBindingFormat
	}
	return ParseBinding(a[0], a[1])
}

// ParseBinding creates a Binding between a MAC and a IP, expressed as strings
func ParseBinding(hw, ip string) (Binding, error) {
	hwAddr, err := net.ParseMAC(hw)
	if err != nil {
		return Binding{}, ErrBadHWAddrFormat
	}
	ipAddr := net.ParseIP(ip)
	if ipAddr == nil {
		return Binding{}, ErrBadIPFormat
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
	return b.EqualHW(x.HW)
}

// String converts the binding in its dhcphosts (man 8 dnsmasq) representation
func (b Binding) String() string {
	return fmt.Sprintf("%s,%s", b.HW.String(), b.IP.String())
}

// Conf represents the configured Bindings
type Conf struct {
	// this is not really for efficiency, even though it's a nice plus,
	// but rather  because MAC (as string) is the key here.
	bindings map[string]Binding
}

func NewConf() *Conf {
	return &Conf{
		bindings: make(map[string]Binding),
	}
}

// Len returns the number of configured Bindings
func (m *Conf) Len() int {
	return len(m.bindings)
}

// String converts all the registered bindings in the Conf in content in dhcphosts (man 8 dnsmasq) representation
func (m *Conf) String() string {
	var sb strings.Builder
	for _, bi := range m.bindings {
		sb.WriteString(fmt.Sprintf("%s\n", bi.String()))
	}
	return sb.String()
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
		return fmt.Errorf("%s: %s", ErrDuplicateFound, x)
	}
	m.bindings[b.HW.String()] = b
	log.Printf("dhcphosts: added [[%s]]", b)
	return nil
}

// Add registers a new Binding
func (m *Conf) Add(mac, ip string) (Binding, error, bool) {
	hwAddr, err := net.ParseMAC(mac)
	if err != nil {
		return Binding{}, err, false
	}
	ret := Binding{
		HW: hwAddr,
		IP: net.ParseIP(ip),
	}
	if ret.IP == nil {
		return ret, ErrBadIPFormat, false
	}
	err = m.add(ret)
	return ret, err, err != nil
}

func (m *Conf) Remove(mac string) (Binding, bool) {
	ret, removed := m.bindings[mac]
	delete(m.bindings, mac)
	log.Printf("dhcphosts: removed [[%s]] -> %v", ret, removed)
	return ret, removed
}

func (m *Conf) GetByHWAddr(hw string) (Binding, error) {
	err := ErrHWAddrNotFound
	var ret Binding

	defer func() {
		log.Printf("dhcphosts: GetByHWAddr(%s) -> (%s, %v)", hw, ret, err)
	}()

	ret, ok := m.bindings[hw]
	if ok {
		err = nil
	}
	return ret, err
}

func (m *Conf) GetByIP(ip string) (Binding, error) {
	err := ErrIPAddrNotFound
	var ret Binding

	defer func() {
		log.Printf("dhcphosts: GetByIP(%s) -> (%s, %v)", ip, ret, err)
	}()

	x := net.ParseIP(ip)
	if x == nil {
		return Binding{}, ErrBadIPFormat
	}
	for _, b := range m.bindings {
		if b.EqualIP(x) {
			ret = b
			err = nil
			break
		}
	}
	return ret, err
}

// Parse creates a Conf from a reader, which must return content in dhcphosts (man 8 dnsmasq) format
func Parse(r io.Reader) (*Conf, error) {
	m := NewConf()
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
