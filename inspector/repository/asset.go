package repository

import (
	"fmt"
	"github.com/viant/linager/inspector/info"
	"os"
	"path/filepath"
	"strings"
)

// HasFileWithSuffixes checks if a directory contains Go files
func HasFileWithSuffixes(dirPath string, inclusionSuffix, exclusionSuffix []string) (bool, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return false, err
	}

outer:
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		for _, suffix := range inclusionSuffix {
			if strings.HasSuffix(entry.Name(), suffix) {

				for _, exclusion := range exclusionSuffix {
					if strings.HasSuffix(entry.Name(), exclusion) {
						continue outer
					}
				}

				return true, nil

			}
		}
	}
	return false, nil
}

func ReadAssetsRecursively(packageDir string, isRoot bool, importPath func(relative string) string, skipExt ...string) ([]*structure.Asset, error) {
	var assets []*structure.Asset
	entries, err := os.ReadDir(packageDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}
	var subFolders []string
	var hasGoFiles bool
outer:
	for _, entry := range entries {
		if entry.IsDir() {
			subFolders = append(subFolders, entry.Name())
			continue
		}

		for _, ext := range skipExt {
			// Skip Go files (already processed)
			if strings.HasSuffix(entry.Name(), "."+ext) {
				hasGoFiles = true
				continue outer
			}
		}

		// Process as asset
		filePath := filepath.Join(packageDir, entry.Name())
		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read asset %s: %w", filePath, err)
		}

		asset := &structure.Asset{
			Path:       filePath,
			ImportPath: importPath(packageDir),
			Content:    content,
		}
		assets = append(assets, asset)
	}

	if hasGoFiles && !isRoot {
		return []*structure.Asset{}, nil
	}
	for _, subFolder := range subFolders {
		subAssets, err := ReadAssetsRecursively(filepath.Join(packageDir, subFolder), false, importPath, skipExt...)
		if err != nil {
			return nil, fmt.Errorf("failed to read assets in subfolder %s: %w", subFolder, err)
		}
		assets = append(assets, subAssets...)
	}

	return assets, nil
}
