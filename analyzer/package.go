package analyzer

import (
	"context"
	"errors"
	"fmt"
	"github.com/viant/afs/storage"
	"github.com/viant/afs/url"
	"github.com/viant/linager/analyzer/linage"
	"io"
	"os"
	"path/filepath"
)

// AnalyzeDir walks a directory tree, detects project roots (e.g. go.mod, pom.xml, package.json),
// and analyses each package found under those roots.
func (a *Analyzer) AnalyzeDir(ctx context.Context, root string) ([]*linage.PackageModel, error) {
	// if project file markers are configured, detect project/module roots
	if len(a.projectFiles) > 0 {
		roots := map[string]bool{}
		visitor := func(ctx context.Context, baseURL, parent string, info os.FileInfo, reader io.Reader) (bool, error) {
			if info.IsDir() {
				return true, nil
			}
			for _, marker := range a.projectFiles {
				if info.Name() == marker {
					roots[url.Join(baseURL, parent)] = true
					break
				}
			}
			return true, nil
		}
		if err := a.fs.Walk(ctx, root, visitor); err != nil {
			return nil, err
		}
		if len(roots) == 0 {
			roots[root] = true
		}
		var all []*linage.PackageModel
		for projectRoot := range roots {
			models, err := a.analyzePackages(ctx, projectRoot)
			if err != nil {
				return nil, err
			}
			all = append(all, models...)
		}
		return all, nil
	}
	// fallback to scan all packages under root
	return a.analyzePackages(ctx, root)
}

// analyzePackages walks a directory tree under root and analyses each package.
func (a *Analyzer) analyzePackages(ctx context.Context, root string) ([]*linage.PackageModel, error) {
	assets := map[string][]string{}
	var visitor storage.OnVisit = func(ctx context.Context, baseURL, parent string, info os.FileInfo, reader io.Reader) (bool, error) {
		if !a.match(info) {
			return false, nil
		}
		if info.IsDir() {
			return true, nil
		}
		pkg := url.Join(baseURL, parent)
		assets[pkg] = append(assets[pkg], info.Name())
		return true, nil
	}
	if err := a.fs.Walk(ctx, root, visitor); err != nil {
		return nil, err
	}
	var models []*linage.PackageModel
	for pkgURL, files := range assets {
		m, err := a.analyzePackage(ctx, pkgURL, files)
		if err != nil {
			return nil, err
		}
		models = append(models, m)
	}
	return models, nil
}

func (a *Analyzer) analyzePackage(ctx context.Context, baseURL string, files []string) (*linage.PackageModel, error) {
	model := &linage.PackageModel{Path: baseURL, Language: a.Language, Files: files, Idents: map[string]*linage.Identifier{}}

	pkgScope := &linage.Scope{ID: baseURL, Kind: "package", Symbols: map[string]*linage.Identifier{}}
	model.Scopes = append(model.Scopes, pkgScope)

	for _, file := range files {
		URL := url.Join(baseURL, file)
		code, err := a.fs.DownloadWithURL(ctx, URL)
		if err != nil {
			return nil, err
		}
		a.AnalyzeSourceCode(baseURL, code, URL, pkgScope, model)
	}

	a.computeTransitiveClosure(model)
	return model, nil
}

func (a *Analyzer) AnalyzeSourceCode(dir string, code []byte, filePath string, pkgScope *linage.Scope, model *linage.PackageModel) error {
	// reset import aliases for this file
	a.importAliases = map[string]string{}
	// record package path and files
	if model.Path == "" {
		model.Path = dir
	}
	// track this source file
	model.Files = append(model.Files, filepath.Base(filePath))
	// parse AST
	tree := a.parser.Parse(nil, code)
	if tree == nil {
		return errors.New("failed to parse code")
	}
	rootNode := tree.RootNode()
	fileScope := &linage.Scope{ID: fmt.Sprintf("%s:%s", dir, filepath.Base(filePath)), Kind: "file", Parent: pkgScope, Symbols: map[string]*linage.Identifier{}, Start: int(rootNode.StartByte()), End: int(rootNode.EndByte())}
	pkgScope.Symbols[filepath.Base(filePath)] = &linage.Identifier{ID: fileScope.ID, Kind: "file", Name: filepath.Base(filePath), Package: dir, File: filePath, StartByte: rootNode.StartByte(), Node: rootNode}
	model.Scopes = append(model.Scopes, fileScope)
	a.walk(rootNode, code, fileScope, model)
	return nil
}

// AnalyzeAll runs analysis over all detected project roots under the given directory
// and merges their PackageModels into a single global model.
func (a *Analyzer) AnalyzeAll(ctx context.Context, root string) (*linage.PackageModel, error) {
	models, err := a.AnalyzeDir(ctx, root)
	if err != nil {
		return nil, err
	}
	merged := linage.Merge(models...)
	// set language for the merged model
	merged.Language = a.Language
	// export intermediate representation graph if configured
	if a.graphExporter != nil {
		graph := buildIRGraph(a, merged)
		if err := a.graphExporter.Export(graph); err != nil {
			return nil, err
		}
	}
	return merged, nil
}
