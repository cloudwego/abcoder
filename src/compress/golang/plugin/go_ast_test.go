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
				homePageDir: "/Users/bytedance/GOPATH/work/hertz/",
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
			// spew.Dump(p)
			f := p.repo.GetFunction(Identity{"github.com/cloudwego/hertz/pkg/route/param", "Params.Get"})
			if out, err := json.MarshalIndent(f, "", "  "); err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			} else {
				println("func:", string(out))
			}
			m := p.repo.GetType(Identity{"github.com/cloudwego/hertz/pkg/route/param", "Params"})
			if out, err := json.MarshalIndent(m, "", "  "); err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			} else {
				println("type:", string(out))
			}
			// out, fun := p.getMain(-1)
			// if fun.Name != "main" {
			// 	t.Fail()
			// }
			// if out, err := json.MarshalIndent(out, "", "  "); err != nil {
			// 	t.Fatalf("json.Marshal() error = %v", err)
			// } else {
			// 	println("size:", len(out), string(out))
			// }
			// if out, err := json.MarshalIndent(fun, "", "  "); err != nil {
			// 	t.Fatalf("json.Marshal() error = %v", err)
			// } else {
			// 	println("size:", len(out), string(out))
			// }
		})
	}
}
