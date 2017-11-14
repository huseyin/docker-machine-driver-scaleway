package scaleway

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/docker/machine/commands/commandstest"
)

const (
	testMachineName = "test-machine"
	testStorePath   = "test-store-path"
)

const (
	testSSHUser        = "scw-user"
	testSSHPort        = 22
	testOrganization   = "d82f47f0-0613-4012-bfbb-34625b1eecb3"
	testToken          = "a08090cd-824f-4e82-868e-dba3578111d2"
	testServerName     = "scw-server"
	testCommercialType = "VC1M"
	testImage          = "scw-image"
	testRegion         = "ams1"
	testReservedIPID   = "bcdf8013-c01f-4897-bd3c-14f5d44321e4"
	testPersistentIP   = true
	testEnableIPv6     = true
	testVolumes        = "100G"
	testTags           = "foo,bar,baz"
)

var d = func() *Driver {
	td := NewDriver(testMachineName, testStorePath)
	td.SetConfigFromFlags(&commandstest.FakeFlagger{
		Data: map[string]interface{}{
			"scaleway-ssh-user":        testSSHUser,
			"scaleway-ssh-port":        testSSHPort,
			"scaleway-organization":    testOrganization,
			"scaleway-token":           testToken,
			"scaleway-server-name":     testServerName,
			"scaleway-commercial-type": testCommercialType,
			"scaleway-image":           testImage,
			"scaleway-region":          testRegion,
			"scaleway-reserved-ip-id":  testReservedIPID,
			"scaleway-persistent-ip":   testPersistentIP,
			"scaleway-enable-ipv6":     testEnableIPv6,
			"scaleway-volumes":         testVolumes,
			"scaleway-tags":            testTags,
		},
	})

	return td.(*Driver)
}()

func TestMachineNameAndStorePath(t *testing.T) {
	if testMachineName != d.GetMachineName() {
		t.Errorf("Expecting '%s', got '%s'\n", testMachineName, d.GetMachineName())
	}

	if testStorePath != d.StorePath {
		t.Errorf("Expecting '%s', got '%s'\n", testStorePath, d.StorePath)
	}
}

func TestSSHConfigs(t *testing.T) {
	if testSSHUser != d.GetSSHUsername() {
		t.Errorf("Expecting '%s', got '%s'\n", testSSHUser, d.GetSSHUsername())
	}

	actualSSHPort, err := d.GetSSHPort()
	if err != nil {
		fmt.Fprintf(os.Stderr, "err %v\n", err)
		t.Error("failed to get ssh port")
	}

	if testSSHPort != actualSSHPort {
		t.Errorf("Expecting '%d', got '%d'\n", testSSHPort, actualSSHPort)
	}
}

func TestScalewayConfigs(t *testing.T) {
	if testOrganization != d.Organization {
		t.Errorf("Expecting '%s', got '%s'\n", testOrganization, d.Organization)
	}

	if testToken != d.Token {
		t.Errorf("Expecting '%s', got '%s'\n", testToken, d.Token)
	}

	if testServerName != d.ServerName {
		t.Errorf("Expecting '%s', got '%s'\n", testServerName, d.ServerName)
	}

	if testCommercialType != d.CommercialType {
		t.Errorf("Expecting '%s', got '%s'\n", testCommercialType, d.CommercialType)
	}

	if testImage != d.Image {
		t.Errorf("Expecting '%s', got '%s'\n", testImage, d.Image)
	}

	if testRegion != d.Region {
		t.Errorf("Expecting '%s', got '%s'\n", testRegion, d.Region)
	}

	if testReservedIPID != d.IPID {
		t.Errorf("Expecting '%s', got '%s'\n", testReservedIPID, d.IPID)
	}

	if testPersistentIP != d.PersistentIP {
		t.Errorf("Expecting '%v', got '%v'\n", testPersistentIP, d.PersistentIP)
	}

	if testEnableIPv6 != d.EnableIPv6 {
		t.Errorf("Expecting '%v', got '%v'\n", testEnableIPv6, d.EnableIPv6)
	}

	if testVolumes != d.Volumes {
		t.Errorf("Expecting '%s', got '%s'\n", testVolumes, d.Volumes)
	}

	c, err := newClient(d)
	if err != nil {
		t.Error(err)
	}

	actualTags := strings.Replace(c.tags(), " ", ",", -1)
	if !strings.Contains(actualTags, testTags) {
		t.Errorf("Expecting '%s', got '%s'\n", testTags, actualTags)
	}
}
