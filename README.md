# CSI-Cert

The purpose of CSI-cert is to create a certification test, which can be used to test controller and node plugins, to check if they implement [Container Storage Interface(CSI)](https://github.com/container-storage-interface/spec/blob/master/spec.md) Spec. 



## Terminology

| Term              | Definition                                       |
|-------------------|--------------------------------------------------|
| Volume            | A unit of storage that will be made available inside of a CO-managed container, via the CSI.                          |
| Block Volume      | A volume that will appear as a block device inside the container.                                                     |
| Mounted Volume    | A volume that will be mounted using the specified file system and appear as a directory inside the container.         |
| CO                | Container Orchestration system, communicates with Plugins using CSI service RPCs.                                     |
| SP                | Storage Provider, the vendor of a CSI plugin implementation.                                                          |
| RPC               | [Remote Procedure Call](https://en.wikipedia.org/wiki/Remote_procedure_call).                                         |
| Node              | A host where the user workload will be running, uniquely identifiable from the perspective of a Plugin by a `NodeID`. |
| Plugin            | Aka “plugin implementation”, a gRPC endpoint that implements the CSI Services.                                        |
| Plugin Supervisor | Process that governs the lifecycle of a Plugin, MAY be the CO.                                                        |


## Objective

To evaluate whether the controlle/node plugin is compliant with the RPC service defined in CSI spec.

### Goals in MVP

The CSI-Cert will evaluate whether the controller plugin is:

* Responding to GetCapabilities RPC call, and run different tests according to controller's capability.
* Responding to CreateVolume/DeleteVolume/ControllerPublishVolume/ControllerUnpublishVolume requests
* Responding to CreateVolume/DeleteVolume/ControllerPublishVolume/ControllerUnpublishVolume request idempotently (Send the request for the second time will get the same response)
* Responding with an error response to a CreateVolume/DeleteVolume request without volume name.
* Able to list the volume been created.
* Responding to GetCapacity request with none empty response.

The CSI-Cert will evaluate whether the node plugin is 
* Responding to NodePublishVolume/NodeUnpublishVolume requests
* Responding to NodePublishVolume/NodeUnpublishVolume requests idempotently (Send the request for the second time will get the same response)
* Responding with an error response to a NodePublishVolume/NodeUnpublishVolume request without volume ID.

### Non-Goals in MVP

The CSI-Cert will not:

* Evaluate the usability of the Volume in CO's perspective.
* Evaluate paging aspects of ListVolume of the controller plugin.
* Evaluate the response of GetCapacity matches the real capacity.

## How to use

### Pre-requirement

* Install [golang](https://golang.org/doc/install).

* Setup GOPATH

```bash
export GOPATH=<YOUR_GOPATH>
```
* Setup PATH to include GOPATH/bin

```bash
export PATH=$GOPATH/bin:$GOPATH
```

* Start the plugins you want to test.

* Create a json file with the address/port your plugins are listening on. 

example:
```json
{
  "controller_address": "127.0.0.1:50051",
  "node_address":"127.0.0.1:50052"
}
```

* Set environment variable `FIXTURE_FILENAME` to point to the json file created:

```bash
export FIXTURE_FILENAME=<YOUR_JSON_FILE>
``` 

### Run the test 

#### Using go get
```bash
go get github.com/paulcwarren/csi-cert
go get github.com/onsi/ginkgo/ginkgo
pushd $GOPATH/src/github.com/paulcwarren/csi-cert
./scripts/go_get_all_dep.sh
ginkgo -r -p
```

#### Using pre-built binaries
1. Download the [latest binary](https://github.com/paulcwarren/csi-cert/releases)
1. Unpackage the tgz file ```tar -xzvf csi-cert.test.tgz```
1. Run the test ```./csi-cert.test.<YOUR_OS>```
