package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/cloudwego/abcoder/lang/golang/parser"
	"github.com/cloudwego/abcoder/lang/uniast"
	"github.com/spf13/cobra"
)

type outgoingProfile struct {
	locateRepo          time.Duration
	locateSymbol        time.Duration
	parseRealtimeSymbol time.Duration
	resolveOutgoing     time.Duration
	loadUniastRefs      time.Duration
	patchUniast         time.Duration
	total               time.Duration
}

type outgoingResult struct {
	nodeType   string
	signature  string
	content    string
	line       int
	deps       []map[string]interface{}
	modPath    string
	pkgPath    string
	symbolKind string
}

func newOutgoingCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "outgoing <file_path> <name>",
		Short:   "Resolve live outgoing for a symbol",
		Long:    "Resolve a symbol from latest source code, return latest code and depth=1 outgoing, optionally load references from uniast and patch uniast incrementally.",
		Example: `abcoder cli outgoing internal/cmd/cli/get_file_symbol.go newGetFileSymbolCmd`,
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			started := time.Now()
			var prof outgoingProfile

			filePath := filepath.Clean(args[0])
			symbolName := args[1]

			locateStarted := time.Now()
			repoRoot, err := os.Getwd()
			if err != nil {
				return err
			}
			prof.locateRepo = time.Since(locateStarted)

			locateSymbolStarted := time.Now()
			live, err := resolveLiveOutgoing(repoRoot, filePath, symbolName)
			if err != nil {
				return err
			}
			prof.locateSymbol = time.Since(locateSymbolStarted)

			prof.parseRealtimeSymbol = live.parseDuration
			prof.resolveOutgoing = live.outgoingDuration

			var refsOnly []map[string]interface{}
			patchStarted := time.Now()
			astsDir, err := getASTsDir(cmd)
			if err == nil {
				repoFile := findRepoFileByCwd(astsDir, repoRoot)
				if repoFile != "" {
					refsStarted := time.Now()
					data, readErr := loadRepoFileData(repoFile)
					if readErr == nil {
						refsOnly, _ = getSymbolReferencesOnly(data, live.modPath, live.pkgPath, symbolName)
					}
					prof.loadUniastRefs = time.Since(refsStarted)
					if readErr == nil {
						patchErr := patchOutgoingToRepoFile(repoFile, data, filePath, symbolName, live)
						if patchErr != nil && verbose {
							fmt.Fprintf(os.Stderr, "[VERBOSE] patch uniast skipped: %v\n", patchErr)
						}
					}
				}
			}
			prof.patchUniast = time.Since(patchStarted)

			node := map[string]interface{}{
				"name":         symbolName,
				"type":         live.nodeType,
				"file":         filePath,
				"line":         live.line,
				"codes":        live.content,
				"signature":    live.signature,
				"dependencies": live.deps,
				"references":   refsOnly,
			}
			resp := map[string]interface{}{"node": node}
			b, _ := json.MarshalIndent(resp, "", "  ")
			fmt.Fprintf(os.Stdout, "%s\n", b)

			prof.total = time.Since(started)
			if verbose {
				fmt.Fprintf(os.Stderr, "[VERBOSE] outgoing.locate_repo=%s\n", prof.locateRepo)
				fmt.Fprintf(os.Stderr, "[VERBOSE] outgoing.locate_symbol=%s\n", prof.locateSymbol)
				fmt.Fprintf(os.Stderr, "[VERBOSE] outgoing.parse_realtime_symbol=%s\n", prof.parseRealtimeSymbol)
				fmt.Fprintf(os.Stderr, "[VERBOSE] outgoing.resolve_outgoing=%s\n", prof.resolveOutgoing)
				fmt.Fprintf(os.Stderr, "[VERBOSE] outgoing.load_uniast_references=%s\n", prof.loadUniastRefs)
				fmt.Fprintf(os.Stderr, "[VERBOSE] outgoing.patch_uniast=%s\n", prof.patchUniast)
				fmt.Fprintf(os.Stderr, "[VERBOSE] outgoing.total=%s\n", prof.total)
			}
			return nil
		},
	}
}

type liveOutgoing struct {
	nodeType        string
	signature       string
	content         string
	line            int
	deps            []map[string]interface{}
	modPath         string
	pkgPath         string
	parseDuration   time.Duration
	outgoingDuration time.Duration
}

func resolveLiveOutgoing(repoRoot, filePath, symbolName string) (*liveOutgoing, error) {
	absPath := filePath
	if !filepath.IsAbs(absPath) {
		absPath = filepath.Join(repoRoot, filePath)
	}

	goParser, err := parser.NewParser(repoRoot, repoRoot, parser.DefaultOptions())
	if err != nil {
		return nil, fmt.Errorf("init go parser: %w", err)
	}

	parseStarted := time.Now()
	id, err := goParser.ParseSymbolInFile(absPath, symbolName)
	if err != nil {
		return nil, err
	}
	modPath := id.ModPath
	pkgPath := id.PkgPath
	if modPath == "" {
		return nil, fmt.Errorf("file not in repo modules: %s", filePath)
	}
	parseDuration := time.Since(parseStarted)

	outgoingStarted := time.Now()
	repo := goParser.GetRepoForCLI()
	node := repo.GetNode(uniast.NewIdentity(modPath, pkgPath, symbolName))
	if node == nil {
		return nil, fmt.Errorf("symbol '%s' not found in file '%s'", symbolName, filePath)
	}
	if node.FileLine().File != filePath {
		return nil, fmt.Errorf("symbol '%s' not found in file '%s'", symbolName, filePath)
	}

	deps := buildGroupedRelations(repo, node.Dependencies)
	outgoingDuration := time.Since(outgoingStarted)

	return &liveOutgoing{
		nodeType:        node.Type.String(),
		signature:       node.Signature(),
		content:         node.Content(),
		line:            node.FileLine().Line,
		deps:            deps,
		modPath:         modPath,
		pkgPath:         pkgPath,
		parseDuration:   parseDuration,
		outgoingDuration: outgoingDuration,
	}, nil
}

func buildGroupedRelations(repo *uniast.Repository, relations []uniast.Relation) []map[string]interface{} {
	depMap := make(map[string][]string)
	for _, rel := range relations {
		n := repo.GetNode(rel.Identity)
		if n == nil {
			continue
		}
		fp := n.FileLine().File
		if fp == "" {
			continue
		}
		depMap[fp] = append(depMap[fp], rel.Identity.Name)
	}
	files := make([]string, 0, len(depMap))
	for fp := range depMap {
		files = append(files, fp)
	}
	sort.Strings(files)
	ret := make([]map[string]interface{}, 0, len(files))
	for _, fp := range files {
		ret = append(ret, map[string]interface{}{
			"file_path": fp,
			"names":     depMap[fp],
		})
	}
	return ret
}
