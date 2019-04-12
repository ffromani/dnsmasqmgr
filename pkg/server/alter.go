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
	"context"
	"encoding/json"
	"log"
	"net"

	pb "github.com/mojaves/dnsmasqmgr/pkg/dnsmasqmgr"
)

type JournalAddr struct {
	Hostname string `json:"hostname"`
	Macaddr  string `json:"mac"`
	Ipaddr   string `json:"ip"`
}

type JournalEntry struct {
	Action  string      `json:"action"`
	Address JournalAddr `json:"address"`
}

func (ja *JournalAddr) FromAddressRequest(req *pb.AddressRequest) {
	ja.Hostname = req.Addr.Hostname
	ja.Macaddr = req.Addr.Macaddr
	ja.Ipaddr = req.Addr.Ipaddr
}

func FromAddressRequest(action string, req *pb.AddressRequest) *JournalEntry {
	je := JournalEntry{
		Action: action,
	}
	je.Address.FromAddressRequest(req)
	return &je
}

func handleDuplicate(ar *pb.AddressReply, key pb.Key, val string) {
	switch ar.Match {
	case pb.Match_NONE:
		ar.Match = pb.Match_PARTIAL
		ar.Key = key
	case pb.Match_PARTIAL:
		ar.Match = pb.Match_FULL
	default:
		// we are fine as we are
	}
	log.Printf("%s %s already present, skipped", pb.Key_name[int32(key)], val)
}

func (dmm *DNSMasqMgr) RequestAddress(ctx context.Context, req *pb.AddressRequest) (*pb.AddressReply, error) {
	if req == nil || req.Addr == nil || req.Addr.Hostname == "" || req.Addr.Macaddr == "" {
		return nil, ErrRequestData
	}

	dmm.lock.Lock()
	defer dmm.lock.Unlock()

	var err error
	var ipAddr net.IP
	if req.Addr.Ipaddr == "" {
		ipAddr = dmm.ipAlloc.Allocate()
		req.Addr.Ipaddr = ipAddr.String()
	} else {
		ipAddr = net.ParseIP(req.Addr.Ipaddr)
		if ipAddr == nil {
			return nil, err
		}
		dmm.ipAlloc.Reserve(ipAddr)
	}

	ret := pb.AddressReply{
		Match: pb.Match_NONE,
	}
	var present bool
	var aliases []string
	_, err, present = dmm.nameMap.Add(req.Addr.Hostname, req.Addr.Ipaddr, aliases)
	if err != nil {
		return nil, err
	}
	if present {
		handleDuplicate(&ret, pb.Key_HOSTNAME, req.Addr.Hostname)
	}

	_, err, present = dmm.addrMap.Add(req.Addr.Macaddr, req.Addr.Ipaddr)
	if err != nil {
		dmm.nameMap.Remove(req.Addr.Hostname)
		return nil, err
	}
	if present {
		handleDuplicate(&ret, pb.Key_MACADDR, req.Addr.Macaddr)
	}

	entry, err := json.Marshal(FromAddressRequest("add", req))
	if err != nil {
		log.Printf("cannot add to journal: %v", err)
		// intentionally do NOT abort
	} else {
		dmm.changes.Printf("%s", string(entry))
	}
	defer dmm.requestStore()

	ret.Addr = req.Addr
	return &ret, nil
}

func (dmm *DNSMasqMgr) DeleteAddress(ctx context.Context, req *pb.AddressRequest) (*pb.AddressReply, error) {
	dmm.lock.Lock()
	defer dmm.lock.Unlock()
	return nil, ErrNotSupported
}
