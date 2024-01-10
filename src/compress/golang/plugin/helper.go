/**
 * Copyright 2024 ByteDance Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

type cache map[interface{}]bool

func (c cache) Visited(val interface{}) bool {
	ok := c[val]
	if !ok {
		c[val] = true
	}
	return ok
}

// RemoveCycle remove cycle reference to repeated pointer, in case of marshalling fail
func (f *Function) RemoveCycle() {
	var visited = cache(map[interface{}]bool{})
	f.deLoop(visited)
	if f.AssociatedStruct != nil {
		f.AssociatedStruct.deLoop(visited)
	}
}

func (f *Function) deLoop(visited cache) {
	for k, ff := range f.InternalFunctionCalls {
		if visited.Visited(ff) {
			var nf = new(Function)
			nf.PkgPath = ff.PkgPath
			nf.Name = ff.Name
			f.InternalFunctionCalls[k] = nf
		} else {
			ff.deLoop(visited)
		}
	}
	for k, ff := range f.InternalMethodCalls {
		if visited.Visited(ff) {
			var nf = new(Function)
			nf.PkgPath = ff.PkgPath
			nf.Name = ff.Name
			nf.IsMethod = ff.IsMethod
			f.InternalMethodCalls[k] = nf
		} else {
			ff.deLoop(visited)
			if ff.AssociatedStruct != nil {
				ff.AssociatedStruct.deLoop(visited)
			}
		}
	}
}

func (f *Struct) deLoop(visited cache) {
	for k, ff := range f.InternalStructs {
		if visited.Visited(ff) {
			var nf = new(Struct)
			nf.PkgPath = ff.PkgPath
			nf.Name = ff.Name
			f.InternalStructs[k] = nf
		} else {
			ff.deLoop(visited)
		}
	}
	for k, ff := range f.Methods {
		if visited.Visited(ff) {
			var nf = new(Function)
			nf.PkgPath = ff.PkgPath
			nf.Name = ff.Name
			nf.IsMethod = ff.IsMethod
			f.Methods[k] = nf
		} else {
			ff.deLoop(visited)
			if ff.AssociatedStruct != nil {
				ff.AssociatedStruct.deLoop(visited)
			}
		}
	}
}
