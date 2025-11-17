// Copyright 2025 CloudWeGo Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pkg

import . "a.b/c/pkg/entity"

var G2 = G1

var I1 Integer

type S MyStructD

func CaseMergeRef() MyStruct {
	_ = G1 + G2
	return MyStruct{
		MyStructD: MyStructD{},
	}
}
