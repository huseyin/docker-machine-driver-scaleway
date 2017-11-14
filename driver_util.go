package scaleway

import (
	"strings"

	scw "github.com/scaleway/scaleway-cli/pkg/api"
)

type client struct {
	api    *scw.ScalewayAPI
	driver *Driver
}

func newClient(d *Driver) (*client, error) {
	scwAPI, err := scw.NewScalewayAPI(d.Organization, d.Token, "", d.Region)
	if err != nil {
		return nil, err
	}

	return &client{scwAPI, d}, nil
}

func (c *client) createServer(config *scw.ConfigCreateServer) (string, error) {
	return scw.CreateServer(c.api, config)
}

func (c *client) startServer() error {
	return c.api.PostServerAction(c.driver.ServerID, "poweron")
}

func (c *client) rebootServer() error {
	return c.api.PostServerAction(c.driver.ServerID, "reboot")
}

func (c *client) stopServer() error {
	return c.api.PostServerAction(c.driver.ServerID, "poweroff")
}

func (c *client) removeServer() error {
	if err := c.api.DeleteServerForce(c.driver.ServerID); err != nil {
		return err
	}

	_, err := scw.WaitForServerState(c.api, c.driver.ServerID, "")
	if err != nil {
		return nil
	}

	if !c.driver.PersistentIP {
		if err = c.api.DeleteIP(c.driver.IPID); err != nil {
			return err
		}
	}

	return nil
}

func (c *client) waitForServerReady() error {
	_, err := scw.WaitForServerReady(c.api, c.driver.ServerID, "")
	return err
}

func (c *client) checkCredentials() error {
	return c.api.CheckCredentials()
}

func (c *client) reserveIP() (*scw.ScalewayGetIP, error) {
	if c.driver.IPID != "" {
		return c.api.GetIP(c.driver.IPID)
	}

	return c.api.NewIP()
}

func (c *client) getServer() (*scw.ScalewayServer, error) {
	return c.api.GetServer(c.driver.ServerID)
}

func (c *client) tags() string {
	var tagList []string

	for _, t := range strings.Split(c.driver.Tags, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			tagList = append(tagList, t)
		}
	}

	return strings.Join(tagList, " ")
}
