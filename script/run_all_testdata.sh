#!/bin/bash
# Generate uniast for all testdata.
#
# USAGE:
# 1. Save the uniast to out/
# $ OUTDIR=out/ ./script/run_testdata.sh
#
# 2. Save the uniast to out/ , colorize output for human readable terminal
# OUTDIR=out/ PARALLEL_FLAGS=--ctag ./script/run_testdata.sh
#
# 3. Use a custom abcoder executable
# OUTDIR=out/ ABCEXE="./other_abcoder" ./script/run_testdata.sh

SCRIPT_DIR=$(dirname "$(readlink -f "$0")")
REPO_ROOT=$(realpath --relative-to=$(pwd) "$SCRIPT_DIR/..")

ABCEXE=${ABCEXE:-"$REPO_ROOT/abcoder"}
OUTDIR=${OUTDIR:?Error: OUTDIR is a mandatory environment variable}
PARALLEL_FLAGS=${PARALLEL_FLAGS:---tag}

LANGS=(go rust python cxx)

detect_jobs() {
	local ABCEXE=${1:-$ABCEXE}
	for lang in ${LANGS[@]}; do
		for repo in "$REPO_ROOT/testdata/$lang"/*; do
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
