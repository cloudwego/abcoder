package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

//---------------- Golang Parser -----------------

// golang parser, used parse multle packages from the entire project
type goParser struct {
	modName               string
	homePageDir           string
	visited               map[string]bool
	processedPkgFunctions map[PkgPath]map[string]*Function
	processedPkgStruct    map[PkgPath]map[string]*Struct
}

// newGoParser
func newGoParser(modName, homePageDir string) *goParser {
	abs, err := filepath.Abs(homePageDir)
	if err != nil {
		panic(fmt.Sprintf("cannot get absolute path form homePageDir:%v", err))
	}

	p := &goParser{
		modName:               modName,
		homePageDir:           abs,
		processedPkgFunctions: map[PkgPath]map[string]*Function{},
		processedPkgStruct:    map[PkgPath]map[string]*Struct{},
		visited:               map[string]bool{},
	}
	if p.modName == "" {
		var err error
		p.modName, err = getModuleName(p.homePageDir + "/go.mod")
		if err != nil {
			panic(err.Error())
		}
	}
	return p
}

// ToABS converts a local package path to absolute path
// If the path is not a local package, return empty string
func (p *goParser) pkgPathToABS(path PkgPath) string {
	if !strings.HasPrefix(string(path), p.modName) {
		return ""
	} else {
		return filepath.Join(p.homePageDir, strings.TrimPrefix(string(path), p.modName))
	}
}

// FromABS converts an absolute path to local mod path
func (p *goParser) pkgPathFromABS(path string) PkgPath {
	if rel, _ := filepath.Rel(p.homePageDir, path); rel == "" {
		return ""
	} else {
		return filepath.Join(p.modName, rel)
	}
}

func getGoFilesInDir(dir string) []string {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}

	goFiles := make([]string, 0)
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".go") {
			goFiles = append(goFiles, filepath.Join(dir, file.Name()))
		}
	}
	return goFiles
}

func (p *goParser) associateStructWithMethods() {
	for _, fs := range p.processedPkgFunctions {
		for _, f := range fs {
			if f.IsMethod && f.AssociatedStruct != nil {
				// entrue the Struct has been visted
				if f.AssociatedStruct.FilePath != "" {
					if f.AssociatedStruct.Methods == nil {
						f.AssociatedStruct.Methods = map[string]*Function{}
					}
					f.AssociatedStruct.Methods[f.Name] = f
				}
			}
		}
	}
}

// TODO: Parallel transformation
// ParseTilTheEnd parse the all go files from the starDir,
// and their related go files in the project recursively
func (p *goParser) ParseTilTheEnd(startDir string) error {
	if err := p.ParseDir(startDir); err != nil {
		return err
	}
	for path, pkg := range p.processedPkgFunctions {
		// ignore third-party packages
		if !strings.Contains(path, p.modName) {
			continue
		}
		for _, f := range pkg {
			// Notice: local funcs has been parsed in ParseDir
			for _, fc := range f.InternalFunctionCalls {
				if p.visited[fc.PkgPath] {
					continue
				}
				if err := p.ParseTilTheEnd(p.pkgPathToABS(fc.PkgPath)); err != nil {
					return err
				}
			}
			for _, fc := range f.InternalMethodCalls {
				if p.visited[fc.PkgPath] {
					continue
				}
				if err := p.ParseTilTheEnd(p.pkgPathToABS(fc.PkgPath)); err != nil {
					return err
				}
			}
		}
	}

	p.associateStructWithMethods()
	return nil
}

type MainStream struct {
	MainFunc string

	RelatedFunctions []SingleFunction

	RelatedStruct []SingleStruct
}

