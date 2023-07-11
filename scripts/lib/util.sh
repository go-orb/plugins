## bash
RED='\033[0;31m'
NC='\033[0m'
GREEN='\033[0;32m'
BAR="-------------------------------------------------------------------------------"

export RICHGO_FORCE_COLOR="true"
# export IN_TRAVIS_CI="true"
# export TRAVIS="true"

# Print a green colored message to the screen.
function print_msg() {
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

	print_msg "Found ${#dirs[@]} directories to test"
	echo "Changed dirs:"
	printf '%s \n' "${dirs[@]}"
	printf '\n\n'
	sleep 1
}

# Add a job summary to GitHub Actions.
[ -z ${GITHUB_STEP_SUMMARY+x} ] && GITHUB_STEP_SUMMARY=$(mktemp)

function add_summary() {
	printf "${1}\n" >>"${GITHUB_STEP_SUMMARY}"
}