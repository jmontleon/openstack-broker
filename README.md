# Openstack Broker

## Introduction
This is a PoC Openstack broker to load APB's based on available Openstack Resources. It was created by vendoring the Automation Broker.

## To build
dep ensure
go build ./cmd/openstackbroker && mv openstackbroker build

## To containerize
pushd build; docker build -t docker.io/jmontleon/openstackbroker:latest . && docker push docker.io/jmontleon/openstackbroker:latest; popd

## To install and run in an Openshift Cluster
WIP

## TODO
* Add meaningful functionality. As of now it compiles but that's about it.
  * Build APB Definitions and load them based on available Openstack Resources.
  * Create an Openstack Runner APB that can launch a VM and/or some other service
