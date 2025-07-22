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

package prompt

import (
	"bytes"
	_ "embed"
	"html/template"
	"os"
)

type Prompt interface {
	String() string
}

type FilePrompt struct {
	Type PromptType         `json:"type"`
	Path string             `json:"path"`
	Data any                `json:"data"`
	tpl  *template.Template `json:"-"`
	file []byte             `json:"-"`
}

type PromptType string

const (
	PromptTypePlainText  PromptType = "text"
	PromptTypeDummy      PromptType = "dummy"
	PromptTypeGoTemplate PromptType = "go-template"
)

func (p FilePrompt) String() string {
	var buf = bytes.NewBuffer(nil)
	err := p.tpl.Execute(buf, p.Data)
	if err != nil {
		panic(err)
	}
	return buf.String()
}

func NewFilePrompt(c *FilePrompt) Prompt {
	switch c.Type {
	case PromptTypePlainText:
		bs, err := os.ReadFile(c.Path)
		if err != nil {
			panic(err)
		}
		c.file = bs
		return c
	case PromptTypeDummy:
		return TextPrompt("")
	case PromptTypeGoTemplate:
		tpl, err := template.ParseFiles(c.Path)
		if err != nil {
			panic(err)
		}
		var buf = bytes.NewBuffer(nil)
		err = tpl.Execute(buf, c.Data)
		if err != nil {
			panic(err)
		}
		c.tpl = tpl
		return c
	default:
		panic("unsupported prompt type " + c.Type)
	}
}

type TextPrompt string

func (p TextPrompt) String() string {
	return string(p)
}

func NewTextPrompt(content string) Prompt {
	return TextPrompt(content)
}

//go:embed analyzer.md
var PromptAnalyzeRepo string
