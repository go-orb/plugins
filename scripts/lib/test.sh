source ${SCRIPT_DIR}/lib/util.sh

function run() {
    dir="${1}"

    print_header "Running unit tests for ${dir}"

    # Install dependencies if required.
    pre_test "${dir}"

    pushd "${dir}" >/dev/null || exit

    # Download all modules.
    go get -v -t -d ./...

    # Run tests.
    $(go env GOPATH)/bin/richgo test ./... ${ORB_GO_TEST_FLAGS}

    # Keep track of exit code.
    if [[ $? -ne 0 ]]; then
        print_red_msg "Failed"
        exit 1
    fi

    popd >/dev/null || exit

    # Kill all depdency processes.
    post_test "${dir}"
}

printf "%s\n" "$(run $1 "${ORB_ROOT}" 2>&1)"