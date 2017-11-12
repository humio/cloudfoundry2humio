# Design Overview of the PCF2Humio Logging Nozzle

The `cloudfoundry2humio` project provides a logging bridge between a Cloud Native application running on Pivotal Cloud Foundry to Humio.

## Package Organisation

This codebase is organised across two main packages:

* `humio` is the directory/package that contains the functions to push events to Humio. It simply does so via a HTTP POST call (using `gorequest̀`). The `events.go` module contains the functions to map PCF events to Humio events format.

* `nozzle` is the directory/package that contains two concerns: the firehose client (that's the websocket client to the PCF event hose, it relies on the PCF `noaa` library) and the `nozzle` functions that consume from the firehose, map events to an acceptable Humio format and then push those events to Humio (using the `humio` package as previously described). It also listens to signals (such as SIGINT/Ctrl-C) to stop the nozzle app. _Note_: Events aren't buffered until either the buffer reaches 500 events or 5s have passed since the last push.

## Extending for new Events

You can extend this codebase to support additional logging events from your Cloud Native applications running on Pivotal Cloud Foundry, such as perhaps consuming service instance runtime metrics,by extending the components in the `humio/events.go` module in order to map these new events over into what is pushed to Humio.