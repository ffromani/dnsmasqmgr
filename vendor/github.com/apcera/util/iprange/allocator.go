// Copyright 2014 Apcera Inc. All rights reserved.

package iprange

import (
	"bytes"
	"math/big"
	"math/rand"
	"net"
	"sync"
)

// a const from Go itself, used to represent IPv4 within an 16 byte slice
var ipv6in4 = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xff, 0xff}

// IPRangeAllocator can be used to allocate IP addresses from the provided
// range.
type IPRangeAllocator struct {
	ipRange     *IPRange
	size        int64
	remaining   int64
	reserved    map[int64]bool
	startBig    *big.Int
	startIsIPv4 bool
	mutex       sync.Mutex
}

// NewAllocator creates a new IPRangeAllocator for the provided IPRange.
func NewAllocator(ipr *IPRange) *IPRangeAllocator {
	a := &IPRangeAllocator{
		ipRange:     ipr,
		reserved:    make(map[int64]bool),
		startIsIPv4: bytes.Compare(ipr.Start.To16()[0:12], ipv6in4) == 0,
	}

	// calculate the size of the range
	a.startBig = big.NewInt(0)
	a.startBig.SetBytes(a.ipRange.Start)
	endBig := big.NewInt(0)
	endBig.SetBytes(a.ipRange.End)
	sizeBig := endBig.Sub(endBig, a.startBig)

	// 1 is added to the size because the end IP is inclusive
	a.size = sizeBig.Int64() + 1
	a.remaining = a.size

	return a
}

// Allocate can be used to allocate a new IP address within the provided
// range. It will ensure that it is unique. If the allocator has no additional
// IP addresses available, then it will return nil.
func (a *IPRangeAllocator) Allocate() net.IP {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	// ensure we have some IPs first
	if a.remaining <= 0 {
		return nil
	}

	// get a random number within the size to start with
	idx := rand.Int63n(a.size)

	// find the next available index after the random number that is available
	idx = a.findNextAvailbleIndex(idx)

	// if idx is now -1, then it couldn't find one, which is very unlikely,
	// however if that is, treat it as no more remaining and ensure remaining is
	// now 0
	if idx == -1 {
		a.remaining = 0
		return nil
	}

	// reserve the idx, get the IP bytes
	a.reserved[idx] = true
	a.remaining--
	newBig := big.NewInt(0).Add(a.startBig, big.NewInt(idx))

	return a.bigIntToIP(newBig)
}

// Reserve allows reserving a specific IP address within the specified range to
// ensure it is not allocated.
func (a *IPRangeAllocator) Reserve(ip net.IP) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	// ensure the specified IP is within the range
	if !a.ipRange.Contains(ip) {
		return
	}

	// calculate the idx from the start
	ipBig := big.NewInt(0)
	ipBig.SetBytes(ip)
	idx := ipBig.Sub(ipBig, a.startBig).Int64()

	// if it isn't already reserved, then mark it reserved and decrement the
	// remaining count
	if !a.reserved[idx] {
		a.reserved[idx] = true
		a.remaining--
	}
}

// Release can be used to release an IP address that had previously been
// allocated or reserved.
func (a *IPRangeAllocator) Release(ip net.IP) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	// calculate the idx from the start
	ipBig := big.NewInt(0)
	ipBig.SetBytes(ip)
	idx := ipBig.Sub(ipBig, a.startBig).Int64()

	// check if the idx is reserved
	if a.reserved[idx] {
		delete(a.reserved, idx)
		a.remaining++
	}
}

// Subtract marks all of the IPs from another IPRange as reserved in the current
// allocator.
func (a *IPRangeAllocator) Subtract(iprange *IPRange) {
	curBig := big.NewInt(0)
	curBig.SetBytes(iprange.Start)
	endBig := big.NewInt(0)
	endBig.SetBytes(iprange.End)

	for ; curBig.Cmp(endBig) < 1; curBig = curBig.Add(big.NewInt(1), curBig) {
		ip := a.bigIntToIP(curBig)
		if a.ipRange.Contains(ip) {
			a.Reserve(ip)
		}
	}
}

// IPRange returns a copy of the IPRange provided to the allocator.
func (a *IPRangeAllocator) IPRange() *IPRange {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	return &IPRange{
		Start: a.ipRange.Start,
		End:   a.ipRange.End,
		Mask:  a.ipRange.Mask,
	}
}

// Size returns the size of the allowable IP addresses specified by the range.
func (a *IPRangeAllocator) Size() int64 {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	return a.size
}

// Remaining returns the number of remaining IP addresses within the provided
// range that have not been already allocated.
func (a *IPRangeAllocator) Remaining() int64 {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	return a.remaining
}

// findNextAvailableIndex finds the value of idx which is available and not in
// the reserved list.
func (a *IPRangeAllocator) findNextAvailbleIndex(idx int64) int64 {
	// walk up from the index
	for i := idx; i < a.size; i++ {
		if !a.reserved[i] {
			return i
		}
	}

	// nothing above that one... lets try to walk down to find something
	for i := idx - 1; i >= 0; i-- {
		if !a.reserved[i] {
			return i
		}
	}

	// ok, everything is probably taken
	return -1
}

func (a *IPRangeAllocator) bigIntToIP(newBig *big.Int) net.IP {
	// Convert it back into a 16 byte slice. net.IP expects a 16 byte
	// slice, and expects the elements to be not be the leading bytes
	// but the trailing. So we must create a new slice and populate its
	// tail.
	buf := newBig.Bytes()
	ipbytes := make([]byte, 16)
	position := 16 - len(buf)

	// If the position we need to copy to is less than 0, then this
	// would cause an index out of range. This will only happen when
	// we've max'd out 16 bytes, so then we'll just loop around to zero.
	if position >= 0 {
		if a.startIsIPv4 {
			// copy only the last 4 bytes and ensure we set the IPv4 in v6 prefix
			copy(ipbytes, ipv6in4)
			copy(ipbytes[12:], buf[len(buf)-4:])
		} else {
			// copy into the 16 byte slice, as it is IPv6
			copy(ipbytes[16-len(buf):], buf)
		}
	}

	return net.IP(ipbytes)
}
