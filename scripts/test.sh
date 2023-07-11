#!/usr/bin/env bash

export SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

MICRO_VERSION="."

source ${SCRIPT_DIR}/lib/util.sh


# Install dependencies, usually servers.
#
# Can can be used to run an script needed to complete tests.
#
# To run a script add it to the HAS_DEPS variable, e.g.: ("redis" "nats").
# And make sure to add a script to deps/<name>.sh
function install_deps() {
	for dep in "${HAS_DEPS[@]}"; do
		if grep -q "${dep}" <<<"${1}"; then
			script="scripts/deps/${dep}.sh"

			# Check if script exists
			if [[ -f ${script} ]]; then
				echo "Installing depencies for ${dep}"
				bash "${script}"
				echo "${dep}"
				return 0
			fi
		fi
	done
}

# Kill all PIDs of setups.
function kill_deps() {
	for dep in "${HAS_DEPS[@]}"; do
		if grep -q "${dep}" <<<"${1}"; then
			# Itterate over all PIDs and kill them.
			pids=($(pgrep "${dep}"))
			if [[ ${#pids[@]} -ne 0 ]]; then
				echo "Killing:"
			fi

			for dep_pid in "${pids[@]}"; do
				ps -aux | grep -v "grep" | grep "${dep_pid}"

				kill "${dep_pid}"
				return 0
			done
		fi
	done
}

# Find directories that contain changes.
function find_changes() {
	# Find all directories that have changed files.
	changes=($(git diff --name-only origin/main | xargs -d'\n' -I{} dirname {} | sort -u))

	# Filter out directories without go.mod files.
	changes=($(find "${changes[@]}" -maxdepth 1 -name 'go.mod' -printf '%h\n'))

	echo "${changes[@]}"
}

# Find all go directories.
function find_all() {
	find "${MICRO_VERSION}" -name 'go.mod' -printf '%h\n'
}

# Get the dir list based on command type.
function get_dirs() {
	if [[ $1 == "all" ]]; then
		find_all
	else
		find_changes
	fi
}

# Run GoLangCi Linters.
function run_linter() {
	[[ -e $(go env GOPATH)/bin/golangci-lint ]] || curl -sSfL "https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh" | sh -s -- -b $(go env GOPATH)/bin

	$(go env GOPATH)/bin/golangci-lint --version

	dirs=$1
	cwd=$(dirname "${SCRIPT_DIR}/..")

	printf "%s\0" "${dirs[@]}" | xargs -0 -n1 -P $(nproc) -- /bin/bash -c '
		source ${SCRIPT_DIR}/lib/util.sh

		function run_lint() {
			print_msg "Running linter on ${1}"

			if ! cd "${1}"; then
				echo "::error cd ${1}"
				exit 1
			fi

			$(go env GOPATH)/bin/golangci-lint run --out-format github-actions -c "${2}/.golangci.yaml"

			# Keep track of exit code of linter
			if [[ $? -ne 0 ]]; then
				echo "::error lint ${1}"
				exit 1
			fi
		}

		printf "%s\n" "$(run_lint $1 $(dirname "${SCRIPT_DIR}") 2>&1)"
	' '_'
}

# Run Unit tests with RichGo for pretty output.
function run_test() {
	cwd=$(pwd)
	dirs=$1
	failed="false"

	print_msg "Downloading dependencies..."

	go install github.com/kyoh86/richgo@latest

	for dir in "${dirs[@]}"; do
		bash -c "cd ${dir}; go mod tidy &>/dev/null"
	done

	for dir in "${dirs[@]}"; do
		print_msg "Running unit tests for ${dir}"

		# Install dependencies if required.
		install_deps "${dir}"

		pushd "${dir}" >/dev/null || exit

		# Download all modules.
		go get -v -t -d ./...

		# Run tests.
		$(go env GOPATH)/bin/richgo test ./... ${GO_TEST_FLAGS}

		# Keep track of exit code.
		if [[ $? -ne 0 ]]; then
			failed="true"
		fi

		popd >/dev/null || exit

		# Kill all depdency processes.
		kill_deps "${dir}"
	done

	if [[ ${failed} == "true" ]]; then
		print_red "Tests failed"
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
		install_deps "${dir}"

		pushd "${dir}" >/dev/null || continue
		print_msg "Creating summary for ${dir}"

		add_summary "\n### ${dir}\n"

		# Download all modules.
		go get -v -t -d ./...

		go test "${GO_TEST_FLAGS}" -json ./... |
			tparse -notests -format=markdown >>"${GITHUB_STEP_SUMMARY}"

		if [[ $? -ne 0 ]]; then
			failed="true"
		fi

		popd >/dev/null || continue

		# Kill all depdency processes.
		kill_deps "${dir}"
	done

	if [[ ${failed} == "true" ]]; then
		print_red "Tests failed"
		exit 1
	fi
}

[ ! -d ../go-orb ] && git clone https://github.com/go-orb/go-orb ../go-orb

case $1 in
"lint")
	dirs=($(get_dirs "${2}"))
	[[ ${#dirs[@]} -eq 0 ]] && print_red "No changed Go files detected" && exit 0

	print_list "${dirs[@]}"
	run_linter "${dirs[@]}"
	;;
"test")
	dirs=($(get_dirs "${2}"))
	[[ ${#dirs[@]} -eq 0 ]] && print_red "No changed Go files detected" && exit 0

	print_list "${dirs[@]}"

	run_test "${dirs[@]}"
	;;
"summary")
	dirs=($(get_dirs "${2}"))
	[[ ${#dirs[@]} -eq 0 ]] && print_red "No changed Go files detected" && exit 0

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
