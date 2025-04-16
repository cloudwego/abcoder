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
echo "[Making parser]"
./script/make_parser.sh
echo "[Done making parser]"

parser=tools/parser/lang
mkdir -p testdata/jsons

do_test() {
	lang=$1
	srcpath=$2
	name=$3
	flags=$4

	echo $name...
	$parser -d -v --no-need-comment collect $lang $srcpath > testdata/jsons/$name.json 2>testdata/jsons/$name.log
	cat testdata/jsons/$name.log
	python script/check_lineno.py --json testdata/jsons/$name.json --base $srcpath $flags > testdata/jsons/$name.check

	if grep -q "All functions verified successfully!" testdata/jsons/$name.check; then
		echo "  [PASS]"
	else
		echo "  [FAIL]"
		exit 1
	fi
}
do_test go src/lang go "--zero_linebase"
do_test rust testdata/rust2-wobyted rust2 "--zero_linebase --implheads"
