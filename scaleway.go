package scaleway

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"strings"

	"github.com/docker/machine/libmachine/drivers"
	"github.com/docker/machine/libmachine/log"
	"github.com/docker/machine/libmachine/mcnflag"
	"github.com/docker/machine/libmachine/ssh"
	"github.com/docker/machine/libmachine/state"
	"github.com/scaleway/scaleway-cli/pkg/api"
)

const (
	defaultImage          = "ubuntu-xenial"
	defaultCommercialType = "VC1S"
	defaultRegion         = "ams1"
)

// Driver represents the Scaleway Docker Machine Driver and limits.
type Driver struct {
	*drivers.BaseDriver
	Organization   string
	Token          string
	ServerID       string
	ServerName     string
	CommercialType string
	Image          string
	Region         string
	IPID           string
	PersistentIP   bool
	EnableIPv6     bool
	Volumes        string
	Tags           string
}

// NewDriver returns a new Scaleway driver instance using the default and
// optional arguments.
func NewDriver(hostName, storePath string) drivers.Driver {
	return &Driver{
		Image:          defaultImage,
		CommercialType: defaultCommercialType,
		Region:         defaultRegion,
		BaseDriver: &drivers.BaseDriver{
			MachineName: hostName,
			StorePath:   storePath,
		},
	}
}

// DriverName returns the name of the driver.
func (d *Driver) DriverName() string {
	return "scaleway"
}

// GetCreateFlags registers the "machine create" flags recognized by this driver,
// including their help text and defaults.
func (d *Driver) GetCreateFlags() []mcnflag.Flag {
	return []mcnflag.Flag{
		mcnflag.StringFlag{
			EnvVar: "SCALEWAY_SSH_USER",
			Name:   "scaleway-ssh-user",
			Usage:  "SSH user name",
			Value:  drivers.DefaultSSHUser,
		},
		mcnflag.IntFlag{
			EnvVar: "SCALEWAY_SSH_PORT",
			Name:   "scaleway-ssh-port",
			Usage:  "SSH port",
			Value:  drivers.DefaultSSHPort,
		},
		mcnflag.StringFlag{
			EnvVar: "SCALEWAY_ORGANIZATION",
			Name:   "scaleway-organization",
			Usage:  "Scaleway organization id",
		},
		mcnflag.StringFlag{
			EnvVar: "SCALEWAY_TOKEN",
			Name:   "scaleway-token",
			Usage:  "Scaleway access token",
		},
		mcnflag.StringFlag{
			EnvVar: "SCALEWAY_SERVER_NAME",
			Name:   "scaleway-server-name",
			Usage:  "Scaleway server name",
		},
		mcnflag.StringFlag{
			EnvVar: "SCALEWAY_COMMERCIAL_TYPE",
			Name:   "scaleway-commercial-type",
			Usage:  "Scaleway commercial type (e.g.: vc1s)",
			Value:  defaultCommercialType,
		},
		mcnflag.StringFlag{
			EnvVar: "SCALEWAY_IMAGE",
			Name:   "scaleway-image",
			Usage:  "Scaleway image name (e.g.: ubuntu-xenial)",
			Value:  defaultImage,
		},
		mcnflag.StringFlag{
			EnvVar: "SCALEWAY_REGION",
			Name:   "scaleway-region",
			Usage:  "Scaleway region name (e.g.: ams1,par1)",
			Value:  defaultRegion,
		},
		mcnflag.StringFlag{
			EnvVar: "SCALEWAY_RESERVED_IP_ID",
			Name:   "scaleway-reserved-ip-id",
			Usage:  "Scaleway reserved IP id",
		},
		mcnflag.BoolFlag{
			EnvVar: "SCALEWAY_PERSISTENT_IP",
			Name:   "scaleway-persistent-ip",
			Usage:  "enable IP persistent",
		},
		mcnflag.BoolFlag{
			EnvVar: "SCALEWAY_ENABLE_IPv6",
			Name:   "scaleway-enable-ipv6",
			Usage:  "enable IPv6 for server",
		},
		mcnflag.StringFlag{
			EnvVar: "SCALEWAY_VOLUMES",
			Name:   "scaleway-volumes",
			Usage:  "attach additional volume (e.g.: 50G)",
		},
		mcnflag.StringFlag{
			EnvVar: "SCALEWAY_TAGS",
			Name:   "scaleway-tags",
			Usage:  "comma-separated list of tags to apply to the server",
		},
	}
}

