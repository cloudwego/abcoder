package main

import (
	"encoding/json"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func Test_goParser_ParseTilTheEnd(t *testing.T) {
	type fields struct {
		modName     string
		homePageDir string
	}
	type args struct {
		startDir string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "test",
			fields: fields{
				homePageDir: "/Users/bytedance/GOPATH/work/hertz",
			},
			args: args{
				startDir: "cmd/hz",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newGoParser(tt.fields.modName, tt.fields.homePageDir)
			err := p.ParseTilTheEnd(tt.args.startDir)
			if err != nil {
				t.Fatalf("goParser.ParseTilTheEnd() error = %v", err)
			}
			// spew.Dump(p)
			out, fun := p.getMain(2)
			if fun.Name != "main" {
				t.Fail()
			}
			if out, err := json.MarshalIndent(fun, "", "  "); err != nil {
				t.Fatal(err)
			} else {
				println("func size:", len(out), string(out))
			}
			if out, err := json.MarshalIndent(out, "", "  "); err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			} else {
				println("size:", len(out), string(out))
			}
		})
	}
}

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
				homePageDir: "../../../../testdata/golang",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newGoParser(tt.fields.modName, tt.fields.homePageDir)
			err := p.ParseRepo()
			if err != nil {
				t.Fatalf("goParser.ParseTilTheEnd() error = %v", err)
			}
			spew.Dump(p)
			x := p.repo.GetType(Identity{"a.b/c/pkg/entity", "MyStruct"})
			spew.Dump(x.InlineStruct, x.SubStruct)
			out, fun := p.getMain(2)
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
