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

import "encoding/json"

func (r *Repository) SetNode(id Identity) *Node {
	key := id.Full()
	node, ok := r.Graph[key]
	if !ok {
		node = &Node{
			Identity: id,
		}
		r.Graph[key] = node
	}
	if f := r.GetFunction(id); f != nil {
		node.Type = FUNC
	} else if t := r.GetType(id); t != nil {
		node.Type = TYPE
	} else if v := r.GetVar(id); v != nil {
		node.Type = VAR
	} else {
		return node
	}
	node.Repo = r
	return node
}

func (r *Repository) AddRelation(node *Node, dep Identity) {
	node.Dependencies = append(node.Dependencies, Relation{Target: dep, Kind: DEPENDENCY})
	key := dep.Full()
	nd, ok := r.Graph[key]
	if !ok {
		nd = &Node{
			Identity: dep,
		}
		r.Graph[key] = nd
	}
	nd.References = append(nd.References, Relation{
		Target: node.Identity,
		Kind:   REFERENCE,
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

func (r *Repository) BuildGraph() error {
	r.Graph = make(map[string]*Node)
	for _, mod := range r.Modules {
		for _, pkg := range mod.Packages {
			for _, f := range pkg.Functions {
				n := r.SetNode(f.Identity)
				for _, dep := range f.FunctionCalls {
					r.AddRelation(n, dep)
				}
				for _, dep := range f.MethodCalls {
					r.AddRelation(n, dep)
				}
				for _, dep := range f.Types {
					r.AddRelation(n, dep)
				}
				for _, dep := range f.GolobalVars {
					r.AddRelation(n, dep)
				}
			}

			for _, t := range pkg.Types {
				n := r.SetNode(t.Identity)
				for _, dep := range t.SubStruct {
					r.AddRelation(n, dep)
				}
				for _, dep := range t.InlineStruct {
					r.AddRelation(n, dep)
				}
			}

			for _, v := range pkg.Vars {
				n := r.SetNode(v.Identity)
				if v.Type != nil {
					r.AddRelation(n, *v.Type)
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
	Kind   RelationKind
	Target Identity
	Desc   string
}

type marshalerRelation struct {
	// Kind   RelationKind
	Target string
	Desc   string
}

func (r Relation) MarshalJSON() ([]byte, error) {
	rr := marshalerRelation{
		// Kind:   r.Kind,
		Target: r.Target.Full(),
		Desc:   r.Desc,
	}
	return json.Marshal(rr)
}

type NodeType string

const (
	FUNC NodeType = "FUNC"
	TYPE NodeType = "TYPE"
	VAR  NodeType = "VAR"
)

type Node struct {
	Identity     `json:"-"`
	Type         NodeType
	Dependencies []Relation
	References   []Relation
	Repo         *Repository `json:"-"`
}

func (n Node) Content() (string, bool) {
	if n.Repo == nil {
		return "", false
	}
	switch n.Type {
	case FUNC:
		if f := n.Repo.GetFunction(n.Identity); f != nil {
			return f.Content, true
		} else {
			return "", false
		}
	case TYPE:
		if f := n.Repo.GetType(n.Identity); f != nil {
			return f.Content, true
		} else {
			return "", false
		}
	case VAR:
		if f := n.Repo.GetVar(n.Identity); f != nil {
			return f.Content, true
		} else {
			return "", false
		}
	}
	return "", false
}

func (n Node) CompressData() (*string, bool) {
	if n.Repo == nil {
		return nil, false
	}
	switch n.Type {
	case FUNC:
		if f := n.Repo.GetFunction(n.Identity); f != nil {
			return f.CompressData, true
		} else {
			return nil, false
		}
	case TYPE:
		if f := n.Repo.GetType(n.Identity); f != nil {
			return f.CompressData, true
		} else {
			return nil, false
		}
	case VAR:
		if f := n.Repo.GetVar(n.Identity); f != nil {
			return f.CompressData, true
		} else {
			return nil, false
		}
	}
	return nil, false
}
