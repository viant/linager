package analyzer

import (
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/viant/linager/analyzer/linage"
	"os"
	"path/filepath"
	"strings"
)

type Option func(*Analyzer)

// AnalyzerPlugin defines extension hooks for analyzer passes.
// BeforeWalk is called for each AST node before default processing.
// AfterResolveIdent is called after an identifier is resolved.
type AnalyzerPlugin interface {
	BeforeWalk(n *sitter.Node, src []byte, scope *linage.Scope, model *linage.PackageModel)
	AfterResolveIdent(n *sitter.Node, id *linage.Identifier, scope *linage.Scope, model *linage.PackageModel)
}

// AnnotationHook is a callback invoked when an identifier with annotations is found.
// It can be used to add custom data-flow edges based on code metadata (e.g., tags, annotations).
type AnnotationHook func(id *linage.Identifier, anns linage.Annotations, scope *linage.Scope, model *linage.PackageModel)

func WithLanguage(language *sitter.Language) Option {
	return func(a *Analyzer) {
		a.parser.SetLanguage(language)
	}
}

// WithLanguageName sets a language tag (e.g., "go", "java") for normalization across services
func WithLanguageName(name string) Option {
	return func(a *Analyzer) {
		a.Language = name
	}
}

func WithMacher(matcher MatcherFn) Option {
	return func(a *Analyzer) {
		a.match = matcher
	}
}

// WithProjectFiles sets filenames (e.g. go.mod, pom.xml, package.json) used to detect project roots
func WithProjectFiles(files ...string) Option {
	return func(a *Analyzer) {
		a.projectFiles = files
	}
}

// WithAnnotationHook registers a hook to process annotations on identifiers.
func WithAnnotationHook(hook AnnotationHook) Option {
	return func(a *Analyzer) {
		a.annotationHooks = append(a.annotationHooks, hook)
	}
}

// WithPlugin registers an AnalyzerPlugin for extended analysis.
func WithPlugin(p AnalyzerPlugin) Option {
	return func(a *Analyzer) {
		a.plugins = append(a.plugins, p)
	}
}

// WithInterprocedural enables inter-procedural call-return analysis (linking actual args to formals and returns to call sites).
func WithInterprocedural() Option {
	return func(a *Analyzer) {
		a.interprocedural = true
	}
}

func GolangFiles(info os.FileInfo) bool {
	if info.IsDir() {
		if info.Name() == "vendor" {
			return false
		}
		return true
	}
	name := info.Name()
	return filepath.Ext(name) == ".go" && !strings.HasSuffix(name, "_test.go")
}

// JavaFiles matches Java source files and skips common build directories.
func JavaFiles(info os.FileInfo) bool {
	if info.IsDir() {
		// skip typical Java build/output dirs
		name := info.Name()
		if name == "target" || name == "build" || name == "out" {
			return false
		}
		return true
	}
	name := info.Name()
	return filepath.Ext(name) == ".java"
}
