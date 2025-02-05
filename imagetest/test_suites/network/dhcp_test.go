//go:build cit
// +build cit

package network

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/guest-test-infra/imagetest/utils"
)

const (
	networkctlCmd = "networkctl"
	nmcliCmd      = "nmcli"
	wickedCmd     = "wicked"
)

// TestDHCP test secondary interfaces are configured with a single dhclient process.
func TestDHCP(t *testing.T) {
	var cmd *exec.Cmd
	var err error

	// Run every case: if one command or check succeeds, the test passes.
	if utils.CheckLinuxCmdExists(networkctlCmd) {
		cmd = exec.Command(networkctlCmd, "status")
		if err = parseNetworkctlOutput(cmd); err == nil {
			return
		}
	}

	if utils.CheckLinuxCmdExists(nmcliCmd) {
		cmd = exec.Command(nmcliCmd, "device", "show")
		if err = parseNmcliOutput(cmd); err == nil {
			return
		}
	}

	if utils.CheckLinuxCmdExists(wickedCmd) {
		cmd = exec.Command(wickedCmd, "show", "all")
		if err = parseWickedOutput(cmd); err == nil {
			return
		}
	}

	// Base dhcp case for debian 10, debian 11, ubuntu 16, etc.
	if err = checkDHCPProcess(); err != nil {
		t.Fatalf("did not find dhcp process: %v", err)
	}

	if err != nil {
		t.Fatalf("dhcp command failed or not found: %v", err)
	}
}

func parseNetworkctlOutput(cmd *exec.Cmd) error {
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("networkctl command failed %v", err)
	}

	// check for any line with dhcpv4. If the line is found, check that
	// a valid IP address is found in the same line.
	for _, line := range strings.Split(string(out), "\n") {
		upperLine := strings.ToUpper(line)
		if strings.Contains(upperLine, "DHCPV4") {
			for _, token := range strings.Fields(upperLine) {
				if validIPOrCIDR(token) {
					return nil
				}
			}
		}
	}

	return fmt.Errorf("valid ip4 address not found in networkctl output")
}

func parseNmcliOutput(cmd *exec.Cmd) error {
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("nmcli command failed %v", err)
	}

	// check for any line with dhcpv4. If the line is found, check that
	// a valid IP address is found in the same line.
	for _, line := range strings.Split(string(out), "\n") {
		upperLine := strings.ToUpper(line)
		if strings.Contains(upperLine, "IP4.ADDRESS") {
			for _, token := range strings.Fields(upperLine) {
				if validIPOrCIDR(token) {
					return nil
				}
			}
		}
	}

	return fmt.Errorf("IPV4Address not found in nmcli output")
}

func parseWickedOutput(cmd *exec.Cmd) error {
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("wicked command failed %v", err)
	}

	for _, line := range strings.Split(string(out), "\n") {
		upperLine := strings.ToUpper(line)
		if strings.Contains(upperLine, "IPV4") && strings.Contains(upperLine, "DHCP") {
			for _, token := range strings.Fields(upperLine) {
				if validIPOrCIDR(token) {
					return nil
				}
			}
		}
	}

	return fmt.Errorf("dhcpv4 or ip address not found in wicked output")
}

func checkDHCPProcess() error {
	iface, err := utils.GetInterface(1)
	if err != nil {
		return fmt.Errorf("couldn't get secondary interface: %v", err)
	}

	cmd := exec.Command("ps", "x")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("ps command failed %v", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, fmt.Sprintf("dhclient %s", iface.Name)) {
			return nil
		}
	}
	return fmt.Errorf("failed finding dhclient process")
}

// accepts IP addresses in the form of a.b.c.d or a.b.c.d/IPNET
func validIPOrCIDR(token string) bool {
	if IPAddress := net.ParseIP(token); IPAddress != nil {
		return true
	}

	IPAddress, _, err := net.ParseCIDR(token)
	if IPAddress != nil && err == nil {
		return true
	}

	return false
}