// SetConfigFromFlags initializes driver values from the command line values and
// checks if the arguments have values.
func (d *Driver) SetConfigFromFlags(flags drivers.DriverOptions) error {
	d.SSHUser = flags.String("scaleway-ssh-user")
	d.SSHPort = flags.Int("scaleway-ssh-port")
	d.Organization = flags.String("scaleway-organization")
	d.Token = flags.String("scaleway-token")
	d.ServerName = flags.String("scaleway-server-name")
	d.CommercialType = flags.String("scaleway-commercial-type")
	d.Image = flags.String("scaleway-image")
	d.Region = flags.String("scaleway-region")
	d.IPID = flags.String("scaleway-reserved-ip-id")
	d.PersistentIP = flags.Bool("scaleway-persistent-ip")
	d.EnableIPv6 = flags.Bool("scaleway-enable-ipv6")
	d.Volumes = flags.String("scaleway-volumes")
	d.Tags = flags.String("scaleway-tags")

	d.SetSwarmConfigFromFlags(flags)

	if d.Organization == "" {
		return errors.New("scaleway driver requires the --scaleway-organization option")
	}

	if d.Token == "" {
		return errors.New("scaleway driver requires the --scaleway-token option")
	}

	return nil
}

// GetURL returns a socket address to connect to Docker engine of the server.
func (d *Driver) GetURL() (string, error) {
	if err := drivers.MustBeRunning(d); err != nil {
		return "", err
	}

	ip, err := d.GetIP()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("tcp://%s", net.JoinHostPort(ip, "2376")), nil
}

// GetSSHHostname returns an IP address or hostname for the instance.
func (d *Driver) GetSSHHostname() (string, error) {
	return d.GetIP()
}

// PreCreateCheck allows for pre-create operations to make sure a driver is
// ready for creation.
func (d *Driver) PreCreateCheck() error {
	c, err := newClient(d)
	if err != nil {
		return err
	}

	return c.checkCredentials()
}

// Create creates a new server using the Scaleway API and the helper methods of
// the *Driver instance.
func (d *Driver) Create() error {
	c, err := newClient(d)
	if err != nil {
		return err
	}

	log.Infof("Creating SSH key for server...")
	pub, err := d.createSSHKey()
	if err != nil {
		return err
	}

	log.Infof("Reserving IP...")
	ip, err := c.reserveIP()
	if err != nil {
		return err
	}

	serverConfig := &api.ConfigCreateServer{
		Name:              d.ServerName,
		CommercialType:    d.CommercialType,
		ImageName:         d.Image,
		IP:                ip.IP.Address,
		EnableIPV6:        d.EnableIPv6,
		AdditionalVolumes: d.Volumes,
		Env:               d.authorizedKey(pub) + " " + c.tags(),
	}

	log.Infof("Creating server...")
	d.ServerID, err = c.createServer(serverConfig)
	if err != nil {
		return err
	}

	log.Infof("Starting server...")
	if err = c.startServer(); err != nil {
		return err
	}

	log.Info("Waiting for server to be ready...")
	return c.waitForServerReady()
}

// GetState returns the state of the server.
func (d *Driver) GetState() (state.State, error) {
	c, err := newClient(d)
	if err != nil {
		return state.Error, err
	}

	server, err := c.getServer()
	if err != nil {
		return state.Error, err
	}

	switch server.State {
	case "starting":
		return state.Starting, nil
	case "running":
		return state.Running, nil
	case "stopping":
		return state.Stopping, nil
	case "stopped":
		return state.Stopped, nil
	}

	return state.None, nil
}

// Start starts the server using the API wrapper. If the server is already running,
// the wrapper is not called.
func (d *Driver) Start() error {
	c, err := newClient(d)
	if err != nil {
		return err
	}

	return c.startServer()
}

// Stop stops the server using the API wrapper. If the server is already stopping,
// the wrapper is not called.
func (d *Driver) Stop() error {
	c, err := newClient(d)
	if err != nil {
		return err
	}

	return c.stopServer()
}

// Restart restarts the server using the API wrapper.
func (d *Driver) Restart() error {
	c, err := newClient(d)
	if err != nil {
		return err
	}

	return c.rebootServer()
}

// Kill kills the server using the API wrapper.
func (d *Driver) Kill() error {
	return errors.New("kill is not supported for scaleway driver")
}

// Remove deletes the server and optionally the resources.
func (d *Driver) Remove() error {
	c, err := newClient(d)
	if err != nil {
		return err
	}

	return c.removeServer()
}

func (d *Driver) authorizedKey(pub string) string {
	pub = strings.Replace(pub[:len(pub)-1], " ", "_", -1)
	return "AUTHORIZED_KEY" + "=" + pub
}

func (d *Driver) createSSHKey() (string, error) {
	if err := ssh.GenerateSSHKey(d.GetSSHKeyPath()); err != nil {
		return "", err
	}

	pub, err := ioutil.ReadFile(d.publicSSHKeyPath())
	if err != nil {
		return "", err
	}

	return string(pub), nil
}

func (d *Driver) publicSSHKeyPath() string {
	return d.GetSSHKeyPath() + ".pub"
}
