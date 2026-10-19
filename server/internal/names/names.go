// Package names hands out readable agent names, so a board shared by many
// agents shows who holds what at a glance.
//
// The names are first names of characters from Robert Jordan's The Wheel of
// Time, taken from Karl Hammond's Compendium of Wheel of Time Characters
// (https://hammondkd.github.io/WoT-compendium/). They ship with the binary:
// LocalTask makes no outbound requests. To add names, append them to
// wot.txt, one per line.
package names

import (
	_ "embed"
	"fmt"
	"math/rand/v2"
	"strings"
)

//go:embed wot.txt
var wot string

// All is every bundled name, in list order.
var All = parse(wot)

func parse(text string) []string {
	var out []string
	for _, line := range strings.Split(text, "\n") {
		if name := strings.TrimSpace(line); name != "" {
			out = append(out, name)
		}
	}
	return out
}

// Pick returns a random name from pool that taken doesn't hold, comparing
// without case. When every name is taken it adds a number to one, such as
// "Rand-2", so it always returns a name nobody holds.
func Pick(pool []string, taken map[string]bool, intN func(int) int) string {
	if intN == nil {
		intN = rand.IntN
	}
	held := make(map[string]bool, len(taken))
	for name := range taken {
		held[strings.ToLower(name)] = true
	}
	var free []string
	for _, name := range pool {
		if !held[strings.ToLower(name)] {
			free = append(free, name)
		}
	}
	if len(free) > 0 {
		return free[intN(len(free))]
	}
	base := pool[intN(len(pool))]
	for n := 2; ; n++ {
		name := fmt.Sprintf("%s-%d", base, n)
		if !held[strings.ToLower(name)] {
			return name
		}
	}
}
