#!/usr/bin/env bash

export SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
export ORB_ROOT=$(realpath "${SCRIPT_DIR}/..")

# Run all tests/benchmarks with a single cpu core.
export GOMAXPROCS=1

export ORB_GO_TEST_FLAGS="-v -race -cover"

source "${SCRIPT_DIR}/lib/util.sh"


# Find directories that contain changes.
function find_changes() {
	# Find all directories that have changed files.
	changes=($(git diff --name-only origin/main | xargs -d'\n' -I{} dirname {} | sort -u))

	# Filter out directories without go.mod files.
	changes=($(find "${changes[@]}" -maxdepth 1 -name 'go.mod' -printf '%h\n' 2>/dev/null))

	echo "${changes[@]}"
}

# Find all go directories.
function find_all() {
	find "${ORB_ROOT}" -name 'go.mod' -printf '%h\n'
}

# Get the dir list based on command type.
function get_dirs() {
	if [[ $1 == "all" ]]; then
		echo $(find_all)
	elif [[ $1 == "changes" ]]; then
		echo $(find_changes)
	else
		echo ${@}
	fi
}

# Run GoLangCi Linters.
function run_linter() {
	[[ -e $(go env GOPATH)/bin/golangci-lint ]] || curl -sSfL "https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh" | sh -s -- -b $(go env GOPATH)/bin

	$(go env GOPATH)/bin/golangci-lint --version

	dirs=$1
	printf "%s\0" "${dirs[@]}" | xargs -0 -n1 -P $(nproc) -- /usr/bin/env bash "${SCRIPT_DIR}/lib/lint.sh"
	rc=$?

	if [[ "${rc}" != "0" ]]; then
		print_red_header "Lint failed"
	else
		print_header "Lint OK"
	fi
}

# Run Unit tests with RichGo for pretty output.
function run_test() {
	dirs=$1

	print_header "Downloading go dependencies..."

	go install github.com/kyoh86/richgo@latest

	for dir in "${dirs[@]}"; do
		bash -c "cd ${dir}; go mod tidy &>/dev/null"
	done

	dirs=$1
	printf "%s\0" "${dirs[@]}" | xargs -0 -n1 -P $(nproc) -- /usr/bin/env bash "${SCRIPT_DIR}/lib/test.sh"
	rc=$?

	if [[ $? != 0 ]]; then
		print_red_header "Tests failed"
		exit 1
	fi
}

# Run unit tests with tparse to create a summary.
function create_summary() {
	go install github.com/mfridman/tparse@latest

	add_summary "## Test Summary"

	cwd=$(pwd)
	dirs=$1
	failed="false"
	for dir in "${dirs[@]}"; do
		# Install dependencies if required.
		pre_test "${dir}"

		pushd "${dir}" >/dev/null || continue
		print_header "Creating summary for ${dir}"

		add_summary "\n### ${dir}\n"

		# Download all modules.
		go get -v -t -d ./...

		go test ./... ${GO_TEST_FLAGS} -json |
			tparse -notests -format=markdown >>"${GITHUB_STEP_SUMMARY}"

		if [[ $? -ne 0 ]]; then
			failed="true"
			print_red_msg "Failed"
		fi

		popd >/dev/null || continue

		# Kill all depdency processes.
		post_test "${dir}"

		print_msg "Succeded"
	done

	if [[ ${failed} == "true" ]]; then
		print_red_header "Tests failed"
		exit 1
	fi
}

if [[ ! -d ../go-orb ]]; then
	git clone https://github.com/go-orb/go-orb ../go-orb
fi

case $1 in
"lint")
	read -a dirs <<< $(get_dirs "${@:2}")
	[[ ${#dirs[@]} -eq 0 ]] && print_red_header "No changed Go files detected" && exit 0

	print_list "${dirs[@]}"
	run_linter "${dirs[@]}"
	;;
"test")
	read -a dirs <<< $(get_dirs "${@:2}")
	[[ ${#dirs[@]} -eq 0 ]] && print_red_header "No changed Go files detected" && exit 0

	print_list "${dirs[@]}"

	run_test "${dirs[@]}"
	;;
"summary")
	read -a dirs <<< $(get_dirs "${@:2}")
	[[ ${#dirs[@]} -eq 0 ]] && print_red_header "No changed Go files detected" && exit 0

	print_list "${dirs[@]}"
	create_summary "${dirs[@]}"
	;;
"")
	printf "Please provide a command [lint, test, summary]."
	exit 1
	;;
*)
	printf "Command not found: $1. Select one of [lint, test, summary]"
	exit 1
	;;
esac
