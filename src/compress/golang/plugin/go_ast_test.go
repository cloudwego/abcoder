package main

import (
	"encoding/json"
	"testing"
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
				homePageDir: "/data00/home/duanyi.aster/Rust/ABCoder/target/debug/tmp/cloudwego/localsession",
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
			pj, err := json.MarshalIndent(p.repo, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			println(string(pj))
			fun, _ := p.GetNode(Identity{"github.com/cloudwego/localsession", "CurSession"})
			// spew.Dump(p)
			if out, err := json.MarshalIndent(fun, "", "  "); err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			} else {
				println("size:", len(out), string(out))
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
				homePageDir: "/data00/home/duanyi.aster/Rust/ABCoder/tmp/cloudwego/kitex",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newGoParser(tt.fields.modName, tt.fields.homePageDir)
			out, fun := p.GetMain(-1)
			if fun.Name != "main" {
				t.Fail()
			}
			if out, err := json.MarshalIndent(out, "", "  "); err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			} else {
				println("size:", len(out), string(out))
			}
			if out, err := json.MarshalIndent(fun, "", "  "); err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			} else {
				println("size:", len(out), string(out))
			}
		})
	}
}
