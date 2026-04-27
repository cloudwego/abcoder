package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/abcoder/lang/utils"
)

func getSymbolReferencesOnly(data []byte, modPath, pkgPath, symbolName string) ([]map[string]interface{}, error) {
	refs, err := getGraphRelationsByKind(data, modPath, pkgPath, symbolName, "References", "Reference")
	if err != nil {
		return nil, err
	}
	refMap := make(map[string][]string)
	for _, r := range refs {
		filePath := findSymbolFile(data, r["mod_path"], r["pkg_path"], r["name"])
		refMap[filePath] = append(refMap[filePath], r["name"])
	}
	files := make([]string, 0, len(refMap))
	for fp := range refMap {
		files = append(files, fp)
	}
	sort.Strings(files)
	out := make([]map[string]interface{}, 0, len(files))
	for _, fp := range files {
		out = append(out, map[string]interface{}{
			"file_path": fp,
			"names":     refMap[fp],
		})
	}
	return out, nil
}

func getGraphRelationsByKind(data []byte, modPath, pkgPath, symbolName, fieldName, kind string) ([]map[string]string, error) {
	graphKey := modPath + "?" + pkgPath + "#" + symbolName
	graphVal, err := sonic.Get(data, "Graph")
	if err != nil {
		return nil, err
	}
	nodeVal := graphVal.Get(graphKey)
	if !nodeVal.Exists() {
		return nil, nil
	}
	fieldVal := nodeVal.Get(fieldName)
	if !fieldVal.Exists() {
		return nil, nil
	}
	return parseRelationItems(*fieldVal, kind)
}

func patchOutgoingToRepoFile(repoFile string, data []byte, filePath, symbolName string, live *liveOutgoing) error {
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}
	modules, ok := result["Modules"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid Modules")
	}
	mod, ok := modules[live.modPath].(map[string]interface{})
	if !ok {
		return fmt.Errorf("module not found: %s", live.modPath)
	}
	packages, ok := mod["Packages"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid Packages")
	}
	pkg, ok := packages[live.pkgPath].(map[string]interface{})
	if !ok {
		return fmt.Errorf("package not found: %s", live.pkgPath)
	}

	category, sym, err := findSymbolMap(pkg, filePath, symbolName)
	if err != nil {
		return err
	}
	oldDeps := extractGraphDependencyKeys(result, live.modPath, live.pkgPath, symbolName)

	sym["Content"] = live.content
	if live.signature != "" {
		sym["Signature"] = live.signature
	}
	sym["Line"] = live.line

	newDeps, err := toGraphRelations(result, live.deps)
	if err != nil {
		return err
	}
	setSymbolDependencies(category, sym, newDeps)
	setGraphDependencies(result, live.modPath, live.pkgPath, symbolName, newDeps)
	patchNeighborReferences(result, graphKeyOf(live.modPath, live.pkgPath, symbolName), oldDeps, newDeps)

	prettyJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	tmpPath := repoFile + ".tmp"
	if err := utils.MustWriteFile(tmpPath, prettyJSON); err != nil {
		return err
	}
	return os.Rename(tmpPath, repoFile)
}

func findSymbolMap(pkg map[string]interface{}, filePath, symbolName string) (string, map[string]interface{}, error) {
	for _, category := range []string{"Functions", "Types", "Vars"} {
		categoryMap, ok := pkg[category].(map[string]interface{})
		if !ok {
			continue
		}
		val, ok := categoryMap[symbolName].(map[string]interface{})
		if !ok {
			continue
		}
		fp, _ := val["File"].(string)
		if fp == filePath {
			return category, val, nil
		}
	}
	return "", nil, fmt.Errorf("symbol '%s' not found in file '%s'", symbolName, filePath)
}

func extractGraphDependencyKeys(result map[string]interface{}, modPath, pkgPath, symbolName string) map[string]map[string]interface{} {
	keys := map[string]map[string]interface{}{}
	node := ensureGraphNode(result, graphKeyOf(modPath, pkgPath, symbolName))
	deps, _ := node["Dependencies"].([]interface{})
	for _, item := range deps {
		if m, ok := item.(map[string]interface{}); ok {
			keys[relationKey(m)] = m
		}
	}
	return keys
}

