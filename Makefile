proto: pkg/dnsmasqmgr/dnsmasqmgr.pb.go

pkg/dnsmasqmgr/dnsmasqmgr.pb.go: pkg/dnsmasqmgr/dnsmasqmgr.proto
	cd pkg && protoc -I dnsmasqmgr/ dnsmasqmgr/dnsmasqmgr.proto --go_out=plugins=grpc:dnsmasqmgr
