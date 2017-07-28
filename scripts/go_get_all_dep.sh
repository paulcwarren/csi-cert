#!/bin/bash

set -e

echo "installing lager..."
go get -u "code.cloudfoundry.org/lager"
go get -u "code.cloudfoundry.org/lager/lagertest"
echo "installing gomega..."
go get -u "github.com/onsi/gomega"
go get -u "github.com/onsi/gomega/types"
echo "installing context..."
go get -u "golang.org/x/net/context"
echo "installing grpc..."
go get -u "google.golang.org/grpc"
echo "installing csi spec..."
go get -u "github.com/container-storage-interface/spec">/dev/null 2>&1 || true

echo "building csi proto..."
pushd $GOPATH/src/github.com/container-storage-interface/spec
  go get -u github.com/golang/protobuf/{proto,protoc-gen-go}
  make csi.proto
  make csi.pb.go
popd
echo "done."
