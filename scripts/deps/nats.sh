#!/bin/bash
pushd registry/nats/test/bin/linux_amd64 || exit 1

ZIPFILE="nats-server-v2.9.20-linux-amd64"

if [[ ! -x nats-server ]]; then
	wget -q -O nats.zip https://github.com/nats-io/nats-server/releases/download/v2.9.20/${ZIPFILE}.zip
	unzip nats.zip "*/nats-server"
    
    mv "${ZIPFILE}/nats-server" .
    chmod +x nats-server

    rm -rf "./${ZIPFILE}"
    rm -f "nats.zip"
fi

popd || exit 1