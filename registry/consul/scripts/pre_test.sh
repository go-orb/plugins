#!/usr/bin/env bash
set -e; set -o pipefail

# Import util.sh
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source ${SCRIPT_DIR}/../../../scripts/lib/util.sh

GOOS=$(go env GOOS)
GOARCH=$(go env GOARCH)

VERSION="$(get_latest_gh_release hashicorp/consul)"
VERSION="${VERSION:1}" # Remove the leading v

mkdir -p test/bin/${GOOS}_${GOARCH}
pushd test/bin/${GOOS}_${GOARCH}

if [[ ! -x consul ]]; then
	print_msg "Downloading curl ${VERSION}"

	curl -s -L https://releases.hashicorp.com/consul/${VERSION}/consul_${VERSION}_${GOOS}_${GOARCH}.zip -o consul.zip
	unzip consul.zip 1>/dev/null
	chmod +x consul
	rm -f consul.zip
fi

popd

exit 0
