package matcher

import (
	"regexp"
	"testing"

	"github.com/UnendingLoop/DistributedGrepClone/internal/model"
)

func TestFindMatch(t *testing.T) {
	tests := []struct {
		name       string
		pattern    string
		ignoreCase bool
		exactMatch bool
		text       string
		wantMatch  bool
	}{
		{
			name:       "exact string match",
			pattern:    "abc",
			exactMatch: true,
			text:       "xxabcxx",
			wantMatch:  true,
		},
		{
			name:       "exact string match not found",
			pattern:    "xyz",
			exactMatch: true,
			text:       "abc",
			wantMatch:  false,
		},
		{
			name:      "regexp match",
			pattern:   "a.c",
			text:      "xxabcxx",
			wantMatch: true,
		},
		{
			name:       "ignore case",
			pattern:    "abc",
			ignoreCase: true,
			exactMatch: true,
			text:       "AbC",
			wantMatch:  true,
		},
		{
			name:      "regexp OR pattern",
			pattern:   "(sdfg|123)",
			text:      "here is 123",
			wantMatch: true,
		},
		{
			name:      "digits in the beginning of string",
			pattern:   `^\d{3}-\d{2}-\d{4}$`,
			text:      "123-45-6789",
			wantMatch: true,
		},
		{
			name:      "invalid regexp digits format",
			pattern:   `^\d{3}-\d{2}-\d{4}$`,
			text:      "12-345-6789",
			wantMatch: false,
		},
		{
			name:      "or regexp statement '|'",
			pattern:   `cat|dog|bird`,
			text:      "I have a dog",
			wantMatch: true,
		},
		{
			name:      "groups and quantifiers",
			pattern:   `(ab){2,4}`,
			text:      "zzabababzz",
			wantMatch: true,
		},
		{
			name:       "case-non-sensitive",
			pattern:    `hello\s+world`,
			text:       "HeLLo    WoRLd",
			ignoreCase: true,
			wantMatch:  true,
		},
		{
			name:      "lookahead positive",
			pattern:   `foo(bar)`,
			text:      "foobar",
			wantMatch: true,
		},
		{
			name:      "lookahead negative",
			pattern:   `foo(bar)`,
			text:      "faabor",
			wantMatch: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var SP model.GrepParam
			SP.Pattern = tc.pattern
			SP.IgnoreCase = tc.ignoreCase
			SP.ExactMatch = tc.exactMatch

			pattern := tc.pattern
			if tc.ignoreCase {
				pattern = `(?i)` + pattern
			}
			re, err := regexp.Compile(pattern)
			if err != nil {
				t.Fatalf("ошибка компиляции regexp %q: %v", pattern, err)
			}
			SP.RegexpPattern = re

			got := FindMatch(&SP, tc.text)
			if got != tc.wantMatch {
				t.Errorf("%s failed:\nExpected: %v, Actual: %v \nPattern: %q\nInput: %q)", tc.name, tc.wantMatch, got, tc.pattern, tc.text)
			}
		})
	}
}
