#!/bin/bash
# Generate uniast for all testdata.
#
# USAGE:
# 1. Save the uniast for all testdata to out/
# $ OUTDIR=out/ ./script/run_testdata.sh all
#
# 2. Save the uniast for the first testdata item (0_*) in each language to out/
# $ OUTDIR=out/ ./script/run_testdata.sh first
#
# 3. Use a custom abcoder executable
# OUTDIR=out/ ABCEXE="./other_abcoder" ./script/run_testdata.sh all

if [[ "$1" != "all" && "$1" != "first" ]]; then
	echo "Usage: $0 all|first" >&2
	echo "	all:   Run on all testdata." >&2
	echo "	first: Run only on testdata starting with '0_*' in each language directory." >&2
	exit 1
fi
MODE=$1

SCRIPT_DIR=$(dirname "$(readlink -f "$0")")
REPO_ROOT=$(realpath --relative-to=$(pwd) "$SCRIPT_DIR/..")

ABCEXE=${ABCEXE:-"$REPO_ROOT/abcoder"}
OUTDIR=${OUTDIR:?Error: OUTDIR is a mandatory environment variable}
PARALLEL_FLAGS=${PARALLEL_FLAGS:---tag}
LANGS=${LANGS:-"go rust python cxx"}

detect_jobs() {
	local ABCEXE=${1:-$ABCEXE}
	for lang in ${LANGS[@]}; do
	local repo_glob="$REPO_ROOT/testdata/$lang/*"
	if [[ "$MODE" == "first" ]]; then
		repo_glob="$REPO_ROOT/testdata/$lang/0_*"
	fi
	for repo in $repo_glob; do
		# Skip if glob doesn't match anything to avoid errors
		[[ -e "$repo" ]] || continue
		local rel_path=$(realpath --relative-to="$REPO_ROOT/testdata" "$repo")
		local outname=$(echo "$rel_path" | sed 's/[/:? ]/_/g')
		echo $ABCEXE parse $lang $repo -o $OUTDIR/$outname.json
		done
	done
}

if [[ ! -x "$ABCEXE" ]]; then
	echo "Error: The specified abcoder executable '$ABCEXE' does not exist or is not executable." >&2
	exit 1
fi
mkdir -pv "$OUTDIR"
detect_jobs
echo
detect_jobs | parallel $PARALLEL_FLAGS -j$(nproc --all) --jobs 0 "eval {}" 2>&1
