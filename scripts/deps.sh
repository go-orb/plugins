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

function remove_prefix() {
	echo "${1//\.\//}"
}

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
    for pkg in $(python3 ${SCRIPT_DIR}/release_order.py); do
	    upgrade "${pkg}" "0"
	done
}

function upgrade_specific() {
	set +o noglob
	while read -r pkg; do
		# If path contains a star find all relevant packages
		if echo "${pkg}" | grep -q "\*"; then
			while read -r p; do
				update_deps "$(remove_prefix "${p}")" "0"
			done < <(find "${pkg}" -name 'go.mod' -printf "%h\n")
		else
			update_deps "${pkg}" "0"
		fi
	done < <(echo "${1}" | tr "," "\n")
	# set -o noglob
	# set +o noglob
}

case $1 in
"all")
	upgrade_all
	;;
*)
	upgrade_specific "${1}"
esac