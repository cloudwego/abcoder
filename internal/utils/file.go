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
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudwego/abcoder/llm/log"
	"github.com/fsnotify/fsnotify"
)

func MustWriteFile(fpath string, data []byte) error {
	dir := filepath.Dir(fpath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir %s failed: %v", dir, err)
	}
	if err := os.WriteFile(fpath, data, 0644); err != nil {
		return fmt.Errorf("write file %s failed: %v", fpath, err)
	}
	return nil
}

// use fsnotify to watch the file changes
func WatchDir(dir string, cb func(op fsnotify.Op, file string)) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create watcher failed: %v", err)
	}

	if err := watcher.Add(dir); err != nil {
		return fmt.Errorf("add watch dir %s failed: %v", dir, err)
	}

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					log.Error("invalid watcher event")
					return
				}
				cb(event.Op, event.Name)
			case err, ok := <-watcher.Errors:
				if !ok {
					log.Error("invalid watcher event")
					return
				}
				log.Error("watcher error: %v", err)
			}
		}
	}()

	return nil
}
