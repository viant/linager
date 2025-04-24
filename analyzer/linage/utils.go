package linage

// Merge merges multiple PackageModel instances into a single global model.
// It concatenates scopes, files, and dataflows, and merges identifiers.
// In case of duplicate identifier keys, the first occurrence is preserved.
func Merge(models ...*PackageModel) *PackageModel {
	merged := NewPackageModel()
	// Use empty path for global model
	merged.Path = ""
	for _, m := range models {
		// append files
		merged.Files = append(merged.Files, m.Files...)
		// append scopes
		merged.Scopes = append(merged.Scopes, m.Scopes...)
		// merge identifiers
		for key, ident := range m.Idents {
			if _, exists := merged.Idents[key]; !exists {
				merged.Idents[key] = ident
			}
		}
		// append dataflow edges
		merged.DataFlows = append(merged.DataFlows, m.DataFlows...)
	}
	return merged
}
