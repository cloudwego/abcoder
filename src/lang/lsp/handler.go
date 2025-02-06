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
	"container/list"
	"context"
	"sync"
	"time"

	"github.com/cloudwego/abcoder/src/lang/log"
	"github.com/sourcegraph/jsonrpc2"
)

type lspHandler struct {
	notify  chan *jsonrpc2.Request
	mutex   *sync.RWMutex
	history *list.List
	close   chan struct{}
}

func newLSPHandler() *lspHandler {
	ret := &lspHandler{
		notify:  make(chan *jsonrpc2.Request, 1),
		history: list.New(),
		mutex:   &sync.RWMutex{},
		close:   make(chan struct{}),
	}
	// ticker to clean history
	go func() {
		for {
			select {
			case <-time.After(10 * time.Second):
				ret.mutex.Lock()
				total := ret.history.Len() / 2
				if total > 1000 {
					count := 0
					for e := ret.history.Front(); e != nil; e = e.Next() {
						ret.history.Remove(e)
						count++
						if count >= total {
							break
						}
					}
				}
				ret.mutex.Unlock()
			case <-ret.close:
				return
			}
		}
	}()
	return ret
}

func (h *lspHandler) WaitFirstNotify(method string) *jsonrpc2.Request {
	// check history first
loop:
	h.mutex.RLock()
	for e := h.history.Front(); e != nil; e = e.Next() {
		req := e.Value.(*jsonrpc2.Request)
		if req.Method == method {
			h.history.Remove(e)
			h.mutex.RUnlock()
			return req
		}
	}
	h.mutex.RUnlock()
	// wait for notify
	for {
		select {
		case req := <-h.notify:
			if req.Method == method {
				return req
			}
			goto loop
		case <-h.close:
			return nil
		}
	}
}

func (h *lspHandler) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	// This method will be called for both requests and notifications
	log.Info("handle method: %s\n", req.Method)
	if req.Params != nil {
		log.Info("param: %s\n", string(*req.Params))
	}
	if req.Notif {
		// This is a notification
		h.handleNotification(ctx, conn, req)
	} else {
		// This is a request
		h.handleRequest(ctx, conn, req)
	}
}

func (h *lspHandler) handleRequest(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	switch req.Method {
	default:
		log.Error("Received unknown request: %s\n", req.Method)
	}
}

func (h *lspHandler) sendNotify(req *jsonrpc2.Request) {
	// send to channel or save to history
	select {
	case h.notify <- req:
	default:
		h.mutex.Lock()
		h.history.PushBack(req)
		h.mutex.Unlock()
	}
}

func (h *lspHandler) handleNotification(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	switch req.Method {
	// exit
	case "exit":
		log.Info("Received exit notification\n")
		close(h.close)
		conn.Close()
		return
	// Add more cases for other notification types
	default:
		h.sendNotify(req)
	}
}

func (h *lspHandler) Close() {
	close(h.close)
}
