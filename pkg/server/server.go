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

// the package dhcphosts provides utilities to work with configuration in the dhcphosts
// (see man 8 dnsmasq) format
package server

import (
	"errors"
	"log"
	"os"

	dhcpmap "github.com/mojaves/dnsmasqmgr/pkg/dhcphosts"
	resolv "github.com/mojaves/dnsmasqmgr/pkg/etchosts"
)

var (
	ErrNotSupported error = errors.New("Operation not supported")
	ErrRequestData  error = errors.New("Malformed request")
	ErrInvalidParam error = errors.New("Invalid parameter in request")
	ErrMissingKey   error = errors.New("Missing key for research")
)

type DNSMasqMgr struct {
	readOnly   bool
	hostsPath  string
	leasesPath string
	addrMap    *dhcpmap.Conf
	nameMap    *resolv.Conf
}

func NewDNSMasqMgrReadOnly(hostsPath, leasesPath string) (*DNSMasqMgr, error) {
	dmm, err := NewDNSMasqMgr(hostsPath, leasesPath)
	if dmm != nil {
		dmm.readOnly = true
	}
	log.Printf("server: started in ReadOnly mode")
	return dmm, err
}

func NewDNSMasqMgr(hostsPath, leasesPath string) (*DNSMasqMgr, error) {
	var err error
	dmm := DNSMasqMgr{
		hostsPath:  hostsPath,
		leasesPath: leasesPath,
	}
	hostsFile, err := os.Open(hostsPath)
	if err != nil {
		return nil, err
	}
	defer hostsFile.Close()
	leasesFile, err := os.Open(leasesPath)
	if err != nil {
		return nil, err
	}
	defer leasesFile.Close()

	dmm.nameMap, err = resolv.Parse(hostsFile)
	if err != nil {
		return nil, err
	}
	log.Printf("server: parsed %d entries from '%v'", dmm.nameMap.Len(), hostsPath)

	dmm.addrMap, err = dhcpmap.Parse(leasesFile)
	if err != nil {
		return nil, err
	}
	log.Printf("server: parsed %d entries from '%v'", dmm.addrMap.Len(), leasesPath)

	log.Printf("server: set up DNSMasqMgr")
	return &dmm, nil
}

func (dmm *DNSMasqMgr) Store() error {
	return ErrNotSupported
}
