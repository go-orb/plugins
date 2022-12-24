#!/bin/bash
pushd registry/consul/test/bin/linux_amd64 || exit 1

if [ -e consul ]; then
    wget -q -O consul.zip https://releases.hashicorp.com/consul/1.14.3/consul_1.14.3_linux_amd64.zip
    unzip consul.zip
fi

popd || exit 1