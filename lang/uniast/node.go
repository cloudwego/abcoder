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

package uniast

import (
	"strconv"
	"strings"
)

func (r *Repository) GetNode(id Identity) *Node {
	key := id.Full()
	node, ok := r.Graph[key]
	if !ok {
		return nil
	}
	return node
}

func (r *Repository) GetPackage(mod ModPath, pkg PkgPath) *Package {
	m, ok := r.Modules[mod]
	if !ok {
		return nil
	}
	return m.Packages[pkg]
}

func (r *Repository) GetModule(mod ModPath) *Module {
	m, ok := r.Modules[mod]
	if !ok {
		return nil
	}
	return m
}

// NOTICE: if entity not exist, only set the node on graph
func (r *Repository) SetNode(id Identity, typ NodeType) *Node {
	key := id.Full()
	node, ok := r.Graph[key]
	if !ok {
		node = &Node{
			Identity: id,
			Type:     typ,
			Repo:     r,
		}
		r.Graph[key] = node
	}
	node.Repo = r
	switch typ {
	case FUNC:
		if f := r.GetFunction(id); f == nil {
			r.SetFunction(id, &Function{
				Identity: id,
			})
		}
	case TYPE:
		if t := r.GetType(id); t == nil {
			r.SetType(id, &Type{
				Identity: id,
			})
		}
	case VAR:
		if v := r.GetVar(id); v == nil {
			r.SetVar(id, &Var{
				Identity: id,
			})
		}
	}
	return node
}

func calOffset(ref, dep FileLine) int {
	refLine := dep.Line - ref.Line
	if refLine < 0 {
		return -1
	}
	return refLine
}

func (r *Repository) AddRelation(node *Node, dep Identity, depFl FileLine) {
	line := calOffset(node.FileLine(), depFl)
	node.Dependencies = append(node.Dependencies, Relation{
		Identity: dep,
		Kind:     DEPENDENCY,
		Line:     line,
	})
	key := dep.Full()
	nd, ok := r.Graph[key]
	if !ok {
		nd = &Node{
			Identity: dep,
			Repo:     r,
		}
		r.Graph[key] = nd
	}
	nd.References = append(nd.References, Relation{
		Identity: node.Identity,
		Kind:     REFERENCE,
		Line:     line,
	})
	if f := r.GetFunction(dep); f != nil {
		nd.Type = FUNC
	} else if t := r.GetType(dep); t != nil {
		nd.Type = TYPE
	} else if v := r.GetVar(dep); v != nil {
		nd.Type = VAR
	} else {
		return
	}
	nd.Repo = r
}

func (r *Repository) AllNodesSetRepo() {
	for _, node := range r.Graph {
		node.Repo = r
	}
}

func (r *Repository) BuildGraph() error {
	r.Graph = make(map[string]*Node)
	for _, mod := range r.Modules {
		if mod.IsExternal() {
			continue
		}
		for _, pkg := range mod.Packages {
			for _, f := range pkg.Functions {
				n := r.SetNode(f.Identity, FUNC)
				for _, dep := range f.FunctionCalls {
					r.AddRelation(n, dep.Identity, dep.FileLine)
				}
				for _, dep := range f.MethodCalls {
					r.AddRelation(n, dep.Identity, dep.FileLine)
				}
				for _, dep := range f.Types {
					r.AddRelation(n, dep.Identity, dep.FileLine)
				}
				for _, dep := range f.GlobalVars {
					r.AddRelation(n, dep.Identity, dep.FileLine)
				}
			}

			for _, t := range pkg.Types {
				n := r.SetNode(t.Identity, TYPE)
				for _, dep := range t.SubStruct {
					r.AddRelation(n, dep.Identity, dep.FileLine)
				}
				for _, dep := range t.InlineStruct {
					r.AddRelation(n, dep.Identity, dep.FileLine)
				}
			}

			for _, v := range pkg.Vars {
				n := r.SetNode(v.Identity, VAR)
				if v.Type != nil {
					r.AddRelation(n, *v.Type, v.FileLine)
				}
			}
		}
	}
	return nil
}

type RelationKind string

const (
	DEPENDENCY RelationKind = "Dependency"
	REFERENCE  RelationKind = "Reference"
)

type Relation struct {
	Kind     RelationKind
	Identity // target node
	Line     int
	Desc     *string `json:",omitempty"`
	Codes    *string `json:",omitempty"`
}

// type marshalerRelation struct {
// 	// Kind   RelationKind
// 	Target string
// 	Desc   string
// }

// func (r Relation) MarshalJSON() ([]byte, error) {
// 	rr := marshalerRelation{
// 		// Kind:   r.Kind,
// 		Target: r.Target.Full(),
// 		Desc:   r.Desc,
// 	}
// 	return json.Marshal(rr)
// }

// Node 类型
type NodeType int

const (
	UNKNOWN NodeType = iota
	FUNC
	TYPE
	VAR
)

func (t NodeType) String() string {
	switch t {
	case FUNC:
		return "FUNC"
	case TYPE:
		return "TYPE"
	case VAR:
		return "VAR"
	default:
		return "UNKNOWN"
	}
}

func (t NodeType) MarshalJSON() ([]byte, error) {
	return []byte(`"` + t.String() + `"`), nil
}

func (t *NodeType) UnmarshalJSON(b []byte) error {
	v, err := strconv.Unquote(string(b))
	if err != nil {
		return err
	}
	typ := NewNodeType(v)
	*t = typ
	return nil
}

