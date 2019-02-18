package clientport

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"strings"

	"github.com/coredhcp/coredhcp/handler"
	"github.com/coredhcp/coredhcp/logger"
	"github.com/coredhcp/coredhcp/plugins"
	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/insomniacslk/dhcp/dhcpv6"
)

var log = logger.GetLogger()

func init() {
	plugins.RegisterPlugin("file", setupFile6, setupFile4)
}

// StaticRecords holds a MAC -> IP address mapping
var StaticRecords map[string]net.IP

// DHCPv6Records and DHCPv4Records are mappings between MAC addresses in
// form of a string, to network configurations.
var (
	DHCPv6Records map[string]net.IP
	DHCPv4Records map[string]net.IP
)
var serverID *dhcpv6.OptServerId

// LoadDHCPv6Records loads the DHCPv6Records global map with records stored on
// the specified file. The records have to be one per line, a mac address and an
// IPv6 address.
func LoadDHCPv6Records(filename string) (map[string]net.IP, error) {
	log.Printf("plugins/file: reading leases from %s", filename)
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	records := make(map[string]net.IP, 0)
	// TODO ignore comments
	for _, lineBytes := range bytes.Split(data, []byte{'\n'}) {
		line := string(lineBytes)
		if len(line) == 0 {
			continue
		}
		tokens := strings.Fields(line)
		if len(tokens) != 2 {
			return nil, fmt.Errorf("plugins/file: malformed line: %s", line)
		}
		hwaddr, err := net.ParseMAC(tokens[0])
		if err != nil {
			return nil, fmt.Errorf("plugins/file: malformed hardware address: %s", tokens[0])
		}
		ipaddr := net.ParseIP(tokens[1])
		if ipaddr.To16() == nil {
			return nil, fmt.Errorf("plugins/file: expected an IPv6 address, got: %v", ipaddr)
		}
		records[hwaddr.String()] = ipaddr
	}
	return records, nil
}

// LoadDHCPv4Records loads the DHCPv6Records global map with records stored on
// the specified file. The records have to be one per line, a mac address and an
// IPv6 address.
func LoadDHCPv4Records(filename string) (map[string]net.IP, error) {
	log.Printf("plugins/file: reading leases from %s", filename)
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	records := make(map[string]net.IP, 0)
	// TODO ignore comments
	for _, lineBytes := range bytes.Split(data, []byte{'\n'}) {
		line := string(lineBytes)
		if len(line) == 0 {
			continue
		}
		tokens := strings.Fields(line)
		if len(tokens) != 2 {
			return nil, fmt.Errorf("plugins/file: malformed line: %s", line)
		}
		hwaddr, err := net.ParseMAC(tokens[0])
		if err != nil {
			return nil, fmt.Errorf("plugins/file: malformed hardware address: %s", tokens[0])
		}
		ipaddr := net.ParseIP(tokens[1])
		if ipaddr.To4() == nil {
			return nil, fmt.Errorf("plugins/file: expected an IPv4 address, got: %v", ipaddr)
		}
		records[hwaddr.String()] = ipaddr
	}
	return records, nil
}

// Handler6 handles DHCPv6 packets for the file plugin
func Handler6(req, resp dhcpv6.DHCPv6) (dhcpv6.DHCPv6, bool) {
	mac, err := dhcpv6.ExtractMAC(req)
	if err != nil {
		return nil, false
	}

	ipaddr, ok := StaticRecords[mac.String()]
	if !ok {
		return nil, false
	}
	log.Printf("Found IP address %s for MAC %s", ipaddr, mac)
	// TODO add an OptIANA based on the above data
	return resp, true
}

// Handler4 handles DHCPv4 packets for the file plugin
func Handler4(req, resp *dhcpv4.DHCPv4) (*dhcpv4.DHCPv4, bool) {
	mac := req.ClientHWAddr

	ipaddr, ok := StaticRecords[mac.String()]
	if !ok {
		return nil, false
	}
	log.Printf("Found IP address %s for MAC %s", ipaddr, mac)

	resp.ClientIPAddr = ipaddr
	resp.ServerIPAddr = ipaddr
	resp.YourIPAddr = ipaddr
	resp.UpdateOption(dhcpv4.OptMessageType(dhcpv4.MessageTypeOffer))

	log.Printf("Response %s", resp.Summary())
	// TODO add an OptIANA based on the above data
	return resp, true
}

func setupFile6(args ...string) (handler.Handler6, error) {
	h6, err := setupFile(true, args...)
	return h6, err
}

func setupFile4(args ...string) (handler.Handler4, error) {
	log.Print("plugins/file: loading `file` plugin for DHCPv4")
	h4, err := setupFilev4(true, args...)
	return h4, err
}

func setupFile(v6 bool, args ...string) (handler.Handler6, error) {
	if len(args) < 1 {
		return nil, errors.New("plugins/file: need a file name")
	}
	filename := args[0]
	if filename == "" {
		return nil, errors.New("plugins/file: got empty file name")
	}
	records, err := LoadDHCPv6Records(filename)
	if err != nil {
		return nil, fmt.Errorf("plugins/file: failed to load DHCPv6 records: %v", err)
	}
	log.Printf("plugins/file: loaded %d leases from %s", len(records), filename)
	StaticRecords = records

	return Handler6, nil
}


func setupFilev4(v4 bool, args ...string) (handler.Handler4, error) {
	if len(args) < 1 {
		return nil, errors.New("plugins/file: need a file name")
	}
	filename := args[0]
	if filename == "" {
		return nil, errors.New("plugins/file: got empty file name")
	}
	records, err := LoadDHCPv4Records(filename)
	if err != nil {
		return nil, fmt.Errorf("plugins/file: failed to load DHCPv4 records: %v", err)
	}
	log.Printf("plugins/file: loaded %d leases from %s", len(records), filename)
	StaticRecords = records

	return Handler4, nil
}
