Archived. No maintenance. See official driver
=============================================

[![Build Status](https://travis-ci.org/huseyin/docker-machine-driver-scaleway.svg?branch=master)](https://travis-ci.org/huseyin/docker-machine-driver-scaleway)

Docker Machine Scaleway Driver is a driver plugin for Docker machine. It allows
to create Docker hosts on Scaleway servers.

Installation
------------

Use standard Unix tools:

	$ wget -O docker-machine-driver-scaleway.zip RELEASE_URL
	$ unzip docker-machine-driver-scaleway.zip
	$ sudo cp docker-machine-driver-scaleway /usr/bin

Usage
-----

### 1. Credentials

To use the driver, you must have an organization id and API token.

Follow these steps:

- [Retrieve organization ID](https://www.scaleway.com/docs/retrieve-my-organization-id-throught-the-api/)
- [Create API token](https://www.scaleway.com/docs/generate-an-api-token/)

### 2. Create a machine

These instructions assume that `docker-machine` and `docker-machine-driver-scaleway`
are in your PATH.

	$ docker-machine --driver scaleway \
		--scaleway-organization SCALEWAY_ORGANIZATION_ID \
		--scaleway-token SCALEWAY_API_TOKEN \
		MACHINE_NAME

**P.S.** Try the `overlay` driver for problems with `aufs` storage driver incompatibility
in the docker installation. See for details: https://docs.docker.com/machine/reference/create/#specifying-configuration-options-for-the-created-docker-engine

### 3. Setting up and tests

Load environment variables to use remote machine. This step is required to connect
to the Docker Engine socket of remote machine.

	$ eval $(docker-machine env MACHINE_NAME)

Run test commands

	$ docker pull golang
	$ docker images

### 4. Options

|Option                      |Description               |Default        |required|
|----------------------------|--------------------------|---------------|--------|
|`--scaleway-ssh-user`       |SSH username              |`root`         |no      |
|`--scaleway-ssh-port`       |SSH port                  |`22`           |no      |
|`--scaleway-organization`   |Organization id           |`none`         |yes     |
|`--scaleway-token`          |API token                 |`none`         |yes     |
|`--scaleway-server-name`    |Server name               |`none`         |no      |
|`--scaleway-commercial-type`|Commercial type           |`VC1S`         |no      |
|`--scaleway-image`          |Image                     |`ubuntu-xenial`|no      |
|`--scaleway-region`         |Region                    |`ams1`         |no      |
|`--scaleway-reserved-ip-id` |Use an existing IP adress |`none`         |no      |
|`--scaleway-persistent-ip`  |IP persistent             |`false`        |no      |
|`--scaleway-enable-ipv6`    |Enable IPv6               |`false`        |no      |
|`--scaleway-volumes`        |Add an additional volume  |`none`         |no      |
|`--scaleway-tags`           |Add tags                  |`none`         |no      |

Build from source
-----------------

Requirements:

- A working Go environment (see: https://golang.org/doc/code.html)
- [Golint](https://github.com/golang/lint)
- [Dep](https://github.com/golang/dep)

Install dependencies

	go get -u github.com/golang/lint/golint
	go get -u github.com/golang/dep/cmd/dep

Run tests

	$ make test

Run build command

	$ make build

Todo
----

- Security groups
- User datas

License
-------

The MIT License (MIT) - see [`LICENSE`](https://github.com/huseyin/docker-machine-driver-scaleway/blob/master/LICENSE) for more details
