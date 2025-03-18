package repository

import (
	"bufio"
	"context"
	"github.com/viant/afs"
	"golang.org/x/mod/modfile"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Detector identifies project root folders and provides project-related information
type Detector struct {
	// Common project root marker files/directories
	markers []string
}

// New creates a new project detector instance
func New() *Detector {
	return &Detector{
		markers: []string{
			"go.mod",           // Go projects
			"pom.xml",          // Java/Maven projects
			"build.gradle",     // Java/Gradle projects
			"package.json",     // JavaScript/Node projects
			"composer.json",    // PHP projects
			"Cargo.toml",       // Rust projects
			"pyproject.toml",   // Python projects
			"requirements.txt", // Python projects
			"Gemfile",          // Ruby projects
			".git",             // Generic VCS marker
		},
	}
}

// DetectProject identifies the project root for the given file path and returns project info
func (d *Detector) DetectProject(filePath string, baseURL ...string) (*Project, error) {
	// Get the absolute path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, err
	}

	// If the path is a directory, start from there
	// If it's a file, start from its parent directory
	startDir := absPath
	fileInfo, err := os.Stat(absPath)
	if err != nil {
		return nil, err
	}

	if !fileInfo.IsDir() {
		startDir = filepath.Dir(absPath)
	}

	// Search up the directory tree for project markers
	rootPath, projectType := d.findProjectRoot(startDir)

	// Create default Project with fallback values
	info := &Project{
		Type:     "unknown",
		RootPath: absPath,
	}

	// Use baseURL if provided and no project root found
	if rootPath == "" && len(baseURL) > 0 && baseURL[0] != "" {
		info.RootPath = baseURL[0]
	} else if rootPath != "" {
		info.RootPath = rootPath
		info.Type = projectType
	}

	// Calculate relative path from project root to the file
	relPath, err := filepath.Rel(info.RootPath, absPath)
	if err != nil {
		// Fallback to just the filename if we can't get the relative path
		relPath = filepath.Base(absPath)
	}
	info.RelativePath = filepath.ToSlash(relPath)

	// Try to extract project name from config files
	if projectType != "" {
		info.Name = d.extractProjectName(rootPath, projectType)
	}

	return info, nil
}

// DetectRepository identifies the repository containing the given file path
func (d *Detector) DetectRepository(filePath string) (*Repository, error) {
	// Get the absolute path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, err
	}

	// If the path is a directory, start from there
	// If it's a file, start from its parent directory
	startDir := absPath
	fileInfo, err := os.Stat(absPath)
	if err != nil {
		return nil, err
	}

	if !fileInfo.IsDir() {
		startDir = filepath.Dir(absPath)
	}

	// First try to find a git repository
	gitRoot := d.findGitRoot(startDir)
	if gitRoot != "" {
		// We found a git repository
		repo := &Repository{
			Kind: "git",
			Root: gitRoot,
		}

		// Try to get the origin URL
		repo.Origin = d.extractGitOrigin(gitRoot)

		// Get project info
		info, err := d.DetectProject(filePath)
		if err == nil {
			repo.Info = info
		}

		return repo, nil
	}

	// If no git repository, try to find another type of project
	info, err := d.DetectProject(filePath)
	if err != nil {
		return nil, err
	}

	// If we found a project but not a git repository, create a simple repository
	repo := &Repository{
		Kind: info.Type,
		Root: info.RootPath,
		Info: info,
	}

	return repo, nil
}

// findProjectRoot searches up from the current directory for project markers
func (d *Detector) findProjectRoot(startDir string) (string, string) {
	dir := startDir

	// Search up the directory tree
	for {
		for _, marker := range d.markers {
			markerPath := filepath.Join(dir, marker)
			if _, err := os.Stat(markerPath); err == nil {
				projectType := determineProjectType(marker)
				return dir, projectType
			}
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// We've reached the filesystem root with no match
			break
		}
		dir = parent
	}

	return "", ""
}

// findGitRoot finds the root of the git repository containing the given directory
func (d *Detector) findGitRoot(startDir string) string {
	dir := startDir

	homeDir := os.Getenv("HOME")
	// Search up the directory tree for .git directory
	for {
		gitDir := filepath.Join(dir, ".git")
		if _, err := os.Stat(gitDir); err == nil {
			return dir
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// We've reached the filesystem root with no match
			break
		}
		if homeDir == parent {
			return ""
		}
		dir = parent
	}

	return ""
}

// extractGitOrigin extracts the origin URL from git config
func (d *Detector) extractGitOrigin(gitRoot string) string {
	configPath := filepath.Join(gitRoot, ".git", "config")
	if _, err := os.Stat(configPath); err != nil {
		return ""
	}

	file, err := os.Open(configPath)
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	foundRemote := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.Contains(line, "[remote \"origin\"]") {
			foundRemote = true
			continue
		}

		if foundRemote && strings.HasPrefix(line, "url = ") {
			return strings.TrimPrefix(line, "url = ")
		}
	}

	return ""
}

// extractProjectName attempts to extract a project name from configuration files
func (d *Detector) extractProjectName(rootPath string, projectType string) string {
	switch projectType {
	case "go":
		return extractGoModuleName(filepath.Join(rootPath, "go.mod"))
	case "javascript":
		return extractJSPackageName(filepath.Join(rootPath, "package.json"))
	case "java":
		if name := extractMavenProjectName(filepath.Join(rootPath, "pom.xml")); name != "" {
			return name
		}
		return extractGradleProjectName(filepath.Join(rootPath, "build.gradle"))
	case "python":
		if name := extractPyProjectName(filepath.Join(rootPath, "pyproject.toml")); name != "" {
			return name
		}
		return extractPythonPackageName(rootPath)
	case "rust":
		return extractCargoProjectName(filepath.Join(rootPath, "Cargo.toml"))
	case "git":
		// Extract name from git remote or directory name
		return extractGitProjectName(rootPath)
	default:
		// Fall back to directory name
		return filepath.Base(rootPath)
	}
}

