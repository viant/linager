package java

import (
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/viant/linager/inspector/info"
	"strings"
)

// extractDocumentation extracts documentation comments from a node's modifiers
// and returns separate comment and annotation strings along with their locations
func extractDocumentation(node *sitter.Node, source []byte) (*info.LocationNode, *info.LocationNode) {
	var comments []string
	var annotations []string
	var commentLocation info.Location
	var annotationLocation info.Location

	// Check if the node has leading comments (Javadoc style)
	// Extract comments that appear before the node
	cursor := sitter.NewTreeCursor(node)
	if cursor.GoToFirstChild() {
		for {
			currentNode := cursor.CurrentNode()
			if currentNode.Type() == "comment" {
				commentText := strings.TrimSpace(currentNode.Content(source))
				// Remove comment markers (/* */, //, etc.)
				commentText = cleanCommentMarkers(commentText)

				// Store the location of the comment
				startPos := currentNode.StartByte()
				endPos := currentNode.EndByte()

				if !strings.HasPrefix(commentText, "@") {
					comments = append(comments, commentText)

					// Update comment location
					if commentLocation.Start == 0 || startPos < uint32(commentLocation.Start) {
						commentLocation.Start = int(startPos)
					}
					if endPos > uint32(commentLocation.End) {
						commentLocation.End = int(endPos)
					}
				} else {
					annotations = append(annotations, commentText)

					// Update annotation location
					if annotationLocation.Start == 0 || startPos < uint32(annotationLocation.Start) {
						annotationLocation.Start = int(startPos)
					}
					if endPos > uint32(annotationLocation.End) {
						annotationLocation.End = int(endPos)
					}
				}
			}
			if !cursor.GoToNextSibling() {
				break
			}
		}
	}

	// Check if the node has modifiers with annotations
	if node.NamedChildCount() > 0 && node.NamedChild(0).Type() == "modifiers" {
		modifiersNode := node.NamedChild(0)

		// Extract all annotations as documentation
		for i := uint32(0); i < modifiersNode.NamedChildCount(); i++ {
			modifier := modifiersNode.NamedChild(int(i))
			if modifier.Type() == "marker_annotation" || modifier.Type() == "annotation" {
				annotationText := modifier.Content(source)
				annotations = append(annotations, annotationText)

				// Update annotation location
				startPos := modifier.StartByte()
				endPos := modifier.EndByte()

				if annotationLocation.Start == 0 || startPos < uint32(annotationLocation.Start) {
					annotationLocation.Start = int(startPos)
				}
				if endPos > uint32(annotationLocation.End) {
					annotationLocation.End = int(endPos)
				}
			}
		}
	}

	commentNode := &info.LocationNode{
		Text:     strings.Join(comments, "\n"),
		Location: commentLocation,
	}

	annotationNode := &info.LocationNode{
		Text:     strings.Join(annotations, "\n"),
		Location: annotationLocation,
	}

	return commentNode, annotationNode
}

// cleanCommentMarkers removes comment markers from a comment string
func cleanCommentMarkers(comment string) string {
	// Remove /* */ style markers
	if strings.HasPrefix(comment, "/*") && strings.HasSuffix(comment, "*/") {
		comment = comment[2 : len(comment)-2]
	}
	// Remove // style markers
	if strings.HasPrefix(comment, "//") {
		comment = comment[2:]
	}
	// Clean up any * at the beginning of lines (common in Javadoc)
	lines := strings.Split(comment, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "*") {
			lines[i] = strings.TrimSpace(line[1:])
		} else {
			lines[i] = line
		}
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

// extractAnnotationFromComment extracts annotation information from a comment string
func extractAnnotationFromComment(comment string) string {
	var annotations []string
	lines := strings.Split(comment, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "@") {
			annotations = append(annotations, line)
		}
	}

	return strings.Join(annotations, "\n")
}

// extractRegularComment extracts regular comments excluding annotations
func extractRegularComment(comment string) string {
	var comments []string
	lines := strings.Split(comment, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "@") {
			comments = append(comments, line)
		}
	}

	return strings.Join(comments, "\n")
}
