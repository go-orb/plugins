#!/usr/bin/env bash
set -e; set -o pipefail

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

# https://gist.github.com/lukechilds/a83e1d7127b78fef38c2914c4ececc3c
function get_latest_gh_release() {
  curl --silent "https://api.github.com/repos/$1/releases/latest" | # Get latest release from GitHub api
    grep '"tag_name":' |                                            # Get tag line
    sed -E 's/.*"([^"]+)".*/\1/'                                    # Pluck JSON value
}

GOOS=$(go env GOOS)
GOARCH=$(go env GOARCH)

VERSION=$(get_latest_gh_release nats-io/nats-server)
ZIPFILE="nats-server-${VERSION}-${GOOS}-${GOARCH}"

WORKDIR="$(realpath "${SCRIPT_DIR}/..")/test/bin/${GOOS}_${GOARCH}"

mkdir -p "${WORKDIR}"
pushd "${WORKDIR}"

if [[ ! -x nats-server ]]; then
    echo "Downloading NATS ${VERSION}"

    curl -s -L https://github.com/nats-io/nats-server/releases/download/${VERSION}/${ZIPFILE}.zip -o nats.zip
	unzip nats.zip "*/nats-server" 1>/dev/null
    
    mv "${ZIPFILE}/nats-server" .
    chmod +x nats-server

    rm -rf "./${ZIPFILE}"
    rm -f "nats.zip"
fi

popd

exit 0