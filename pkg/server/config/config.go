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
package config

import (
	"encoding/json"
	"fmt"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	DefaultIface string = "127.0.0.1"
	DefaultPort  int    = 50777
)

type Config struct {
	IPRange     string `json:"iprange"`
	HostsPath   string `json:"hostspath"`
	LeasesPath  string `json:"leasespath"`
	CertFile    string `json:"certfile"`
	KeyFile     string `json:"keyfile"`
	Iface       string `json:"iface"`
	Port        int    `json:"port"`
	JournalPath string `json:"journalpath"`
}

func Default() *Config {
	return &Config{
		Iface: DefaultIface,
		Port:  DefaultPort,
	}
}

func ParseFile(path string) (*Config, error) {
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

func (cfg *Config) SetupTLS() ([]grpc.ServerOption, error) {
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
