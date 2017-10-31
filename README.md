# Humio Nozzle For Cloud Foundry

This is the Cloud Foundry Nozzle to drain logs from a Cloud Foundry system
and forward them to Humio over their ElasticSearch endpoint.

## Prerequisites

### Environment

This nozzle expects that your environment is properly set:

* a Cloud Foundry installation
* an admin user on that environment
* the [CF cli](https://github.com/cloudfoundry/cli)
* the [uaac cli](https://github.com/cloudfoundry/cf-uaac)

For simple test, you may want to install [PCFDev](https://pivotal.io/pcf-dev)
locally.

### Deploy - Push the Nozzle as an App to Cloud Foundry

#### 1. Use the CF CLI to authenticate with your CF instance
```
$ cf login -a https://api.${ENDPOINT} -u ${CF_USER} --skip-ssl-validation
```

#### 2. Create a CF user and grant required privileges
The nozzle requires a CF user who is authorized to access the loggregator
firehose, e.g. with the `doppler.firehose` scope.

```
$ uaac target https://uaa.${ENDPOINT} --skip-ssl-validation
$ uaac token client get admin
$ cf create-user ${FIREHOSE_USER} ${FIREHOSE_USER_PASSWORD}
$ uaac member add cloud_controller.admin ${FIREHOSE_USER}
$ uaac member add doppler.firehose ${FIREHOSE_USER}
```

#### 3. Set environment variables in [manifest.yml](./manifest.yml)
```
API_ADDR                  : The api URL of the CF environment
DOPPLER_ADDR              : Loggregator's traffic controller URL
FIREHOSE_USER             : CF user who has admin and firehose access
FIREHOSE_USER_PASSWORD    : Password of the CF user
EVENT_FILTER              : Event types to be filtered out. The format is a comma separated list, valid event types are METRIC,LOG,HTTP
SKIP_SSL_VALIDATION       : If true, allows insecure connections to the UAA and the Trafficcontroller
CF_ENVIRONMENT            : Set to any string value for identifying logs and metrics from different CF environments
IDLE_TIMEOUT              : Keep Alive duration for the firehose consumer
LOG_LEVEL                 : Logging level of the nozzle, valid levels: DEBUG, INFO, ERROR
LOG_EVENT_COUNT           : If true, the total count of events that the nozzle has received and sent will be logged to OMS Log Analytics as CounterEvents
LOG_EVENT_COUNT_INTERVAL  : The time interval of logging event count to OMS Log Analytics
```


### 4. Push the app
```
$ cf push
```

## Test

You need [ginkgo](https://github.com/onsi/ginkgo) to run the test.
Run the following command to execute test:

```
$ ginkgo -r
```
