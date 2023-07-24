source ${SCRIPT_DIR}/lib/util.sh

function run() {
	print_header "Running linter on ${1}"

	if ! cd "${1}"; then
		print_red_msg "Failed to change dir"
		exit 1
	fi

	$(go env GOPATH)/bin/golangci-lint run --out-format github-actions -c "${2}/.golangci.yaml"

	# Keep track of exit code of linter
	if [[ $? -ne 0 ]]; then
		print_red_msg "Failed"
		exit 1
	fi

    exit 0
}

out="$(run $1 "${ORB_ROOT}" 2>&1)"
rc=$?

printf "%s\n" "${out}"
exit $rc