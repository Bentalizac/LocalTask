package project

import (
	"fmt"
	"strings"

	"localtask/internal/pathutil"
)

// ValidationError reports every problem found in a taskboard, rather than
// just the first one, so a human or agent fixing taskboard.json can address
// them in one pass.
type ValidationError struct {
	Issues []string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("taskboard validation failed (%d issue(s)):\n- %s",
		len(e.Issues), strings.Join(e.Issues, "\n- "))
}

// validateTaskboard checks a taskboard against the rules in the project
// spec: required fields, unique IDs, valid enum values, referential
// integrity of parent/depends_on, and that related paths stay within root.
// It is a pure function so it can be run against a trial taskboard before a
// mutation is committed.
func validateTaskboard(tb *Taskboard, root string) error {
	var issues []string

	ids := make(map[string]bool, len(tb.Tickets))
	for _, t := range tb.Tickets {
		if t.ID == "" {
			issues = append(issues, "ticket has no id")
			continue
		}
		if ids[t.ID] {
			issues = append(issues, fmt.Sprintf("duplicate ticket id %q", t.ID))
		}
		ids[t.ID] = true
	}

	for _, t := range tb.Tickets {
		ref := t.ID
		if ref == "" {
			ref = "<no id>"
		}
		prefix := fmt.Sprintf("ticket %s: ", ref)

		if t.Title == "" {
			issues = append(issues, prefix+"missing title")
		}
		if !t.Type.Valid() {
			issues = append(issues, prefix+fmt.Sprintf("invalid type %q", t.Type))
		}
		if !t.Status.Valid() {
			issues = append(issues, prefix+fmt.Sprintf("invalid status %q", t.Status))
		}
		if !t.Priority.Valid() {
			issues = append(issues, prefix+fmt.Sprintf("invalid priority %q", t.Priority))
		}
		if t.Parent != "" && !ids[t.Parent] {
			issues = append(issues, prefix+fmt.Sprintf("parent %q does not exist", t.Parent))
		}
		for _, dep := range t.DependsOn {
			if !ids[dep] {
				issues = append(issues, prefix+fmt.Sprintf("dependency %q does not exist", dep))
			}
		}
		for _, rel := range t.RelatedDocs {
			if err := pathWithinRoot(root, rel); err != nil {
				issues = append(issues, prefix+fmt.Sprintf("related_docs %q: %v", rel, err))
			}
		}
		for _, rel := range t.RelatedCode {
			if err := pathWithinRoot(root, rel); err != nil {
				issues = append(issues, prefix+fmt.Sprintf("related_code %q: %v", rel, err))
			}
		}
	}

	if len(issues) > 0 {
		return &ValidationError{Issues: issues}
	}
	return nil
}

// pathWithinRoot reports an error if rel escapes root — the containment
// rule from the local-only security model (no reading outside the project
// root).
func pathWithinRoot(root, rel string) error {
	_, err := pathutil.WithinRoot(root, rel)
	return err
}
