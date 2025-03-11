#!/bin/bash

######################################################################################
# Release a plugin                                                                   #
#                                                                                    #
# Usage:                                                                             #
#   $ release.sh all                                                                 #
#   $ release.sh server/http                                                         #
#   $ release.sh server/http,server/grpc                                             #
#   $ release.sh server/*                                                            #
#                                                                                    #
######################################################################################

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

function get_last_tag() {
	local pkg="$1"
	local last_tag=$(git tag --list --sort='-creatordate' | grep -E "${pkg}/v[0-9\.]+" | head -n1)
	if [[ ${last_tag} == "" ]]; then
		return 1
	fi

    echo "${last_tag}"

    return 0
}

# Check if the pkg has been changed on git or in filesystem since the last release.
# Return 1 if changed, 0 if not changed.
function check_if_changed() {
	local pkg="$1"
	local last_tag=$(get_last_tag "${pkg}")
	if [[ ${last_tag} == "" ]]; then
		return 1
	fi

	# Check if the package has been changed on git.
	if git diff --name-only "${last_tag}" HEAD | grep -E "${pkg}/[0-9\.a-zA-Z\-_]+$" > /dev/null; then
		echo "# Changed on git diff" 
		return 1
	fi

	# Check if the package has been changed on filesystem.
	if git status --porcelain -s | grep -E "${pkg}/[0-9\.a-zA-Z\-_]+$" > /dev/null; then
		echo "# Changed on filesystem"
		return 1
	fi
	
	return 0
}

function update_deps() {
    local pkg="$1"

    pushd "${pkg}" >/dev/null || exit
	go mod tidy || true
    go get -u github.com/go-orb/go-orb@main
    for m in $(grep github.com/go-orb/plugins/ go.mod | grep -E -v "^module" | awk '{ print $1 }'); do 
        if ! go get -u "${m}@main"; then
			# try another time
			sleep 5
			go get -u "${m}@main"
			if ! go get -u "${m}@main"; then
				echo "updated_deps: Failed to update dependency ${m}"
				exit 1
			fi
		fi
    done

	for m in $(grep github.com/go-orb/plugins-experimental/ go.mod | grep -E -v "^module" | awk '{ print $1 }'); do 
        if ! go get -u "${m}@main"; then
			# try another time
			sleep 5
			go get "${m}@main"
			if ! go get -u "${m}@main"; then
				echo "updated_deps: Failed to update dependency ${m}"
				exit 1
			fi
		fi
    done

    go mod tidy
    popd >/dev/null || exit

    return 0
}

function upgrade() {
	if [[ ! -f "${1}/go.mod" ]]; then
		echo "Unknown package '${1}' given."
		return 1
	fi

	local pkg="${1}"

	echo "Checking ${pkg}"
	if check_if_changed "${pkg}"; then
		return 0
	fi

	echo "Update deps for ${pkg}"
    update_deps "${pkg}"
}

function upgrade_all() {
    find . -name 'go.mod' -print0 | xargs -0 -n 1 -P 0 ${SCRIPT_DIR}/deps.sh
}

function upgrade_specific() {
	while read -r pkg; do
		if [[ "${pkg}" == "./.github/go.mod" ]]; then
			continue
		fi

		update_deps "${pkg%go.mod}" "0"
	done < <(echo "${1}" | tr "," "\n")
}

case $1 in
"all")
	upgrade_all
	;;
*)
	upgrade_specific "${1}"
esac