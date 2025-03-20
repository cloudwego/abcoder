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

package parse

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	. "github.com/cloudwego/abcoder/src/uniast"
)

var (
	referCodeDepth int
	collectComment bool
	excludes       string
)

func init() {
	// init args with flags
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <RepoDir> [id]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.BoolVar(&collectComment, "collect_comment", false, "collect comments for each node")
	flag.IntVar(&referCodeDepth, "refer_code_depth", 0, "the depth to referenced codes, 0 means only return its identity")
	flag.StringVar(&excludes, "excludes", "", "exclude paths, seperated by comma")
}

func Main() {
	var err error
	defer func() {
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
		}
	}()
	flag.Parse()
	as := flag.Args()
	if len(as) < 1 {
		flag.Usage()
		os.Exit(1)
	}

	homeDir := as[0]
	id := ""
	if len(as) >= 2 {
		id = as[1]
	}

	var exs []string
	if excludes != "" {
		exs = strings.Split(excludes, ",")
	}
	p := NewParser(homeDir, homeDir, WithReferCodeDepth(referCodeDepth), WithExcludes(exs))

	var out interface{}

	if id == "" {
		// parse whole repo
		if out, err = p.ParseRepo(); err != nil {
			return
		}
	} else {
		// SPEC: seperate the packagepath and entity name by #
		ids := strings.Split(id, "#")

		if len(ids) == 1 {
			// parse pacakge
			pkgPath := ids[0]
			if out, err = p.ParsePackage(pkgPath); err != nil {
				return
			}
		} else if len(ids) == 2 {
			if out, err = p.ParseNode(ids[0], ids[1]); err != nil {
				return
			}
		}
	}

	buf := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buf)
	encoder.SetEscapeHTML(false)
	err = encoder.Encode(out)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error marshalling functions to JSON:", err)
		os.Exit(1)
	}

	fmt.Println(buf.String())
}

func loadNode(p *goParser, pkgPath string, name string, out *Repository) error {
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
