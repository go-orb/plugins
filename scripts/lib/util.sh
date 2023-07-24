## bash
RED='\033[0;31m'
NC='\033[0m'
GREEN='\033[0;32m'
BAR="-------------------------------------------------------------------------------"

export RICHGO_FORCE_COLOR="true"
# export IN_TRAVIS_CI="true"
# export TRAVIS="true"

function print_msg() {
	printf "${GREEN} > ${1}${NC}\n"
}

# Print a green colored message to the screen.
function print_header() {
	printf "\n\n${GREEN}${BAR}${NC}\n"
	printf "${GREEN}| > ${1}${NC}\n"
	printf "${GREEN}${BAR}${NC}\n\n"
	sleep 1
}

# Print a red colored message to the screen.
function print_red() {
	printf "\n\n${RED}${BAR}${NC}\n"
	printf "${RED}| > ${1}${NC}\n"
	printf "${RED}${BAR}${NC}\n\n"
	sleep 1
}

# Print the contents of the directory array.
function print_list() {
	dirs=$1

	print_header "Found ${#dirs[@]} directories to test"
	echo "Changed dirs:"
	printf '%s \n' "${dirs[@]}"
	printf '\n\n'
	sleep 1
}

# Run a pre_test script for a plugin, usualy downloads a server.
#
function pre_test() {
	if [[ ! -e "${1}/scripts/pre_test.sh" ]]; then
		# Return no error if no such script
		return 0
	fi

	print_msg "Executing pre test for ${1}"
	${1}/scripts/pre_test.sh
	return $?
}

# Run post_test script.
function post_test() {
	if [[ ! -e "${1}/scripts/post_test.sh" ]]; then
		# Return no error if no such script
		return 0
	fi

	print_msg "Executing post test for ${1}"
	${1}/scripts/post_test.sh
	return $?
}

# https://gist.github.com/lukechilds/a83e1d7127b78fef38c2914c4ececc3c
function get_latest_gh_release() {
  curl --silent "https://api.github.com/repos/$1/releases/latest" | # Get latest release from GitHub api
    grep '"tag_name":' |                                            # Get tag line
    sed -E 's/.*"([^"]+)".*/\1/'                                    # Pluck JSON value
}

# Add a job summary to GitHub Actions.
[ -z ${GITHUB_STEP_SUMMARY+x} ] && GITHUB_STEP_SUMMARY=$(mktemp)

function add_summary() {
	printf "${1}\n" >>"${GITHUB_STEP_SUMMARY}"
}