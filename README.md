# Cloud Foundry 2 Humio

This project contains two components:

- A [Cloud Foundry](https://www.cloudfoundry.org/) [nozzle](https://docs.pivotal.io/tiledev/nozzle.html) for pushing logs to Humio
- A [Cloud Foundry](https://www.cloudfoundry.org/) Tile so that Humio can be accessed from the Pivotal Marketplace

For more on the design and layout of this project, see the accompanying [Design Overview](design.md) document.

# Cloud Foundry Nozzle

The [Cloud Foundry](https://www.cloudfoundry.org/)
[nozzle](https://docs.pivotal.io/tiledev/nozzle.html) drains logs from a
Cloud Foundry system and forwards them to [Humio](https://humio.com/) over the
[Elastic Search bulk](https://go.humio.com/docs/integrations/log-shippers/others/index.html#elasticsearch-bulk-api)
endpoint integration.

Please note that of all the 
[available Cloud Foundry events](https://github.com/cloudfoundry/dropsonde-protocol/tree/master/events),
only the HTTP start/stop and application log messages are forwarded to Humio.
Application's failures and metrics are not currently sent.

## Prepare your Cloud Foundry Environment for the Nozzle

### Requirements

This nozzle requires your environment to have the following available:

* a Cloud Foundry installation
* an admin user on that environment
* the [CF cli](https://github.com/cloudfoundry/cli)
* the [uaac cli](https://github.com/cloudfoundry/cf-uaac)

### Production Cloud Foundry Deployment

These instuctions work against a production Cloud Foundry deployment for which
you must have administrative scope. If you do not have access to such an environment you can install [PCFDev](https://pivotal.io/pcf-dev) locally and follow the
separate instructions below in this README that specifically work with [PCFDev](https://pivotal.io/pcf-dev).

#### Setup a firehose user

The nozzle requires a CF user who is authorized to access the loggregator
firehose through the `doppler.firehose` scope. It is best to have a dedicated
user for this access.

```
$ uaac target https://uaa.${ENDPOINT} --skip-ssl-validation
$ uaac token client get admin
$ cf create-user ${FIREHOSE_USER} ${FIREHOSE_USER_PASSWORD}
$ uaac member add cloud_controller.admin ${FIREHOSE_USER}
$ uaac member add doppler.firehose ${FIREHOSE_USER}
```

### Local Cloud Foundry Deployment

#### Create a local Cloud Foundry environment

You might want to work against a local Cloud Foundry instance by using
[PCFDev](https://pivotal.io/pcf-dev). Once installed, you can strt the PCFR local development environment using the following command:

```
$ cf dev start
```

Once the PCF development environment is started, run the following
commands:

```
$ export FIREHOSE_USER=hoseuser
$ export FIREHOSE_USER_PASSWORD=hosepwd
$ uaac target  https://uaa.local.pcfdev.io --skip-ssl-validation
$ cf login --skip-ssl-validation -u admin -p admin
```

You will then be prompted ... TBD

cf login --skip-ssl-validation -u admin -p admin

API endpoint> https://api.local.pcfdev.io
Authenticating...
OK

Select an org (or press enter to skip):
1. pcfdev-org
2. system

Org> 1
Targeted org pcfdev-org

Targeted space pcfdev-space



```
$ uaac token client get admin -s admin-client-secret
$ cf create-user ${FIREHOSE_USER} ${FIREHOSE_USER_PASSWORD}
$ uaac member add cloud_controller.admin ${FIREHOSE_USER}
$ uaac member add doppler.firehose ${FIREHOSE_USER}
```

## Get the nozzle

```
$ git clone https://github.com/humio/cloudfoundry2humio.git
$ cd cloudfoundry2humio
```

## Fill environment variables in [manifest.yml](./manifest.yml)
```
API_ADDR                  : The api URL of the CF environment (e.g. https://api.local.pcfdev.io:443)
DOPPLER_ADDR              : Loggregator's traffic controller URL (websocket) (e.g. wss://doppler.local.pcfdev.io:443)
FIREHOSE_USER             : CF user who has admin and firehose access
FIREHOSE_USER_PASSWORD    : Password of the CF user
HUMIO_HOST                : Address of the Humio ingester endpoint (e.g. https://go.humio.com:443)
HUMIO_DATASPACE           : Name of the Humio dataspace to send events to
HUMIO_INGEST_TOKEN        : Token for that particular dataspace
SKIP_SSL_VALIDATION       : If true, allows insecure connections to the UAA and the Trafficcontroller
CF_ENVIRONMENT            : Set to any string value for identifying logs and metrics from different CF environments
IDLE_TIMEOUT              : Keep Alive duration for the firehose consumer
LOG_LEVEL                 : Logging level of the nozzle, valid levels: DEBUG, INFO, ERROR
```

## Deploy

You can now run the following command to push the application to PCF to begin receiving logs to Humio:

```
$ cf push
```

## Develop

### Requirements

Working on this code base requires:

* [golang](https://golang.org/) >= 1.8
* [govendor](https://github.com/kardianos/govendor): `go get -u github.com/kardianos/govendor`

Ensure you have properly setup
[golang](https://github.com/golang/go/wiki/GOPATH).

### Pushing to Cloud Foundry

As seen before, you can push the nozzle as follows

```
$ cf push
```

You need to ensure the package is properly vendored as well first:

```
$ govendor update +local +vendor
```

If you do not do that, the shipped nozzle will be the one without your changes.

When working against Cloud Foundry, you may also enable more traces by
exporting the following variable:

```
$ export CF_TRACE=true
```

This will trace all communication between your machine and the remote CF API
endpoint.

### Local Build

You may build locally the nozzle for fast local development:

```
$ go build
```

which should generate the binary you can try locally:

```
./cloudfoundry2humio --api-addr https://api.local.pcfdev.io \
    --doppler-addr wss://doppler.local.pcfdev.io:443 \
    --firehose-user ${FIREHOSE_USER} \
    --firehose-user-password ${FIREHOSE_USER_PASSWORD} \
    --skip-ssl-validation \
    --humio-host https://go.humio.com:443 \
    --humio-dataspace Demo \
    --humio-ingest-token XYZ \
    --log-level DEBUG
```

./cloudfoundry2humio --api-addr https://api.local.pcfdev.io --doppler-addr wss://doppler.local.pcfdev.io:443 --firehose-user ${FIREHOSE_USER} --firehose-user-password ${FIREHOSE_USER_PASSWORD} --skip-ssl-validation --humio-host https://go.humio.com:443 --humio-dataspace testspace1 --humio-ingest-token yq1xhM77c3uyw80DlWOAT5jqs47HhE0KO2rTdDszG29e --log-level DEBUG

You may enable more logging by setting:

```
export GOREQUEST_DEBUG=1
```

before running the command.

### Run Local Tests

In order to execute this project's tests locally you will need to execute the following
commands to bring in a couple of test-only dependencies:

```
$ go get github.com/onsi/ginkgo
$ go get github.com/onsi/gomega
```

You can now run the local tests for the project by executing the following 
in the root of the project:

```
$ govendor test
```

## Release

To release a new version of this nozzle and tile, first update the version in
`main.go`.

Then, update the CHANGELOG, tag and release.

# The Humio Pivotal Cloud Foundry Tile

Deploying a tile into your Cloud Foundry environment requires administrator
permissons. As it also requires access to an Ops Manager, this cannot be tried
against PCFDev at this time.

You must install the
[tile generator](https://github.com/cf-platform-eng/tile-generator) to be able
to build this nozzle's tile.

You can generate a new version of the tile:

```
$ cd tile
$ bash build.sh
```

In order to
[deploy](http://docs.pivotal.io/tiledev/pcf-command.html#deploy-tiles)
this tile, create a
[metadata file](cloudfoundry2humio-0.0.1.pivotal) and run the following
command:

```
$ cd tile
$ pcf import product/cloudfoundry2humio-X.Y.Z.pivotal
```
