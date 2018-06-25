# Openstack Broker

## Introduction
This is a PoC Openstack broker to load APB's based on available Openstack Resources. It was created by vendoring the Automation Broker.

## To build
dep ensure
go build ./cmd/openstackbroker && mv openstackbroker build

## To containerize
pushd build; docker build -t docker.io/jmontleon/openstackbroker:latest . && docker push docker.io/jmontleon/openstackbroker:latest; popd

## To install and run in an Openshift Cluster
Set up devstack with HEAT and an image with which to test.

A local.conf similar to this should work. Make sure port 80 and 443 are open on the VM. devstack does not appear to take care of this.

```
iptables -I INPUT -p tcp -m state --state NEW -m tcp --dport 80 -j ACCEPT
iptables -I INPUT -p tcp -m state --state NEW -m tcp --dport 443 -j ACCEPT
```


```
ADMIN_PASSWORD=password
DATABASE_PASSWORD=password
RABBIT_PASSWORD=password
SERVICE_PASSWORD=password
enable_plugin heat https://git.openstack.org/openstack/heat
IMAGE_URL_SITE="http://download.fedoraproject.org"
IMAGE_URL_PATH="/pub/fedora/linux/releases/28/Cloud/x86_64/images/"
IMAGE_URL_FILE="Fedora-Cloud-Base-28-1.1.x86_64.qcow2"
IMAGE_URLS+=","$IMAGE_URL_SITE$IMAGE_URL_PATH$IMAGE_URL_FILE
```

Use oc cluster up to bring up a cluster with the automation broker. Something similar to below should work:

```
oc cluster up --routing-suffix=172.17.0.1.nip.io --public-hostname=172.17.0.1 --base-dir=/tmp/openshift.local.clusterup --tag=latest --image=docker.io/openshift/origin-\\${component}:\\${version} --enable=service-catalog,template-service-broker,router,registry,web-console,persistent-volumes,sample-templates,rhel-imagestreams,automation-service-broker
```

Add the openstack broker to your setup:

```
oc process -f template-openstack-broker.yaml -p OPENSTACK_URL="https://$IP-OF-DEVSTACK-VM" -p OPENSTACK_USER=admin -p OPENSTACK_PASS=password | oc create -f -
```

## TODO
* Add other services and more options for VM's.
* Test and improve.
