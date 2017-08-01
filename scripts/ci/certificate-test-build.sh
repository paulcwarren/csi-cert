#!/bin/bash

set -ex

# Installing protoc
wget https://github.com/google/protobuf/releases/download/v3.3.0/protoc-3.3.0-linux-x86_64.zip
unzip protoc-3.3.0-linux-x86_64.zip
mv bin/protoc /usr/bin

work_dir=$(pwd)

mkdir -p ${work_dir}/go/src/github.com/paulcwarren/

pushd ${work_dir}/go
  export GOROOT=/usr/local/go
  export PATH=$GOROOT/bin:$PATH
  export GOPATH=$PWD
  export PATH=$PWD/bin:$PATH

  go get github.com/onsi/ginkgo/ginkgo

  ln -s ${work_dir}/csi-cert  src/github.com/paulcwarren/csi-cert

  cd src/github.com/paulcwarren/csi-cert
  ./scripts/go_get_all_dep.sh

  GOOS=linux ginkgo build -r
  mv csi-cert.test ${work_dir}/binaries/csi-cert.test.linux
  GOOS=darwin ginkgo build -r
  mv csi-cert.test ${work_dir}/binaries/csi-cert.test.darwin
  GOOS=windows ginkgo build -r
  mv csi-cert.test ${work_dir}/binaries/csi-cert.test.windows
popd

pushd ${work_dir}/binaries/
  tar -czvf csi-cert.test.tgz csi-cert.test.{linux,darwin,windows}
popd