// Helper functions to extract project names from various config files

func extractGoModuleName(goModPath string) string {
	fs := afs.New()
	if content, _ := fs.DownloadWithURL(context.Background(), goModPath); len(content) > 0 {
		if mod, _ := modfile.Parse(goModPath, content, nil); mod != nil {
			return mod.Module.Mod.Path
		}

	}
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return filepath.Base(filepath.Dir(goModPath))
	}
	moduleRegex := regexp.MustCompile(`module\s+([^\s]+)`)
	matches := moduleRegex.FindSubmatch(data)
	if len(matches) < 2 {
		return filepath.Base(filepath.Dir(goModPath))
	}

	// Extract the last part of the module path as the project name
	modulePath := string(matches[1])
	return modulePath
}

func extractJSPackageName(packageJsonPath string) string {
	data, err := os.ReadFile(packageJsonPath)
	if err != nil {
		return filepath.Base(filepath.Dir(packageJsonPath))
	}

	// Simple regex to extract the "name" field from package.json
	// This is not a full JSON parser but works for most cases
	nameRegex := regexp.MustCompile(`"name"\s*:\s*"([^"]+)"`)
	matches := nameRegex.FindSubmatch(data)
	if len(matches) < 2 {
		return filepath.Base(filepath.Dir(packageJsonPath))
	}

	return string(matches[1])
}

func extractMavenProjectName(pomPath string) string {
	data, err := os.ReadFile(pomPath)
	if err != nil {
		return ""
	}

	// Try to extract artifact ID first, then group ID if available
	artifactIDRegex := regexp.MustCompile(`<artifactId>([^<]+)</artifactId>`)
	matches := artifactIDRegex.FindSubmatch(data)
	if len(matches) < 2 {
		return ""
	}

	return string(matches[1])
}

func extractGradleProjectName(gradlePath string) string {
	data, err := os.ReadFile(gradlePath)
	if err != nil {
		return filepath.Base(filepath.Dir(gradlePath))
	}

	// Try to find rootProject.name or project.name
	nameRegex := regexp.MustCompile(`(?:rootProject|project)\.name\s*=\s*['"]([^'"]+)['"]`)
	matches := nameRegex.FindSubmatch(data)
	if len(matches) < 2 {
		return filepath.Base(filepath.Dir(gradlePath))
	}

	return string(matches[1])
}

func extractPyProjectName(pyprojectPath string) string {
	data, err := os.ReadFile(pyprojectPath)
	if err != nil {
		return ""
	}

	// Try to extract name from [tool.poetry] or [project] section
	nameRegex := regexp.MustCompile(`(?:tool\.poetry|project)\.name\s*=\s*["']([^"']+)["']`)
	matches := nameRegex.FindSubmatch(data)
	if len(matches) < 2 {
		return ""
	}

	return string(matches[1])
}

func extractPythonPackageName(rootPath string) string {
	// Look for setup.py or __init__.py to determine package name
	setupPath := filepath.Join(rootPath, "setup.py")
	if _, err := os.Stat(setupPath); err == nil {
		data, err := os.ReadFile(setupPath)
		if err == nil {
			nameRegex := regexp.MustCompile(`name\s*=\s*["']([^"']+)["']`)
			matches := nameRegex.FindSubmatch(data)
			if len(matches) >= 2 {
				return string(matches[1])
			}
		}
	}

	// Fall back to directory name
	return filepath.Base(rootPath)
}

func extractCargoProjectName(cargoPath string) string {
	data, err := os.ReadFile(cargoPath)
	if err != nil {
		return filepath.Base(filepath.Dir(cargoPath))
	}

	// Extract name from [package] section
	nameRegex := regexp.MustCompile(`\[package\](?:.|\n)*?name\s*=\s*["']([^"']+)["']`)
	matches := nameRegex.FindSubmatch(data)
	if len(matches) < 2 {
		return filepath.Base(filepath.Dir(cargoPath))
	}

	return string(matches[1])
}

func extractGitProjectName(gitRoot string) string {
	// Try to get the name from the origin remote
	configPath := filepath.Join(gitRoot, ".git", "config")
	if _, err := os.Stat(configPath); err == nil {
		file, err := os.Open(configPath)
		if err == nil {
			defer file.Close()
			scanner := bufio.NewScanner(file)
			foundRemote := false

			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())

				if strings.Contains(line, "[remote \"origin\"]") {
					foundRemote = true
					continue
				}

				if foundRemote && strings.HasPrefix(line, "url = ") {
					url := strings.TrimPrefix(line, "url = ")
					// Extract repo name from URL
					url = strings.TrimSuffix(url, ".git")
					parts := strings.Split(url, "/")
					if len(parts) > 0 {
						return parts[len(parts)-1]
					}
					break
				}
			}
		}
	}

	// Fall back to directory name
	return filepath.Base(gitRoot)
}

// determineProjectType identifies the type of project based on the marker file
func determineProjectType(marker string) string {
	switch marker {
	case "go.mod":
		return "go"
	case "pom.xml", "build.gradle":
		return "java"
	case "package.json":
		return "javascript"
	case "Cargo.toml":
		return "rust"
	case "pyproject.toml", "requirements.txt":
		return "python"
	case "Gemfile":
		return "ruby"
	case "composer.json":
		return "php"
	case ".git":
		return "git" // Generic project with version control
	default:
		return "unknown"
	}
}
