#!/usr/bin/env bash
set -e; set -o pipefail

# https://gist.github.com/lukechilds/a83e1d7127b78fef38c2914c4ececc3c
get_latest_gh_release() {
  curl --silent "https://api.github.com/repos/$1/releases/latest" | # Get latest release from GitHub api
    grep '"tag_name":' |                                            # Get tag line
    sed -E 's/.*"([^"]+)".*/\1/'                                    # Pluck JSON value
}

VERSION="$(get_latest_gh_release hashicorp/consul)"
VERSION="${VERSION:1}" # Remove the leading v

GOOS=$(go env GOOS)
GOARCH=$(go env GOARCH)

mkdir -p test/bin/${GOOS}_${GOARCH}
pushd test/bin/${GOOS}_${GOARCH}

if [[ ! -x consul ]]; then
	curl -s -L https://releases.hashicorp.com/consul/${VERSION}/consul_${VERSION}_${GOOS}_${GOARCH}.zip -o consul.zip
	unzip consul.zip
	chmod +x consul
	rm -f consul.zip
fi

popd

exit 0
