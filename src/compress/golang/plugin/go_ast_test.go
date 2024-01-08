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
				homePageDir: "../../../../testdata/golang",
			},
			args: args{
				startDir: "./cmd",
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
			spew.Dump(p)
			out, _ := p.getMain()
			if out, err := json.MarshalIndent(out, "", "  "); err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			} else {
				println(string(out))
			}
		})
	}
}
