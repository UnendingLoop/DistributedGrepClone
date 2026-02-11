// Package matcher ckecks input line for matching the pattern - string or regexp, returns bool
package matcher

import (
	"strings"

	"github.com/UnendingLoop/DistributedGrepClone/internal/model"
)

func FindMatch(SP *model.GrepParam, line string) bool {
	if SP.IgnoreCase { //-i
		line = strings.ToLower(line)
	}

	switch {
	case SP.ExactMatch: //-F
		return strings.Contains(line, SP.Pattern)
	default:
		return SP.RegexpPattern.MatchString(line)
	}
}
