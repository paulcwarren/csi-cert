#!/bin/bash

set -ex

# Installing protoc
wget https://github.com/google/protobuf/releases/download/v3.3.0/protoc-3.3.0-linux-x86_64.zip
unzip protoc-3.3.0-linux-x86_64.zip
mv bin/protoc /usr/bin

work_dir=$(pwd)

pushd csi-cert
  export GOROOT=/usr/local/go
  export PATH=$GOROOT/bin:$PATH

  export GOPATH=$PWD
  export PATH=$PWD/bin:$PATH

  ./scripts/go_get_all_dep.sh
  GOOS=linux ginkgo build -r
  mv csi-cert.test ${work_dir}/binaries/csi-cert.test.linux
  GOOS=darwin ginkgo build -r
  mv csi-cert.test ${work_dir}/binaries/csi-cert.test.darwin
  GOOS=windows ginkgo build -r
  mv csi-cert.test ${work_dir}/binaries/csi-cert.test.windows
popd


