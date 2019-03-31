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

	pb "github.com/mojaves/dnsmasqmgr/pkg/dnsmasqmgr"
)

func (dmm *DNSMasqMgr) LookupAddress(ctx context.Context, req *pb.AddressRequest) (*pb.AddressReply, error) {
	if req == nil || req.Addr == nil {
		return nil, ErrRequestData
	}
	switch req.Key {
	case pb.Key_HOSTNAME:
		return dmm.lookupAddressByHostname(ctx, req.Addr.Hostname)
	case pb.Key_MACADDR:
		return dmm.lookupAddressByMacaddr(ctx, req.Addr.Macaddr)
	case pb.Key_IPADDR:
		return dmm.lookupAddressByIpaddr(ctx, req.Addr.Ipaddr)
	}
	return nil, ErrInvalidParam
}

func (dmm *DNSMasqMgr) lookupAddressByHostname(ctx context.Context, hostname string) (*pb.AddressReply, error) {
	reply := pb.AddressReply{
		Match: pb.Match_NONE,
	}
	if hostname == "" {
		return &reply, ErrMissingKey
	}

	host, err := dmm.nameMap.GetByHostname(hostname)
	if err != nil {
		return &reply, err
	}
	reply = pb.AddressReply{
		Addr: &pb.Address{
			Hostname: host.CanonicalHostname,
			Ipaddr:   host.Address.String(),
		},
		Match: pb.Match_PARTIAL,
	}
	binding, err := dmm.addrMap.GetByIP(reply.Addr.Ipaddr)
	if err != nil {
		return &reply, nil
	}
	reply.Addr.Macaddr = binding.HW.String()
	reply.Match = pb.Match_FULL
	return &reply, nil
}

func (dmm *DNSMasqMgr) lookupAddressByMacaddr(ctx context.Context, macaddr string) (*pb.AddressReply, error) {
	reply := pb.AddressReply{
		Match: pb.Match_NONE,
	}
	if macaddr == "" {
		return &reply, ErrMissingKey
	}

	binding, err := dmm.addrMap.GetByHWAddr(macaddr)
	if err != nil {
		return &reply, err
	}
	reply = pb.AddressReply{
		Addr: &pb.Address{
			Macaddr: binding.HW.String(),
			Ipaddr:  binding.IP.String(),
		},
		Match: pb.Match_PARTIAL,
	}

	host, err := dmm.nameMap.GetByAddress(reply.Addr.Ipaddr)
	if err != nil {
		return &reply, nil
	}
	reply.Addr.Hostname = host.CanonicalHostname
	reply.Match = pb.Match_FULL
	return &reply, nil
}

func (dmm *DNSMasqMgr) lookupAddressByIpaddr(ctx context.Context, ipaddr string) (*pb.AddressReply, error) {
	reply := pb.AddressReply{
		Match: pb.Match_NONE,
	}
	if ipaddr == "" {
		return &reply, ErrMissingKey
	}

	host, err := dmm.nameMap.GetByAddress(ipaddr)
	if err != nil {
		return &reply, err
	}
	reply = pb.AddressReply{
		Addr: &pb.Address{
			Hostname: host.CanonicalHostname,
			Ipaddr:   host.Address.String(),
		},
		Match: pb.Match_PARTIAL,
	}
	binding, err := dmm.addrMap.GetByIP(reply.Addr.Ipaddr)
	if err != nil {
		return &reply, nil
	}
	reply.Addr.Macaddr = binding.HW.String()
	reply.Match = pb.Match_FULL
	return &reply, nil
}
