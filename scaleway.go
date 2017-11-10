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
	uuid "github.com/satori/go.uuid"
	"github.com/scaleway/scaleway-cli/pkg/api"
	"github.com/scaleway/scaleway-cli/pkg/clilogger"
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
	IPPersistent   bool
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
			EnvVar: "SCALEWAY_IP_PERSISTENT",
			Name:   "scaleway-ip-persistent",
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
	d.IPPersistent = flags.Bool("scaleway-ip-persistent")
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

// PreCreateCheck allows for pre-create operations to make sure a driver is
// ready for creation.
func (d *Driver) PreCreateCheck() error {
	if d.IPID != "" {
		_, err := uuid.FromString(d.IPID)
		if err != nil {
			return fmt.Errorf("IP UUID %v invalid", d.IPID)
		}
	}

	return nil
}

// Create creates a new server using the Scaleway API and the helper methods of
// the *Driver instance.
func (d *Driver) Create() error {
	log.Infof("Creating SSH key for server...")
	publicKey, err := d.createSSHKey()
	if err != nil {
		return err
	}

	client, err := d.getClient()
	if err != nil {
		return err
	}

	log.Infof("Reserving IP...")
	if err = d.reserveIP(client); err != nil {
		return err
	}

	tags := d.sshKeyEnvFormat(publicKey)
	if d.Tags != "" {
		tags = strings.Join([]string{tags, d.getTags()}, " ")
	}

	serverConfig := &api.ConfigCreateServer{
		Name:              d.ServerName,
		CommercialType:    d.CommercialType,
		ImageName:         d.Image,
		IP:                d.IPAddress,
		EnableIPV6:        d.EnableIPv6,
		AdditionalVolumes: d.Volumes,
		Env:               tags,
	}

	log.Infof("Creating server...")
	d.ServerID, err = api.CreateServer(client, serverConfig)
	if err != nil {
		return err
	}

	log.Info("Waiting for server to be ready...")

	return api.StartServer(client, d.ServerID, true)
}

// getClient returns a Scaleway API instance using the *Driver parameters.
func (d *Driver) getClient() (*api.ScalewayAPI, error) {
	return api.NewScalewayAPI(d.Organization, d.Token, "", d.Region, clilogger.SetupLogger)
}

// createSSHKey creates a new SSH key and returns the public key.
func (d *Driver) createSSHKey() (string, error) {
	if err := ssh.GenerateSSHKey(d.GetSSHKeyPath()); err != nil {
		return "", err
	}

	publicKey, err := ioutil.ReadFile(d.publicSSHKeyPath())
	if err != nil {
		return "", err
	}

	return string(publicKey), nil
}

// publicSSHKeyPath returns the public key filename using the SSH key path.
func (d *Driver) publicSSHKeyPath() string {
	return d.GetSSHKeyPath() + ".pub"
}

// sshKeyEnvFormat returns the SSH key environment variable pair for the server.
func (d *Driver) sshKeyEnvFormat(key string) string {
	key = strings.Replace(key[:len(key)-1], " ", "_", -1)
	return "AUTHORIZED_KEY" + "=" + key
}

// reserveIP reserves and sets the IP address for the server. If IP is given as
// a command line argument, it checks its presence and uses it.
func (d *Driver) reserveIP(client *api.ScalewayAPI) error {
	if d.IPID != "" {
		ips, err := client.GetIPS()
		if err != nil {
			return err
		}

		found := false

		for _, ip := range ips.IPS {
			if ip.ID == d.IPID {
				d.IPAddress = ip.Address
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("IP UUID %v not found", d.IPID)
		}
	} else {
		ip, err := client.NewIP()
		if err != nil {
			return err
		}

		d.IPID = ip.IP.ID
		d.IPAddress = ip.IP.Address
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

// GetState returns the state of the server.
func (d *Driver) GetState() (state.State, error) {
	client, err := d.getClient()
	if err != nil {
		return state.Error, err
	}

	server, err := client.GetServer(d.ServerID)
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
	st, err := d.GetState()
	if err != nil {
		return err
	}

	if st == state.Starting || st == state.Running {
		log.Infof("Server is already running")
		return nil
	}

	return d.postAction("poweron")
}

// Stop stops the server using the API wrapper. If the server is already stopping,
// the wrapper is not called.
func (d *Driver) Stop() error {
	st, err := d.GetState()
	if err != nil {
		return err
	}

	if st == state.Stopping || st == state.Stopped {
		log.Infof("Server is already stopped")
		return nil
	}

	return d.postAction("poweroff")
}

// Restart restarts the server using the API wrapper.
func (d *Driver) Restart() error {
	return d.postAction("reboot")
}

// Kill kills the server using the API wrapper.
func (d *Driver) Kill() error {
	return errors.New("kill is not supported for scaleway driver")
}

// Remove deletes the server and optionally the resources.
func (d *Driver) Remove() error {
	client, err := d.getClient()
	if err != nil {
		return nil
	}

	if err = client.DeleteServerForce(d.ServerID); err != nil {
		return err
	}

	for {
		_, err = client.GetServer(d.ServerID)
		if err != nil {
			break
		}
	}

	if !d.IPPersistent {
		if err = client.DeleteIP(d.IPID); err != nil {
			return err
		}
	}

	return nil
}

// postAction is an API wrapper for Scaleway servers. It performs some basic
// actions such as turning poweron/poweroff the server.
func (d *Driver) postAction(action string) error {
	client, err := d.getClient()
	if err != nil {
		return err
	}

	return client.PostServerAction(d.ServerID, action)
}

func (d *Driver) getTags() string {
	var tagList []string

	for _, t := range strings.Split(d.Tags, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			tagList = append(tagList, t)
		}
	}

	return strings.Join(tagList, " ")
}
