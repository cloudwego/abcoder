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

package lsp

import (
	"context"
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/cloudwego/abcoder/lang/log"
)

type LSPRequestCache struct {
	cachePath     string
	cacheInterval int
	mu            sync.Mutex
	cache         map[string]map[string]json.RawMessage // method -> params -> result
	cancel        context.CancelFunc
}

func NewLSPRequestCache(path string, interval int) *LSPRequestCache {
	c := &LSPRequestCache{
		cachePath:     path,
		cacheInterval: interval,
		cache:         make(map[string]map[string]json.RawMessage),
	}
	c.Init()
	return c
}

func (c *LSPRequestCache) Init() {
	if c.cachePath == "" {
		return
	}
	if err := c.loadCacheFromDisk(); err != nil {
		log.Error("failed to load LSP cache from disk: %v", err)
	} else {
		log.Info("LSP cache loaded from disk")
	}
	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	go c.PeriodicCacheSaver(ctx)
}

func (c *LSPRequestCache) Close() {
	if c.cancel != nil {
		c.cancel()
	}
}

func (c *LSPRequestCache) saveCacheToDisk() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	data, err := json.Marshal(c.cache)
	if err != nil {
		return err
	}
	return os.WriteFile(c.cachePath, data, 0644)
}

func (c *LSPRequestCache) loadCacheFromDisk() error {
	data, err := os.ReadFile(c.cachePath)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, &c.cache); err != nil {
		return err
	}
	return nil
}

func (cli *LSPRequestCache) PeriodicCacheSaver(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(time.Duration(cli.cacheInterval) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := cli.saveCacheToDisk(); err != nil {
					log.Error("failed to save LSP cache to disk: %v", err)
				} else {
					log.Info("LSP cache saved to disk")
				}
			case <-ctx.Done():
				log.Info("LSP cache saver cancelled, shutting down.")
				return
			}
		}
	}()
}

func (cli *LSPRequestCache) Get(method, params string) (json.RawMessage, bool) {
	cli.mu.Lock()
	defer cli.mu.Unlock()
	if methodCache, ok := cli.cache[method]; ok {
		if result, ok := methodCache[params]; ok {
			return result, true
		}
	}
	return nil, false
}

func (cli *LSPRequestCache) Set(method, params string, result json.RawMessage) {
	cli.mu.Lock()
	defer cli.mu.Unlock()
	methodCache, ok := cli.cache[method]
	if !ok {
		methodCache = make(map[string]json.RawMessage)
		cli.cache[method] = methodCache
	}
	methodCache[params] = result
}
