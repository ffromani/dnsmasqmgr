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
	"fmt"
	"os"
	"time"

	flag "github.com/spf13/pflag"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/mojaves/dnsmasqmgr/pkg/client"
	pb "github.com/mojaves/dnsmasqmgr/pkg/dnsmasqmgr"
)

var (
	certFile = flag.String("certfile", "", "The TLS cert file")
	keyFile  = flag.String("keyfile", "", "The TLS key file")
	timeout  = flag.Int("timeout", 1, "The connection timeout (seconds)")
	iface    = flag.String("interface", "127.0.0.1", "The server listening interface")
	port     = flag.Int("port", 50777, "The server port")
)

func main() {
	flag.Usage = client.Usage
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	query, err := client.NewQuery(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
	}

	var opts []grpc.DialOption
	if *certFile != "" && *keyFile != "" {
		creds, err := credentials.NewServerTLSFromFile(*certFile, *keyFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to generate credentials %v\n", err)
			os.Exit(8)
		}
		opts = []grpc.DialOption{grpc.WithTransportCredentials(creds)}
	} else {
		opts = []grpc.DialOption{grpc.WithInsecure()}
	}

	address := fmt.Sprintf("%s:%d", *iface, *port)
	conn, err := grpc.Dial(address, opts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not connect: %v\n", err)
		os.Exit(16)
	}
	defer conn.Close()
	c := pb.NewDNSMasqManagerClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*timeout)*time.Second)
	defer cancel()

	out, _, err := query.RunWith(ctx, c)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error performing: %s: %v\n", query, err)
	}
	fmt.Printf("%s\n", out)
}
