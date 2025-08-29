#!/bin/bash
# Generate uniast for all testdata. Must be run from repo root
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

ABCEXE=${ABCEXE:-./abcoder}
OUTDIR=${OUTDIR:?Error: OUTDIR is a mandatory environment variable}
PARALLEL_FLAGS=${PARALLEL_FLAGS:---tag}

LANGS=(go rust python cxx)

detect_jobs() {
	local ABCEXE=${1:-$ABCEXE}
	for lang in ${LANGS[@]}; do
		for repo in testdata/$lang/*; do
			outname=$(echo $repo | sed 's/^testdata\///; s/[/:? ]/_/g')
			echo $ABCEXE parse $lang $repo -o $OUTDIR/$outname.json
		done
	done
}

mkdir -pv "$OUTDIR"
detect_jobs | parallel $PARALLEL_FLAGS echo {}
echo
detect_jobs | parallel $PARALLEL_FLAGS -j$(nproc --all) --jobs 0 "eval {}" 2>&1