func NewNodeType(typ string) NodeType {
	switch strings.ToLower(typ) {
	case "func", "function":
		return FUNC
	case "type", "struct":
		return TYPE
	case "var", "variable", "variant", "const":
		return VAR
	default:
		return UNKNOWN
	}
}

type Node struct {
	Identity
	Type         NodeType
	Dependencies []Relation
	References   []Relation
	Repo         *Repository `json:"-"`
}

func (n Node) GetDependency(id Identity) *Relation {
	for i, dep := range n.Dependencies {
		if dep.Identity == id {
			return &n.Dependencies[i]
		}
	}
	return nil
}

func NewNode(id Identity, typ NodeType, repo *Repository) *Node {
	return &Node{
		Identity: id,
		Type:     typ,
		Repo:     repo,
	}
}

func (n Node) SetCompressData(data string) bool {
	if n.Repo == nil {
		return false
	}
	switch n.Type {
	case FUNC:
		if f := n.Repo.GetFunction(n.Identity); f != nil {
			f.CompressData = &data
			return true
		}
	case TYPE:
		if f := n.Repo.GetType(n.Identity); f != nil {
			f.CompressData = &data
			return true
		}
	case VAR:
		if f := n.Repo.GetVar(n.Identity); f != nil {
			f.CompressData = &data
			return true
		}
	}

	return false
}

func (n Node) SetContent(content string) bool {
	if n.Repo == nil {
		return false
	}
	switch n.Type {
	case FUNC:
		if f := n.Repo.GetFunction(n.Identity); f != nil {
			f.Content = content
			return true
		} else {
			return false
		}
	case TYPE:
		if f := n.Repo.GetType(n.Identity); f != nil {
			f.Content = content
			return true
		}
		return false
	case VAR:
		if f := n.Repo.GetVar(n.Identity); f != nil {
			f.Content = content
			return true
		}
		return false
	default:
		return false
	}
}

func (n Node) Content() string {
	if n.Repo == nil {
		return ""
	}
	switch n.Type {
	case FUNC:
		if f := n.Repo.GetFunction(n.Identity); f != nil {
			return f.Content
		} else {
			return ""
		}
	case TYPE:
		if f := n.Repo.GetType(n.Identity); f != nil {
			return f.Content
		} else {
			return ""
		}
	case VAR:
		if f := n.Repo.GetVar(n.Identity); f != nil {
			return f.Content
		} else {
			return ""
		}
	}
	return ""
}

func (n Node) CompressData() *string {
	if n.Repo == nil {
		return nil
	}
	switch n.Type {
	case FUNC:
		if f := n.Repo.GetFunction(n.Identity); f != nil {
			return f.CompressData
		} else {
			return nil
		}
	case TYPE:
		if f := n.Repo.GetType(n.Identity); f != nil {
			return f.CompressData
		} else {
			return nil
		}
	case VAR:
		if f := n.Repo.GetVar(n.Identity); f != nil {
			return f.CompressData
		} else {
			return nil
		}
	}
	return nil
}

func (n Node) FileLine() FileLine {
	if n.Repo == nil {
		return FileLine{}
	}
	switch n.Type {
	case FUNC:
		if f := n.Repo.GetFunction(n.Identity); f != nil {
			return f.FileLine
		} else {
			return FileLine{}
		}
	case TYPE:
		if f := n.Repo.GetType(n.Identity); f != nil {
			return f.FileLine
		}
		return FileLine{}
	case VAR:
		if f := n.Repo.GetVar(n.Identity); f != nil {
			return f.FileLine
		}
		return FileLine{}
	default:
		return FileLine{}
	}
}

func (n Node) SetFileLine(file FileLine) {
	if n.Repo == nil {
		return
	}
	switch n.Type {
	case FUNC:
		if f := n.Repo.GetFunction(n.Identity); f != nil {
			f.FileLine = file
		}
	case TYPE:
		if f := n.Repo.GetType(n.Identity); f != nil {
			f.FileLine = file

		}
	case VAR:
		if f := n.Repo.GetVar(n.Identity); f != nil {
			f.FileLine = file
		}
	default:
		return
	}
}

func (n Node) SetIsExported(isExported bool) {
	if n.Repo == nil {
		return
	}
	switch n.Type {
	case FUNC:
		if f := n.Repo.GetFunction(n.Identity); f != nil {
			f.Exported = isExported
		}
	case TYPE:
		if f := n.Repo.GetType(n.Identity); f != nil {
			f.Exported = isExported
		}
	case VAR:
		if f := n.Repo.GetVar(n.Identity); f != nil {
			f.IsExported = isExported
		}
	}
}

func (n Node) IsExported() bool {
	if n.Repo == nil {
		return false
	}
	switch n.Type {
	case FUNC:
		if f := n.Repo.GetFunction(n.Identity); f != nil {
			return f.Exported
		}
	case TYPE:
		if f := n.Repo.GetType(n.Identity); f != nil {
			return f.Exported
		}
	case VAR:
		if f := n.Repo.GetVar(n.Identity); f != nil {
			return f.IsExported
		}
	}
	return false
}

func (n Node) SetIsMethod(isMethod bool) {
	if n.Repo == nil {
		return
	}
	switch n.Type {
	case FUNC:
		if f := n.Repo.GetFunction(n.Identity); f != nil {
			f.IsMethod = isMethod
		}
	}
}

func (n Node) Module() *Module {
	if n.Repo == nil {
		return nil
	}
	return n.Repo.Modules[n.Identity.ModPath]
}
