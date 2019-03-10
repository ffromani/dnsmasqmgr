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

// Package main implements a client for DNSMAsqMgr service.
package main

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"

	pb "github.com/mojaves/dnsmasqmgr/pkg/dnsmasqmgr"
)

const (
	address = "localhost:50777"
)

func main() {
	// Set up a connection to the server.
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewDNSMasqManagerClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	r, err := c.LookupAddress(ctx, &pb.AddressRequest{
		Addr: &pb.Address{
			Hostname: "test",
		},
		Key: pb.Key_HOSTNAME,
	})
	if err != nil {
		log.Fatalf("could not lookup: %v", err)
	}
	log.Printf("%s: mac=%s ip=%s", r.Addr.Hostname, r.Addr.Macaddr, r.Addr.Ipaddr)
}
