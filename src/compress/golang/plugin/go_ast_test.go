package main

import (
	"encoding/json"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func Test_goParser_ParseRepo(t *testing.T) {
	type fields struct {
		modName     string
		homePageDir string
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "test",
			fields: fields{
				homePageDir: "/data00/home/duanyi.aster/Rust/ABCoder/tmp/heroiclabs/nakama",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newGoParser(tt.fields.modName, tt.fields.homePageDir)
			err := p.ParseRepo()
			if err != nil {
				t.Fatal(err)
			}
			spew.Dump(p)
			pj, err := json.MarshalIndent(p.repo, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			println(string(pj))
			out, fun, err := p.GetMain(-1)
			if err != nil {
				t.Fatal(err)
			}
			if fun.Name != "main" {
				t.Fail()
			}
			if out, err := json.MarshalIndent(out, "", "  "); err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			} else {
				println("size:", len(out), string(out))
			}
			fun, _, err = p.GetNode(Identity{"github.com/heroiclabs/nakama/console/openapi-gen-angular", "main"})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := json.MarshalIndent(fun, "", "  "); err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			} else {
				// println("size:", len(out), string(out))
			}
		})
	}
}

func Test_goParser_GeMainOnDepends(t *testing.T) {
	type fields struct {
		modName     string
		homePageDir string
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "test",
			fields: fields{
				homePageDir: "/data00/home/duanyi.aster/Rust/ABCoder/tmp/heroiclabs/nakama",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newGoParser(tt.fields.modName, tt.fields.homePageDir)
			_, fun, err := p.GetMain(-1)
			if err != nil {
				t.Fatal(err)
			}
			if fun.Name != "main" {
				t.Fail()
			}
			fun, _, err = p.GetNode(Identity{"github.com/heroiclabs/nakama/console/openapi-gen-angular", "main"})
			if err != nil {
				t.Fatal(err)
			}
			if fun == nil {
				t.Fatal("nil get node")
			}
		})
	}
}
