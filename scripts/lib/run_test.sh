source ${SCRIPT_DIR}/lib/util.sh

function run() {
    dir="${1}"

    print_header "Running unit tests for ${dir}"

    # Install dependencies if required.
    pre_test "${dir}"

    pushd "${dir}" >/dev/null || exit

    if [[ "x${ORB_NODOWNLOAD}" == "x" ]]; then
        print_msg "Downloading go modules"
        go mod download &>/dev/null 2>&1
        echo ""
    fi

    # Run tests.
    if [[ -f "COVERPKGS" ]]; then
        $(go env GOPATH)/bin/richgo test ./... -v -benchmem -bench=. -cover -coverpkg="$(cat COVERPKGS | tr '\n' ',')"
    else
        $(go env GOPATH)/bin/richgo test ./... ${ORB_GO_TEST_FLAGS}
    fi

    rc=$?

    popd >/dev/null || exit

    # Kill all depdency processes.
    post_test "${dir}"

    return ${rc}
}

out=$(run $1 "${ORB_ROOT}" 2>&1)
rc=$?
printf "%s" "${out}"

exit ${rc}