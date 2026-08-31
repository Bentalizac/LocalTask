package project

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var wordRE = regexp.MustCompile(`[A-Z]+[a-z0-9]*|[a-z0-9]+`)

// idPrefix derives a short ticket-ID prefix from a project name, taking the
// initial of each word (so "DanceNote" and "Job Tracker" both become
// recognizable short codes, e.g. "DN", "JT"). Falls back to "TCK".
func idPrefix(projectName string) string {
	words := wordRE.FindAllString(projectName, -1)
	var b strings.Builder
	for _, w := range words {
		b.WriteString(strings.ToUpper(w[:1]))
	}
	prefix := b.String()
	if len(prefix) < 2 && len(words) > 0 {
		w := words[0]
		n := min(len(w), 3)
		prefix = strings.ToUpper(w[:n])
	}
	if prefix == "" {
		prefix = "TCK"
	}
	return prefix
}

// generateTicketID returns an unused "<prefix>-<n>" ID for the taskboard,
// continuing the numbering sequence already in use for that prefix.
func generateTicketID(tb *Taskboard) string {
	prefix := idPrefix(tb.Project)
	existing := make(map[string]bool, len(tb.Tickets))
	max := 0
	for _, t := range tb.Tickets {
		existing[t.ID] = true
		if n, ok := ticketNumber(t.ID, prefix); ok && n > max {
			max = n
		}
	}
	for n := max + 1; ; n++ {
		id := fmt.Sprintf("%s-%d", prefix, n)
		if !existing[id] {
			return id
		}
	}
}

func ticketNumber(id, prefix string) (int, bool) {
	rest, ok := strings.CutPrefix(id, prefix+"-")
	if !ok {
		return 0, false
	}
	n, err := strconv.Atoi(rest)
	if err != nil {
		return 0, false
	}
	return n, true
}