func toGraphRelations(result map[string]interface{}, deps []map[string]interface{}) (map[string]map[string]interface{}, error) {
	out := map[string]map[string]interface{}{}
	data := mustMarshal(result)
	for _, group := range deps {
		fp, _ := group["file_path"].(string)
		names, ok := toStringSlice(group["names"])
		if !ok {
			continue
		}
		for _, name := range names {
			modPath, pkgPath, err := findPkgPathByFile(data, fp)
			if err != nil {
				continue
			}
			rel := map[string]interface{}{
				"Kind":    "Dependency",
				"ModPath": modPath,
				"PkgPath": pkgPath,
				"Name":    name,
				"File":    fp,
				"Line":    0,
			}
			out[relationKey(rel)] = rel
		}
	}
	return out, nil
}

func setSymbolDependencies(category string, sym map[string]interface{}, deps map[string]map[string]interface{}) {
	_ = category
	list := make([]interface{}, 0, len(deps))
	keys := make([]string, 0, len(deps))
	for k := range deps {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		list = append(list, deps[k])
	}
	sym["Dependencies"] = list
}

func setGraphDependencies(result map[string]interface{}, modPath, pkgPath, symbolName string, deps map[string]map[string]interface{}) {
	node := ensureGraphNode(result, graphKeyOf(modPath, pkgPath, symbolName))
	list := make([]interface{}, 0, len(deps))
	keys := make([]string, 0, len(deps))
	for k := range deps {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		list = append(list, deps[k])
	}
	node["Dependencies"] = list
}

func patchNeighborReferences(result map[string]interface{}, targetKey string, oldDeps, newDeps map[string]map[string]interface{}) {
	for key, dep := range newDeps {
		if _, ok := oldDeps[key]; ok {
			continue
		}
		node := ensureGraphNode(result, graphKeyOf(dep["ModPath"].(string), dep["PkgPath"].(string), dep["Name"].(string)))
		refs := relationSliceToMap(node["References"])
		refs[targetKey] = graphKeyToRelation(targetKey)
		node["References"] = relationMapToSlice(refs)
	}
	for key, dep := range oldDeps {
		if _, ok := newDeps[key]; ok {
			continue
		}
		node := ensureGraphNode(result, graphKeyOf(dep["ModPath"].(string), dep["PkgPath"].(string), dep["Name"].(string)))
		refs := relationSliceToMap(node["References"])
		delete(refs, targetKey)
		node["References"] = relationMapToSlice(refs)
	}
}

func ensureGraphNode(result map[string]interface{}, graphKey string) map[string]interface{} {
	graph, ok := result["Graph"].(map[string]interface{})
	if !ok {
		graph = map[string]interface{}{}
		result["Graph"] = graph
	}
	node, ok := graph[graphKey].(map[string]interface{})
	if !ok {
		node = map[string]interface{}{}
		graph[graphKey] = node
	}
	return node
}

func graphKeyOf(modPath, pkgPath, name string) string {
	return modPath + "?" + pkgPath + "#" + name
}

func relationKey(m map[string]interface{}) string {
	return graphKeyOf(asString(m["ModPath"]), asString(m["PkgPath"]), asString(m["Name"]))
}

func relationSliceToMap(val interface{}) map[string]map[string]interface{} {
	out := map[string]map[string]interface{}{}
	arr, _ := val.([]interface{})
	for _, item := range arr {
		if m, ok := item.(map[string]interface{}); ok {
			out[relationKey(m)] = m
		}
	}
	return out
}

func relationMapToSlice(m map[string]map[string]interface{}) []interface{} {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]interface{}, 0, len(keys))
	for _, k := range keys {
		out = append(out, m[k])
	}
	return out
}

func graphKeyToRelation(key string) map[string]interface{} {
	var modPath, pkgPath, name string
	for i := 0; i < len(key); i++ {
		if key[i] == '?' {
			modPath = key[:i]
			rest := key[i+1:]
			for j := 0; j < len(rest); j++ {
				if rest[j] == '#' {
					pkgPath = rest[:j]
					name = rest[j+1:]
					break
				}
			}
			break
		}
	}
	return map[string]interface{}{
		"Kind":    "Dependency",
		"ModPath": modPath,
		"PkgPath": pkgPath,
		"Name":    name,
		"Line":    0,
	}
}

func asString(v interface{}) string {
	s, _ := v.(string)
	return s
}

func toStringSlice(v interface{}) ([]string, bool) {
	s, ok := v.([]string)
	if ok {
		return s, true
	}
	arr, ok := v.([]interface{})
	if !ok {
		return nil, false
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		str, ok := item.(string)
		if !ok {
			return nil, false
		}
		out = append(out, str)
	}
	return out, true
}

func mustMarshal(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}
