/**
 * Copyright 2025 ByteDance Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package utils

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
)

func TestWatchDir(t *testing.T) {
	t.Skip()
	type args struct {
		dir string
		cb  func(op fsnotify.Op, files string, state *bool)
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			args: args{
				dir: "../../tmp",
				cb: func(op fsnotify.Op, files string, state *bool) {
					*state = true
					t.Logf("watch op %v file %s", op, files)
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var x bool
			if err := WatchDir(tt.args.dir, func(op fsnotify.Op, files string) {
				tt.args.cb(op, files, &x)
			}); (err != nil) != tt.wantErr {
				t.Errorf("WatchDir() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err := os.WriteFile(filepath.Join(tt.args.dir, "test.txtx"), []byte("test"), 0644); err != nil {
				t.Errorf("WriteFile() error = %v, wantErr %v", err, tt.wantErr)
			}
			time.Sleep(time.Second)
			if !x {
				t.Errorf("watch file failed")
			}
			if err := os.Remove(filepath.Join(tt.args.dir, "test.txtx")); err != nil {
				t.Errorf("Remove() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
