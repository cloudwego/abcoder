#!/bin/bash
# Copyright 2025 CloudWeGo Authors
# 
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
# 
#     https://www.apache.org/licenses/LICENSE-2.0
# 
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

root=$(dirname $(realpath $(dirname $0)))
cd $root

mkdir -p testdata/jsons

do_test() {
	name="ast"
	lang=$1
	srcpath=$2
	flags=$4

	echo "go run . parse $lang $srcpath  -verbose --no-need-comment > testdata/jsons/$name.json"
	go run . parse $lang $srcpath -verbose --no-need-comment > testdata/jsons/$name.json 
	python3 script/check_lineno.py --json testdata/jsons/$name.json --base $srcpath $flags > testdata/jsons/$name.check

	if grep -q "All functions verified successfully!" testdata/jsons/$name.check; then
		echo "  [PASS]"
	else
		echo "  [FAIL]"
		exit 1
	fi
}
do_test go testdata/golang "--zero_linebase"
do_test rust testdata/rust2 "--zero_linebase --implheads"