func (m *MainStream) Dedup() {
	fs := map[string]string{}
	for _, f := range m.RelatedFunctions {
		fs[f.CallName] = f.Content
	}
	m.RelatedFunctions = m.RelatedFunctions[:len(fs)]
	i := 0
	for k, v := range fs {
		m.RelatedFunctions[i].CallName = k
		m.RelatedFunctions[i].Content = v
		i++
	}

	fs = map[string]string{}
	for _, f := range m.RelatedStruct {
		fs[f.Name] = f.Content
	}
	m.RelatedStruct = m.RelatedStruct[:len(fs)]
	i = 0
	for k, v := range fs {
		m.RelatedStruct[i].Name = k
		m.RelatedStruct[i].Content = v
		i++
	}
}

type SingleFunction struct {
	CallName string
	Content  string
}

type SingleStruct struct {
	Name    string
	Content string
}

func (p *goParser) getMain(depth int) (*MainStream, *Function) {
	m := &MainStream{
		RelatedFunctions: make([]SingleFunction, 0),
	}

	var mainFunc *Function

	for _, v := range p.processedPkgFunctions {
		for _, vv := range v {
			if vv.Name == "main" {
				mainFunc = vv
				m.MainFunc = vv.Content
				break
			}
		}
	}
	visited := map[string]map[string]bool{}
	p.fillRelatedContent(depth, mainFunc, &m.RelatedFunctions, &m.RelatedStruct, visited)
	return m, mainFunc
}

func (p *goParser) fillRelatedContent(depth int, f *Function, fl *[]SingleFunction, sl *[]SingleStruct, visited map[string]map[string]bool) {
	if depth == 0 {
		return
	}
	if f == nil || (visited[f.PkgPath] != nil && visited[f.PkgPath][f.Name]) {
		return
	} else {
		if visited[f.PkgPath] == nil {
			visited[f.PkgPath] = map[string]bool{}
		}
		visited[f.PkgPath][f.Name] = true
	}
	for call, ff := range f.InternalFunctionCalls {
		s := SingleFunction{
			CallName: call,
			Content:  ff.Content,
		}
		*fl = append(*fl, s)
		p.fillRelatedContent(depth-1, ff, fl, sl, visited)
	}

	for call, ff := range f.InternalMethodCalls {
		content := ff.Content
		if ff.AssociatedStruct != nil && ff.AssociatedStruct.IsInterface {
			content = ff.AssociatedStruct.Content
		}
		s := SingleFunction{
			CallName: call,
			Content:  content,
		}
		*fl = append(*fl, s)
		p.fillRelatedContent(depth-1, ff, fl, sl, visited)

		// for method which has been associated with struct, push the struct
		if ff.AssociatedStruct != nil && ff.AssociatedStruct.Content != "" {
			st := ff.AssociatedStruct
			if visited[st.PkgPath] != nil && visited[st.PkgPath][st.Name] {
				continue
			} else if visited[st.PkgPath] == nil {
				visited[st.PkgPath] = map[string]bool{}
			}
			visited[st.PkgPath][st.Name] = true
			ss := SingleStruct{
				Name:    ff.PkgPath + "." + st.Name,
				Content: st.Content,
			}
			*sl = append(*sl, ss)
		}

		p.fillRelatedContent(depth-1, ff, fl, sl, visited)
	}
}

func getModuleName(modFilePath string) (string, error) {
	file, err := os.Open(modFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "module") {
			// Assuming 'module' keyword is followed by module name
			parts := strings.Split(line, " ")
			if len(parts) > 1 {
				return parts[1], nil
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("failed to scan file: %v", err)
	}

	return "", nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Missing home dir of the project to parse.")
		os.Exit(1)
	}
	//

	homeDir := os.Args[1]

	p := newGoParser("", homeDir)
	if err := p.ParseTilTheEnd(p.homePageDir); err != nil {
		fmt.Println("Error parsing go files:", err)
		os.Exit(1)
	}

	// p.generateStruct()
	m, _ := p.getMain(100)
	m.Dedup()

	out := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(out)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(m)
	if err != nil {
		fmt.Println("Error marshalling functions to JSON:", err)
		os.Exit(1)
	}

	fmt.Println(out.String())
}
