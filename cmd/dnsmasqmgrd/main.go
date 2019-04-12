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
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"

	flag "github.com/spf13/pflag"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	pb "github.com/mojaves/dnsmasqmgr/pkg/dnsmasqmgr"
	"github.com/mojaves/dnsmasqmgr/pkg/server"
)

const (
	DefaultIface string = "127.0.0.1"
	DefaultPort  int    = 50777
)

var (
	readOnly = flag.Bool("readonly", false, "DBs readonly mode")
	iface    = flag.String("interface", DefaultIface, "The server listening interface")
	port     = flag.Int("port", DefaultPort, "The server port")
	makeConf = flag.Bool("makeconf", false, "Create template configuration and exit")
)

type Config struct {
	IPRange    string `json:"iprange"`
	HostsPath  string `json:"hostspath"`
	LeasesPath string `json:"leasespath"`
	CertFile   string `json:"certfile"`
	KeyFile    string `json:"keyfile"`
	Iface      string `json:"iface"`
	Port       int    `json:"port"`
}

func DefaultConfig() *Config {
	return &Config{
		Iface: DefaultIface,
		Port:  DefaultPort,
	}
}

func ParseConfig(path string) (*Config, error) {
	fh, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fh.Close()
	cfg := Config{}
	dec := json.NewDecoder(fh)
	dec.Decode(&cfg)
	return &cfg, nil
}

func (cfg *Config) Check() error {
	if cfg.IPRange == "" {
		return fmt.Errorf("ip range must be specified")
	}
	if cfg.HostsPath == "" || cfg.LeasesPath == "" {
		return fmt.Errorf("missing configuration files: hosts=[%v] leases=[%v]", cfg.HostsPath, cfg.LeasesPath)
	}
	return nil
}

func setupTLS(cfg *Config) ([]grpc.ServerOption, error) {
	var err error
	var opts []grpc.ServerOption
	if cfg.CertFile != "" && cfg.KeyFile != "" {
		creds, err := credentials.NewServerTLSFromFile(cfg.CertFile, cfg.KeyFile)
		if err == nil {
			opts = []grpc.ServerOption{grpc.Creds(creds)}
		}
	}
	return opts, err
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [options] [config.json]\n", filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
	flag.Parse()

	var err error
	var conf *Config

	if *makeConf {
		conf = DefaultConfig()
		enc := json.NewEncoder(os.Stdout)
		enc.Encode(conf)
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) >= 1 {
		conf, err = ParseConfig(args[0])
		if err != nil {
			log.Fatalf("error parsing the configuration %s: %v", args[0], err)
		}
	} else {
		conf = DefaultConfig()
	}
	err = conf.Check()
	if err != nil {
		log.Fatalf("configuration error: %v", err)
	}
	log.Printf("dnsmasqmgrd: using configuration files: hosts=[%v] leases=[%v]", conf.HostsPath, conf.LeasesPath)

	opts, err := setupTLS(conf)
	if err != nil {
		log.Fatalf("Failed to generate credentials %v", err)
	}

	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", conf.Iface, conf.Port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	var mgr *server.DNSMasqMgr
	if *readOnly {
		mgr, err = server.NewDNSMasqMgrReadOnly(conf.IPRange, conf.HostsPath, conf.LeasesPath)
	} else {
		mgr, err = server.NewDNSMasqMgr(conf.IPRange, conf.HostsPath, conf.LeasesPath)
	}
	if err != nil {
		log.Fatalf("%v", err)
	}

	log.Printf("dnsmasqmgrd: ready ===")

	serv := grpc.NewServer(opts...)
	pb.RegisterDNSMasqManagerServer(serv, mgr)
	serv.Serve(lis)
}
