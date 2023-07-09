#!/bin/bash
pushd registry/consul/test/bin/linux_amd64 || exit 1

if [ ! -x consul ]; then
    wget -q -O consul.zip https://releases.hashicorp.com/consul/1.16.0/consul_1.16.0_linux_amd64.zip
    unzip consul.zip
fi

popd || exit 1