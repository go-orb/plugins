#!/usr/bin/env bash
set -e; set -o pipefail

# Import util.sh
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source ${SCRIPT_DIR}/../../../scripts/lib/util.sh

GOOS=$(go env GOOS)
GOARCH=$(go env GOARCH)

VERSION=$(get_latest_gh_release nats-io/nats-server)
ZIPFILE="nats-server-${VERSION}-${GOOS}-${GOARCH}"

WORKDIR="$(realpath "${SCRIPT_DIR}/..")/test/bin/${GOOS}_${GOARCH}"

mkdir -p "${WORKDIR}"
pushd "${WORKDIR}"

if [[ ! -x nats-server ]]; then
    print_msg "Downloading NATS ${VERSION}"

    curl -s -L https://github.com/nats-io/nats-server/releases/download/${VERSION}/${ZIPFILE}.zip -o nats.zip
	unzip nats.zip "*/nats-server" 1>/dev/null
    
    mv "${ZIPFILE}/nats-server" .
    chmod +x nats-server

    rm -rf "./${ZIPFILE}"
    rm -f "nats.zip"
fi

popd

exit 0