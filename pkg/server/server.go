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
	"io/ioutil"
	"log"
	"os"
	"sync"

	"github.com/apcera/util/iprange"

	"github.com/mojaves/dnsmasqmgr/pkg/dhcphosts"
	"github.com/mojaves/dnsmasqmgr/pkg/etchosts"
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
	hostsInfo  os.FileInfo
	leasesPath string
	leasesInfo os.FileInfo
	flushChan  chan bool
	doneChan   chan bool
	lock       sync.RWMutex
	nameMap    *etchosts.Conf
	addrMap    *dhcphosts.Conf
	ipAlloc    *iprange.IPRangeAllocator
	journal    *os.File
	changes    *log.Logger
}

func NewDNSMasqMgrReadOnly(iprangeStr, hostsPath, leasesPath string) (*DNSMasqMgr, error) {
	dmm, err := NewDNSMasqMgr(iprangeStr, hostsPath, leasesPath, "")
	if dmm != nil {
		dmm.readOnly = true
	}
	log.Printf("server: started in ReadOnly mode")
	return dmm, err
}

func NewDNSMasqMgr(iprangeStr, hostsPath, leasesPath, journalPath string) (*DNSMasqMgr, error) {
	var err error
	ips, err := iprange.ParseIPRange(iprangeStr)
	if err != nil {
		return nil, err
	}

	dmm := DNSMasqMgr{
		ipAlloc:    iprange.NewAllocator(ips),
		hostsPath:  hostsPath,
		leasesPath: leasesPath,
		flushChan:  make(chan bool),
		doneChan:   make(chan bool),
	}
	dmm.hostsInfo, err = os.Lstat(hostsPath)
	if err != nil {
		return nil, err
	}
	dmm.leasesInfo, err = os.Lstat(leasesPath)
	if err != nil {
		return nil, err
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

	dmm.nameMap, err = etchosts.Parse(hostsFile)
	if err != nil {
		return nil, err
	}
	log.Printf("server: parsed %d entries from '%v'", dmm.nameMap.Len(), hostsPath)

	dmm.addrMap, err = dhcphosts.Parse(leasesFile)
	if err != nil {
		return nil, err
	}
	log.Printf("server: parsed %d entries from '%v'", dmm.addrMap.Len(), leasesPath)

	if journalPath != "" {
		dmm.journal, err = os.Create(journalPath)
		if err != nil {
			return nil, err
		}
		dmm.changes = log.New(dmm.journal, "", log.LstdFlags)
		log.Printf("server: logging changes on %v", journalPath)
	} else {
		dmm.changes = log.New(ioutil.Discard, "", log.LstdFlags)
		log.Printf("server: NOT logging changes")
	}

	go dmm.storeLoop()
	log.Printf("server: started storing loop")

	log.Printf("server: set up DNSMasqMgr")
	return &dmm, nil
}

func (dmm *DNSMasqMgr) Close() error {
	dmm.flushChan <- false
	<-dmm.doneChan

	if dmm.journal == nil {
		return nil
	}
	return dmm.journal.Close()
}
