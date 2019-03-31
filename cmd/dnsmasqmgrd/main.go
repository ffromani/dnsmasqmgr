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
	"fmt"
	"log"
	"net"

	flag "github.com/spf13/pflag"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	pb "github.com/mojaves/dnsmasqmgr/pkg/dnsmasqmgr"
	"github.com/mojaves/dnsmasqmgr/pkg/server"
)

var (
	readOnly   = flag.Bool("readonly", false, "DBs readonly mode")
	hostsPath  = flag.String("hostspath", "", "The hosts db file")
	leasesPath = flag.String("leasespath", "", "The dnsmasq leases file")
	certFile   = flag.String("certfile", "", "The TLS cert file")
	keyFile    = flag.String("keyfile", "", "The TLS key file")
	iface      = flag.String("interface", "127.0.0.1", "The server listening interface")
	port       = flag.Int("port", 50777, "The server port")
)

func main() {
	flag.Parse()

	if *hostsPath == "" || *leasesPath == "" {
		log.Fatalf("missing configuration files: hosts=[%v] leases=[%v]", *hostsPath, *leasesPath)
	}
	log.Printf("dnsmasqmgrd: using configuration files: hosts=[%v] leases=[%v]", *hostsPath, *leasesPath)

	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", *iface, *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	var opts []grpc.ServerOption
	if *certFile != "" && *keyFile != "" {
		creds, err := credentials.NewServerTLSFromFile(*certFile, *keyFile)
		if err != nil {
			log.Fatalf("Failed to generate credentials %v", err)
		}
		opts = []grpc.ServerOption{grpc.Creds(creds)}
	}

	var mgr *server.DNSMasqMgr
	if *readOnly {
		mgr, err = server.NewDNSMasqMgrReadOnly(*hostsPath, *leasesPath)
	} else {
		mgr, err = server.NewDNSMasqMgr(*hostsPath, *leasesPath)
	}
	if err != nil {
		log.Fatalf("%v", err)
	}

	log.Printf("dnsmasqmgrd: ready ===")

	serv := grpc.NewServer(opts...)
	pb.RegisterDNSMasqManagerServer(serv, mgr)
	serv.Serve(lis)
}
