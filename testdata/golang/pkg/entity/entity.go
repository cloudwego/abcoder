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

package entity

type MyStructC struct {
}

type MyStructD struct {
}

type MyStruct struct {
	a string
	b string
	c MyStructC
	MyStructD
}

type InterfaceB interface {
	String() string
}

func (a MyStruct) String() string {
	return "base struct"
}

func (c MyStructC) String() string {
	return "I'm struct c"
}

func (c MyStructD) String() string {
	return "I'm struct d"
}

func A(in string) int {
	return len(in)
}

const G1 = 1

type Integer int

var V1 = Integer(1)
