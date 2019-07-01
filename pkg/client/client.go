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
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	flag "github.com/spf13/pflag"

	pb "github.com/mojaves/dnsmasqmgr/pkg/dnsmasqmgr"
)

type Address struct {
	Name string `json:"name"`
	Mac  string `json:"mac"`
	IP   string `json:"ip"`
}

func addrToJson(a *pb.Address) string {
	b, err := json.Marshal(Address{
		Name: a.Hostname,
		Mac:  a.Macaddr,
		IP:   a.Ipaddr,
	})
	if err != nil {
		return ""
	}
	return string(b)
}

type Queryable interface {
	String() string
	SetupArgs(args []string) error
	RunWith(ctx context.Context, c pb.DNSMasqManagerClient) (string, string, error)
}

type QueryLookup struct {
	Name string
	req  *pb.AddressRequest
}

func (ql *QueryLookup) String() string {
	return fmt.Sprintf("%s(%s)", ql.Name, ql.req.Addr)
}

func AddressRequestFromArgs(args []string) (*pb.AddressRequest, error) {
	// args:
	// [0]       [1]  [2]
	// <action>  how  what
	if len(args) < 3 {
		return nil, fmt.Errorf("not enough arguments: `%v`", args[1:])
	}
	req := pb.AddressRequest{}
	switch args[1] {
	case "name":
		req.Key = pb.Key_HOSTNAME
		req.Addr = &pb.Address{
			Hostname: args[2],
		}
	case "mac":
		req.Key = pb.Key_MACADDR
		req.Addr = &pb.Address{
			Macaddr: args[2],
		}
	case "ip":
		req.Key = pb.Key_IPADDR
		req.Addr = &pb.Address{
			Ipaddr: args[2],
		}
	default:
		return nil, fmt.Errorf("%s: unsupported method: %s", args[0], args[1])
	}
	return &req, nil
}

func (ql *QueryLookup) SetupArgs(args []string) error {
	var err error
	ql.req, err = AddressRequestFromArgs(args)
	return err
}

func (ql *QueryLookup) RunWith(ctx context.Context, c pb.DNSMasqManagerClient) (string, string, error) {
	r, err := c.LookupAddress(ctx, ql.req)
	return addrToJson(r.Addr), "", err
}

type QueryRequest struct {
	Name string
	addr *pb.Address
}

func (qr *QueryRequest) String() string {
	reqip := ""
	if qr.addr.Ipaddr != "" {
		reqip = fmt.Sprintf(", ip=%s", qr.addr.Ipaddr)
	}
	return fmt.Sprintf("%s(name=%s, mac=%s%s)", qr.Name, qr.addr.Hostname, qr.addr.Macaddr, reqip)
}

func (qr *QueryRequest) SetupArgs(args []string) error {
	// args:
	// [0]     [1]  [2]  [[3]]
	// request host mac  [ip]
	if len(args) < 3 {
		return fmt.Errorf("not enough arguments: `%v`", args[1:])
	}
	qr.addr = &pb.Address{
		Hostname: args[1],
		Macaddr:  args[2],
	}
	if len(args) >= 4 {
		qr.addr.Ipaddr = args[3]
	}
	return nil
}

func (qr *QueryRequest) RunWith(ctx context.Context, c pb.DNSMasqManagerClient) (string, string, error) {
	r, err := c.RequestAddress(ctx, &pb.AddressRequest{
		Addr: qr.addr,
	})
	// TODO: r.Code?
	if err != nil {
		return "", "", err
	}
	return addrToJson(r.Addr), "", nil
}

type QueryDelete struct {
	Name string
	req  *pb.AddressRequest
}

func (ql *QueryDelete) String() string {
	return fmt.Sprintf("%s(%s)", ql.Name, ql.req.Addr)
}

func (ql *QueryDelete) SetupArgs(args []string) error {
	var err error
	ql.req, err = AddressRequestFromArgs(args)
	return err
}

func (ql *QueryDelete) RunWith(ctx context.Context, c pb.DNSMasqManagerClient) (string, string, error) {
	r, err := c.DeleteAddress(ctx, ql.req)
	return addrToJson(r.Addr), "", err
}

func Usage() {
	fmt.Fprintf(os.Stderr, "Usage %s [options] subcommand args:\n", filepath.Base(os.Args[0]))
	fmt.Fprintf(os.Stderr, "subcommands:\n")
	fmt.Fprintf(os.Stderr, "- request <hostname> <macaddr> [ipaddr]\n")
	fmt.Fprintf(os.Stderr, "- delete <how> <what>\n")
	fmt.Fprintf(os.Stderr, "- lookup <how> <what>\n")
	fmt.Fprintf(os.Stderr, "  * how:  one of 'name', 'mac', 'ip'\n")
	fmt.Fprintf(os.Stderr, "options:\n")
	flag.PrintDefaults()
}

func ParseArgs() (*Config, []string) {
	conf := Config{}
	flag.StringVar(&conf.CertFile, "certfile", "", "The TLS cert file")
	flag.StringVar(&conf.KeyFile, "keyfile", "", "The TLS key file")
	flag.IntVar(&conf.Timeout, "timeout", 1, "The connection timeout (seconds)")
	flag.StringVar(&conf.Iface, "interface", "127.0.0.1", "The server listening interface")
	flag.IntVar(&conf.Port, "port", 50777, "The server port")

	flag.Usage = Usage
	flag.Parse()
	args := flag.Args()

	return &conf, args
}

func NewQuery(args []string) (Queryable, error) {
	var query Queryable
	switch args[0] {
	case "lookup":
		query = &QueryLookup{Name: args[0]}
	case "request":
		query = &QueryRequest{Name: args[0]}
	case "delete":
		query = &QueryDelete{Name: args[0]}
	default:
		return nil, fmt.Errorf("Unsupported subcommand %s\n", args[0])
	}

	err := query.SetupArgs(args)
	if err != nil {
		return nil, err
	}
	return query, nil
}

type Config struct {
	CertFile string
	KeyFile  string
	Timeout  int
	Iface    string
	Port     int
}

func RunQuery(conf *Config, query Queryable) (string, string, error) {
	var opts []grpc.DialOption
	if conf.CertFile != "" && conf.KeyFile != "" {
		creds, err := credentials.NewServerTLSFromFile(conf.CertFile, conf.KeyFile)
		if err != nil {
			return "", "", fmt.Errorf("Failed to generate credentials %v", err)
		}
		opts = []grpc.DialOption{grpc.WithTransportCredentials(creds)}
	} else {
		opts = []grpc.DialOption{grpc.WithInsecure()}
	}

	address := fmt.Sprintf("%s:%d", conf.Iface, conf.Port)
	conn, err := grpc.Dial(address, opts...)
	if err != nil {
		return "", "", fmt.Errorf("Could not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewDNSMasqManagerClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(conf.Timeout)*time.Second)
	defer cancel()

	return query.RunWith(ctx, c)
}
