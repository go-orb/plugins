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

CHANGELOG_TEMPLATE="${SCRIPT_DIR}/template/changelog.md"
CHANGELOG_FILE="/tmp/changelog.md"

function increment_minor_version() {
	declare -a part=(${1//\./ })
	part[2]=0
	part[1]=$((part[1] + 1))
	new="${part[*]}"
	echo -e "${new// /.}"
}

function increment_patch_version() {
	declare -a part=(${1//\./ })
	part[2]=$((part[2] + 1))
	new="${part[*]}"
	echo -e "${new// /.}"
}

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

function check_if_changed() {
	local pkg="$1"
	local last_tag=$(get_last_tag "${pkg}")
	if [[ ${last_tag} == "" ]]; then
		return 1
	fi

	if ! git diff --name-only "${last_tag}" HEAD | grep -E "${pkg}/[0-9\.a-zA-Z\-_]+$" > /dev/null; then
		return 1
	fi
	return 0
}

function get_latest_version() {
    local pkg="$1"
    local last_tag=$(get_last_tag "${pkg}")
    if [[ ${last_tag} == "" ]]; then
        return 1
    fi

	declare -a last_tag_split=(${last_tag//\// })

	local v_version=${last_tag_split[-1]}
	echo "${v_version}"

    return 0
}

function update_deps() {
    local pkg="$1"
    local version="$2"

    pushd "${pkg}" >/dev/null || exit
    go get -u github.com/go-orb/go-orb@latest 1>/dev/null 2>&1
    for m in $(grep github.com/go-orb/plugins/ go.mod | grep -E -v "^module" | awk '{ print $1 }'); do 
        current_pkg=${m#github.com/go-orb/plugins/}
        if ! get_last_tag "${current_pkg}" 1>/dev/null; then
            echo "updated_deps: Package ${current_pkg} not released yet."
            exit 1
        fi

        version=$(get_latest_version "${current_pkg}")
        if ! go get "${m}@${version}" 1>/dev/null 2>&1; then
            echo "updated_deps: Failed to update dependency ${current_pkg} to version ${version}"
            exit 1
        fi
    done
    go mod tidy 1>/dev/null 2>&1

    git add go.mod go.sum 1>/dev/null 2>&1
    git commit -S -m "release(${pkg}): ${version}" 1>/dev/null 2>&1
    git push 1>/dev/null 2>&1
    popd >/dev/null || exit

    return 0
}

function ask_for_confirmation() {
	local -r message="Do you want to release $1/$2?"

	read -p "${message} [y/N]: " -n 1 -r
	echo
	if [[ $REPLY =~ ^[Yy]$ ]]; then
		return 0
	fi
	return 1
}

function release() {
	if [[ ! -f "${1}/go.mod" ]]; then
		echo "Unknown package '${1}' given."
		return 1
	fi

	local pkg="${1}"
	local drymode="$2"

    if ! get_last_tag "${pkg}" 1>/dev/null; then
		if [[ "${drymode}" == "1" ]]; then
			echo "Would release ${pkg}/v0.1.0"
			return 0
		fi

        if ! ask_for_confirmation "${pkg}" "v0.1.0"; then
            return 1
        fi

        update_deps "${pkg}" "v0.1.0"
        gh release create "${pkg}/v0.1.0" -n 'Initial release'
        git fetch 1>/dev/null 2>&1
        return 0
	fi

	if ! check_if_changed "${pkg}"; then
		return 1
	fi

	cat "${CHANGELOG_TEMPLATE}" >"${CHANGELOG_FILE}"

	local last_tag=$(get_last_tag "${pkg}")

	# Create changelog file
	git log "${last_tag}..HEAD" --format="%s" "${pkg}" |
		xargs -d'\n' -I{} bash -c "echo \"  * {}\" >> ${CHANGELOG_FILE}"

	declare -a last_tag_split=(${last_tag//\// })

	local v_version=${last_tag_split[-1]}
	local version=${v_version:1}
	# Remove the version from last_tag_split
	unset last_tag_split[-1]

	# Increment minor version if "feat:" commit found, otherwise patch version
	git --no-pager log "${last_tag}..HEAD" --format="%s" "${pkg}/*" | grep -q -E "^feat:"
	if [[ $? == "0" ]]; then
		local tmp_new_tag="$(printf "/%s" "${last_tag_split[@]}")/v$(increment_minor_version "${version}")"
		local new_tag=${tmp_new_tag:1}
	else
		local tmp_new_tag="$(printf "/%s" "${last_tag_split[@]}")/v$(increment_patch_version "${version}")"
		local new_tag=${tmp_new_tag:1}
	fi

	if [[ "${drymode}" == "1" ]]; then
		echo "Would release ${new_tag}"
		git diff --name-only "${last_tag}" HEAD | grep "${pkg}"
		return 0
	fi

	if ! ask_for_confirmation "${pkg}" "${new_tag}"; then
		return 1
	fi	

    update_deps "${pkg}" "${new_tag}"
	gh release create "${new_tag}" --notes-file "${CHANGELOG_FILE}"
    git fetch 1>/dev/null 2>&1
}

function release_all() {
    for pkg in $(python3 ${SCRIPT_DIR}/release_order.py); do
	    release "${pkg}" "0"
	done
}

function release_specific() {
	set +o noglob
	while read -r pkg; do
		# If path contains a star find all relevant packages
		if echo "${pkg}" | grep -q "\*"; then
			while read -r p; do
				release "$(remove_prefix "${p}")" "0"
			done < <(find "${pkg}" -name 'go.mod' -printf "%h\n")
		else
			release "${pkg}" "0"
		fi
	done < <(echo "${1}" | tr "," "\n")
	# set -o noglob
	# set +o noglob
}

case $1 in
"all")
	release_all
	;;
*)
	release_specific "${1}"
	;;
esac