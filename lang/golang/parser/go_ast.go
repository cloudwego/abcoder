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

package parser

import (
	"fmt"

	. "github.com/cloudwego/abcoder/lang/uniast"
)

func loadNode(p *GoParser, pkgPath string, name string, out *Repository) error {
	mod, _ := p.getModuleFromPkg(pkgPath)
	np, err := p.getNode(NewIdentity(mod, PkgPath(pkgPath), name))
	if err != nil {
		return fmt.Errorf("error getting node: %v", err)
	}
	repo := p.getRepo()
	if out.Modules[mod] == nil {
		out.Modules[mod] = newModule(repo.Modules[mod].Name, repo.Modules[mod].Dir)
	}
	if out.Modules[mod].Packages[pkgPath] == nil {
		out.Modules[mod].Packages[pkgPath] = NewPackage(pkgPath)
	}
	if fp, ok := np.(*Function); ok {
		out.Modules[mod].Packages[pkgPath].Functions[name] = fp
	}
	if sp, ok := np.(*Type); ok {
		out.Modules[mod].Packages[pkgPath].Types[name] = sp
	}
	if vp, ok := np.(*Var); ok {
		out.Modules[mod].Packages[pkgPath].Vars[name] = vp
	}
	return nil
}
