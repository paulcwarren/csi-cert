#!/bin/bash

set -ex

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


