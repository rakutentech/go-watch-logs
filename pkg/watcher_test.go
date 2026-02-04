package pkg

import (
	"os"
	"strings"
	"testing"

	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/assert"
)

func setupTempFile(content string) (string, error) {
	tmpfile, err := os.CreateTemp("", "test.log")
	if err != nil {
		return "", err
	}
	if _, err := tmpfile.WriteString(content); err != nil {
		return "", err
	}
	if err := tmpfile.Close(); err != nil {
		return "", err
	}
	return tmpfile.Name(), nil
}

func TestNewWatcher(t *testing.T) {
	filePath := "test.log"    // nolint: goconst
	matchPattern := "error:1" // nolint: goconst
	ignorePattern := "ignore" // nolint: goconst

	f := Flags{
		Match:  matchPattern,
		Ignore: ignorePattern,
	}

	caches := make(map[string]*cache.Cache)
	caches[filePath] = cache.New(cache.NoExpiration, cache.NoExpiration)
	watcher, err := NewWatcher(filePath, f, caches[filePath], nil)
	watcher.incrementScanCount()
	assert.NoError(t, err)
	assert.NotNil(t, watcher)

	defer watcher.Close()
}

func TestReadFileAndMatchErrors(t *testing.T) {
	content := `line1
error:1
error:2
line2
error:1`
	filePath, err := setupTempFile(content)
	assert.NoError(t, err)
	defer os.Remove(filePath)

	matchPattern := `error:1` // nolint: goconst
	ignorePattern := `ignore` // nolint: goconst

	f := Flags{
		Match:  matchPattern,
		Ignore: ignorePattern,
	}

	caches := make(map[string]*cache.Cache)
	caches[filePath] = cache.New(cache.NoExpiration, cache.NoExpiration)
	watcher, err := NewWatcher(filePath, f, caches[filePath], nil)
	watcher.incrementScanCount()
	assert.NoError(t, err)
	defer watcher.Close()

	result, err := watcher.Scan()
	assert.NoError(t, err)
	assert.Equal(t, 2, result.ErrorCount)
	assert.Equal(t, "error:1", result.FirstLine)
	assert.Equal(t, "error:1", result.LastLine)
}

func TestLogRotation(t *testing.T) {
	content := `line1
error:1
line2`
	filePath, err := setupTempFile(content)
	assert.NoError(t, err)
	defer os.Remove(filePath)

	matchPattern := `error:1` // nolint: goconst
	ignorePattern := `ignore` // nolint: goconst

	f := Flags{
		Match:  matchPattern,
		Ignore: ignorePattern,
	}

	caches := make(map[string]*cache.Cache)
	caches[filePath] = cache.New(cache.NoExpiration, cache.NoExpiration)
	watcher, err := NewWatcher(filePath, f, caches[filePath], nil)
	watcher.incrementScanCount()
	assert.NoError(t, err)
	defer watcher.Close()

	result, err := watcher.Scan()
	assert.NoError(t, err)
	assert.Equal(t, 1, result.ErrorCount)
	assert.Equal(t, "error:1", result.FirstLine)
	assert.Equal(t, "error:1", result.LastLine)

	// Simulate log rotation by truncating the file
	err = os.WriteFile(filePath, []byte("error:1\n"), 0644) // nolint: gosec
	assert.NoError(t, err)

	result, err = watcher.Scan()
	assert.NoError(t, err)
	assert.Equal(t, 1, result.ErrorCount)
	assert.Equal(t, "error:1", result.FirstLine)
	assert.Equal(t, "error:1", result.LastLine)
}

func BenchmarkReadFileAndMatchErrors(b *testing.B) {
	content := `line1
error:1
error:2
line2
error:1`
	filePath, err := setupTempFile(content)
	if err != nil {
		b.Fatal(err)
	}
	defer os.Remove(filePath)

	matchPattern := `error:1`
	ignorePattern := `ignore`

	f := Flags{
		Match:  matchPattern,
		Ignore: ignorePattern,
	}

	caches := make(map[string]*cache.Cache)
	caches[filePath] = cache.New(cache.NoExpiration, cache.NoExpiration)
	watcher, err := NewWatcher(filePath, f, caches[filePath], nil)
	watcher.incrementScanCount()
	if err != nil {
		b.Fatal(err)
	}
	defer watcher.Close()

	for i := 0; i < b.N; i++ {
		_, err := watcher.Scan()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLoadAndSaveState(b *testing.B) {
	filePath := "test.log"
	matchPattern := "error:1"
	ignorePattern := "ignore"

	f := Flags{
		Match:  matchPattern,
		Ignore: ignorePattern,
	}

	caches := make(map[string]*cache.Cache)
	caches[filePath] = cache.New(cache.NoExpiration, cache.NoExpiration)
	watcher, err := NewWatcher(filePath, f, caches[filePath], nil)
	watcher.incrementScanCount()
	if err != nil {
		b.Fatal(err)
	}
	defer watcher.Close()

	watcher.lastLineNum = 10

	for i := 0; i < b.N; i++ {
		caches := make(map[string]*cache.Cache)
		caches[filePath] = cache.New(cache.NoExpiration, cache.NoExpiration)
		_, err := NewWatcher(filePath, f, caches[filePath], nil)
		watcher.incrementScanCount()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestSplitPattern(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		expected int // expected number of parts
	}{
		{
			name:     "simple pattern with no pipes",
			pattern:  "error",
			expected: 1,
		},
		{
			name:     "pattern with pipes",
			pattern:  "error|warning|critical",
			expected: 3,
		},
		{
			name:     "pattern with pipes and groups",
			pattern:  "(error|warning)|critical|(info|debug)",
			expected: 5, // Simplified: splits on all pipes, not just top-level
		},
		{
			name:     "pattern with character class containing pipe",
			pattern:  "[a-z|]|error",
			expected: 3, // Simplified: splits on pipes in character classes too
		},
		{
			name:     "pattern with escaped pipe",
			pattern:  `error\|warning|critical`,
			expected: 2,
		},
		{
			name:     "pattern with multiple escaped pipes",
			pattern:  `error\|warning\|info|critical`,
			expected: 2, // "error\|warning\|info" and "critical"
		},
		{
			name:     "pattern with escaped backslash before pipe",
			pattern:  `error\\|warning|critical`,
			expected: 3, // "error\\", "warning", "critical"
		},
		{
			name:     "pattern with double escaped pipe",
			pattern:  `error\\\|warning|critical`,
			expected: 2, // "error\\\|warning" and "critical"
		},
		{
			name:     "pattern with only escaped pipes",
			pattern:  `error\|warning\|critical`,
			expected: 1, // Should not split at all
		},
		{
			name:     "pattern ending with escaped pipe",
			pattern:  `error|warning\|`,
			expected: 2, // "error" and "warning\|"
		},
		{
			name:     "pattern starting with escaped pipe",
			pattern:  `\|error|warning`,
			expected: 2, // "\|error" and "warning"
		},
		{
			name:     "empty pattern",
			pattern:  "",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts := splitPattern(tt.pattern)
			if tt.expected == 0 {
				assert.Empty(t, parts)
			} else {
				assert.Len(t, parts, tt.expected)
			}
		})
	}
}

func TestSplitAndCompilePattern(t *testing.T) {
	tests := []struct {
		name          string
		pattern       string
		shouldSplit   bool
		shouldError   bool
		expectedMatch []string // Test strings that should match
		expectedNoMatch []string // Test strings that should NOT match
	}{
		{
			name:        "short pattern - no split",
			pattern:     "error|warning|critical",
			shouldSplit: false,
			shouldError: false,
			expectedMatch: []string{"error", "warning", "critical", "an error occurred", "warning: failed"},
			expectedNoMatch: []string{"info", "debug", "success"},
		},
		{
			name: "long pattern - should split",
			pattern: "error1|error2|error3|error4|error5|error6|error7|error8|error9|error10|" +
				"error11|error12|error13|error14|error15|error16|error17|error18|error19|error20|" +
				"error21|error22|error23|error24|error25|error26|error27|error28|error29|error30|" +
				"error31|error32|error33|error34|error35|error36|error37|error38|error39|error40|" +
				"error41|error42|error43|error44|error45|error46|error47|error48|error49|error50|" +
				"error51|error52|error53|error54|error55|error56|error57|error58|error59|error60|" +
				"error61|error62|error63|error64|error65|error66|error67|error68|error69|error70|" +
				"error71|error72|error73|error74|error75|error76|error77|error78|error79|error80|" +
				"error81|error82|error83|error84|error85|error86|error87|error88|error89|error90|" +
				"error91|error92|error93|error94|error95|error96|error97|error98|error99|error100",
			shouldSplit: true,
			shouldError: false,
			expectedMatch: []string{"error1", "error50", "error100", "found error1 in log"},
			expectedNoMatch: []string{"error", "warning", "error0"},
		},
		{
			name:        "empty pattern",
			pattern:     "",
			shouldSplit: false,
			shouldError: false,
		},
		{
			name:        "invalid regex - unclosed bracket",
			pattern:     "[invalid",
			shouldSplit: false,
			shouldError: true,
		},
		{
			name:        "invalid regex - unclosed parenthesis",
			pattern:     "(error|warning",
			shouldSplit: false,
			shouldError: true,
		},
		{
			name:        "pattern at threshold boundary (499 chars) - valid regex",
			pattern:     "a|b|c|" + strings.Repeat("x", 493), // 5 chars + 493 = 498 total
			shouldSplit: false,
			shouldError: false,
			expectedMatch: []string{"a", "b", strings.Repeat("x", 493)},
			expectedNoMatch: []string{"d", "y"},
		},
		{
			name:        "pattern just above threshold (501 chars) - should split",
			pattern:     "a|b|c|" + strings.Repeat("d|", 248), // Creates ~500+ char pattern
			shouldSplit: true,
			shouldError: false,
			expectedMatch: []string{"a", "b", "c", "d"},
			expectedNoMatch: []string{"z"},
		},
		{
			name:        "pattern with only escaped pipes - should not split",
			pattern:     `error\|warning\|critical\|fatal\|panic\|alert\|emergency\|disaster\|catastrophe\|failure\|breakdown\|malfunction\|defect\|fault\|flaw\|glitch\|bug\|issue\|problem\|trouble\|difficulty\|complication\|obstacle\|impediment\|hindrance\|barrier\|blockage\|stoppage\|interruption\|disruption\|disturbance\|interference\|conflict\|contradiction\|inconsistency\|discrepancy\|anomaly\|irregularity\|deviation\|aberration\|abnormality\|exception\|violation\|breach\|infringement\|transgression\|offense\|misdeed\|wrongdoing\|misconduct\|malpractice\|negligence\|oversight\|omission\|mistake\|blunder\|gaffe\|slip\|lapse\|oversight`,
			shouldSplit: false, // Even though long, no real | separators
			shouldError: false,
			expectedMatch: []string{
				`error|warning|critical|fatal|panic|alert|emergency|disaster|catastrophe|failure|breakdown|malfunction|defect|fault|flaw|glitch|bug|issue|problem|trouble|difficulty|complication|obstacle|impediment|hindrance|barrier|blockage|stoppage|interruption|disruption|disturbance|interference|conflict|contradiction|inconsistency|discrepancy|anomaly|irregularity|deviation|aberration|abnormality|exception|violation|breach|infringement|transgression|offense|misdeed|wrongdoing|misconduct|malpractice|negligence|oversight|omission|mistake|blunder|gaffe|slip|lapse|oversight`,
			}, // Matches exact literal string with pipes
			expectedNoMatch: []string{"error", "warning", "critical", "error|warning"},
		},
		{
			name: "mixed escaped and unescaped pipes - long pattern",
			pattern: `error\|fatal|warning\|alert|critical\|severe|` +
				`panic\|emergency|debug\|trace|info\|notice|` +
				`exception\|throw|failure\|crash|timeout\|delay|` +
				`connection\|socket|database\|query|authentication\|auth|` +
				`permission\|access|network\|connectivity|memory\|leak|` +
				`disk\|storage|cpu\|processor|thread\|deadlock|` +
				`null\|undefined|overflow\|underflow|parse\|syntax|` +
				`validation\|constraint|transaction\|rollback|lock\|contention|` +
				strings.Repeat(`extra\|pipe|`, 10), // Make it long enough to split
			shouldSplit: true,
			shouldError: false,
			expectedMatch: []string{`error|fatal`, `warning|alert`, `connection|socket`, `extra|pipe`},
			expectedNoMatch: []string{"error", "warning", "random", "connection", "socket"},
		},
		{
			name:        "very long single alternative - no pipes",
			pattern:     `error.*occurred.*in.*module.*with.*very.*long.*description.*and.*many.*words.*and.*more.*words.*and.*even.*more.*words.*to.*make.*it.*exceed.*the.*threshold.*of.*500.*characters.*so.*that.*we.*can.*test.*the.*behavior.*when.*there.*are.*no.*pipe.*separators.*but.*the.*pattern.*is.*still.*very.*long.*and.*needs.*to.*be.*handled.*correctly.*by.*the.*compilation.*logic.*without.*attempting.*to.*split.*it.*into.*multiple.*parts.*because.*there.*are.*no.*pipe.*characters.*to.*split.*on.*at.*all.*in.*this.*entire.*long.*pattern.*string.*that.*we.*are.*testing`,
			shouldSplit: false, // No pipes to split on
			shouldError: false,
			expectedMatch: []string{
				"error occurred in module with very long description and many words and more words and even more words to make it exceed the threshold of 500 characters so that we can test the behavior when there are no pipe separators but the pattern is still very long and needs to be handled correctly by the compilation logic without attempting to split it into multiple parts because there are no pipe characters to split on at all in this entire long pattern string that we are testing",
			},
			expectedNoMatch: []string{"warning", "info", "error module"},
		},
		{
			name:        "many short alternatives",
			pattern:     "a|b|c|d|e|f|g|h|i|j|k|l|m|n|o|p|q|r|s|t|u|v|w|x|y|z|aa|bb|cc|dd|ee|ff|gg|hh|ii|jj|kk|ll|mm|nn|oo|pp|qq|rr|ss|tt|uu|vv|ww|xx|yy|zz|aaa|bbb|ccc|ddd|eee|fff|ggg|hhh|iii|jjj|kkk|lll|mmm|nnn|ooo|ppp|qqq|rrr|sss|ttt|uuu|vvv|www|xxx|yyy|zzz",
			shouldSplit: false, // Total length < 500
			shouldError: false,
			expectedMatch: []string{"a", "z", "aa", "zzz", "found a in log", "the zzz value", "aaaa"}, // aaaa contains aaa
			expectedNoMatch: []string{"A", "1", "ERROR"},
		},
		{
			name:        "pattern with special regex characters",
			pattern:     `\d+\.\d+\.\d+\.\d+|ERROR|WARNING|CRITICAL|[0-9]{4}-[0-9]{2}-[0-9]{2}`,
			shouldSplit: false,
			shouldError: false,
			expectedMatch: []string{"192.168.1.1", "ERROR", "2024-01-01"},
			expectedNoMatch: []string{"192.168", "error", "2024/01/01"},
		},
		{
			name:        "pattern with Unicode characters",
			pattern:     "错误|警告|致命|エラー|警告|致命的|오류|경고|치명적",
			shouldSplit: false,
			shouldError: false,
			expectedMatch: []string{"错误", "エラー", "오류"},
			expectedNoMatch: []string{"error", "warning"},
		},
		{
			name:        "pattern with empty alternatives",
			pattern:     "error||warning|||critical",
			shouldSplit: false,
			shouldError: false,
			expectedMatch: []string{"error", "warning", "critical", "", "x"}, // Empty string matches empty alternative
			expectedNoMatch: []string{},
		},
		{
			name:        "pattern with word boundaries",
			pattern:     `\berror\b|\bwarning\b|\bcritical\b`,
			shouldSplit: false,
			shouldError: false,
			expectedMatch: []string{"error", "warning", "critical", "an error occurred"},
			expectedNoMatch: []string{"errors", "warnings", "critically"},
		},
		{
			name:        "pattern with anchors",
			pattern:     "^error|^warning|^critical|^fatal",
			shouldSplit: false,
			shouldError: false,
			expectedMatch: []string{"error", "warning at start"},
			expectedNoMatch: []string{"an error", "the warning"},
		},
		{
			name:        "pattern with quantifiers",
			pattern:     "error+|warn(ing)?|critical{1,3}",
			shouldSplit: false,
			shouldError: false,
			expectedMatch: []string{"error", "errorr", "errorrr", "warn", "warning", "an error occurred", "warn user", "critical issue"},
			expectedNoMatch: []string{"erro", "wrn", "critik"},
		},
		{
			name:        "pattern with character classes",
			pattern:     "[Ee]rror|[Ww]arning|[Cc]ritical",
			shouldSplit: false,
			shouldError: false,
			expectedMatch: []string{"Error", "error", "Warning", "warning"},
			expectedNoMatch: []string{"ERROR", "WARNING"},
		},
		{
			name: "very long pattern with complex regex",
			pattern: `\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.*ERROR|` +
				`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.*WARN|` +
				`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.*CRITICAL|` +
				`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.*FATAL|` +
				`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.*PANIC|` +
				`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.*ALERT|` +
				`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.*EMERGENCY|` +
				`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.*SEVERE|` +
				`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.*EXCEPTION|` +
				`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.*FAILURE|` +
				`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.*TIMEOUT|` +
				`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.*CRASH|` +
				`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.*ABORT|` +
				`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.*FAULT`,
			shouldSplit: true,
			shouldError: false,
			expectedMatch: []string{"2024-01-01T12:00:00 ERROR", "2024-01-01T12:00:00 WARN", "2024-01-01T12:00:00 some message CRASH"},
			expectedNoMatch: []string{"2024-01-01 ERROR", "ERROR", "12:00:00 ERROR"},
		},
		{
			name:        "pattern with lookahead (not supported in Go)",
			pattern:     `error(?=:)|warning(?=:)|critical(?=:)`,
			shouldSplit: false,
			shouldError: true, // Go regex doesn't support lookahead
		},
		{
			name:        "pattern with backreferences (not supported in Go RE2)",
			pattern:     `(\w+)\s+\1|error|warning`,
			shouldSplit: false,
			shouldError: true, // Go's RE2 regex doesn't support backreferences
		},
		{
			name: "realistic log pattern - short",
			pattern: `level=error|level=fatal|level=panic|severity=ERROR|severity=FATAL`,
			shouldSplit: false,
			shouldError: false,
			expectedMatch: []string{"level=error", "severity=ERROR"},
			expectedNoMatch: []string{"level=info", "severity=WARN"},
		},
		{
			name: "realistic log pattern - long with many log levels",
			pattern: `level=error|level=fatal|level=panic|level=critical|level=alert|level=emergency|` +
				`severity=ERROR|severity=FATAL|severity=PANIC|severity=CRITICAL|severity=ALERT|` +
				`status=error|status=failed|status=failure|status=crashed|status=exception|` +
				`type=error|type=fatal|type=panic|type=critical|type=exception|type=failure|` +
				`category=error|category=fatal|category=critical|category=severe|category=emergency|` +
				`priority=high|priority=critical|priority=urgent|priority=immediate|priority=emergency|` +
				`code=500|code=501|code=502|code=503|code=504|code=505|code=506|code=507|code=508|code=509|code=510`,
			shouldSplit: true,
			shouldError: false,
			expectedMatch: []string{"level=error", "severity=FATAL", "code=500", "priority=urgent"},
			expectedNoMatch: []string{"level=info", "code=200", "priority=low"},
		},
		{
			name:        "pattern with only one alternative - long",
			pattern:     "a_very_long_single_pattern_without_any_pipe_separators_that_exceeds_the_threshold_of_500_characters_by_being_extremely_verbose_and_containing_many_words_and_underscores_to_make_it_longer_and_longer_until_it_finally_reaches_the_required_length_for_testing_purposes_and_to_ensure_that_the_splitting_logic_handles_single_alternatives_correctly_without_attempting_to_split_them_into_multiple_parts_because_there_are_no_pipe_characters_present_in_this_entire_string_at_all_which_makes_it_impossible_to_split_on_anything_other_than_the_pattern_itself",
			shouldSplit: false,
			shouldError: false,
			expectedMatch: []string{"a_very_long_single_pattern_without_any_pipe_separators_that_exceeds_the_threshold_of_500_characters_by_being_extremely_verbose_and_containing_many_words_and_underscores_to_make_it_longer_and_longer_until_it_finally_reaches_the_required_length_for_testing_purposes_and_to_ensure_that_the_splitting_logic_handles_single_alternatives_correctly_without_attempting_to_split_them_into_multiple_parts_because_there_are_no_pipe_characters_present_in_this_entire_string_at_all_which_makes_it_impossible_to_split_on_anything_other_than_the_pattern_itself"},
			expectedNoMatch: []string{"error", "warning", "short"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			regexes, err := splitAndCompilePattern(tt.pattern)

			if tt.shouldError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			if tt.pattern == "" {
				assert.Nil(t, regexes)
				return
			}

			assert.NotEmpty(t, regexes)

			if tt.shouldSplit {
				// For long patterns, we expect multiple regexes
				assert.Greater(t, len(regexes), 1, "Expected pattern to be split into multiple regexes")
			} else {
				// For short patterns, we expect a single regex
				assert.Equal(t, 1, len(regexes), "Expected pattern to remain as single regex")
			}

			// Test actual matching behavior
			for _, matchStr := range tt.expectedMatch {
				matched := false
				for _, re := range regexes {
					if re.MatchString(matchStr) {
						matched = true
						break
					}
				}
				assert.True(t, matched, "Expected pattern to match '%s'", matchStr)
			}

			for _, noMatchStr := range tt.expectedNoMatch {
				matched := false
				for _, re := range regexes {
					if re.MatchString(noMatchStr) {
						matched = true
						break
					}
				}
				assert.False(t, matched, "Expected pattern NOT to match '%s'", noMatchStr)
			}
		})
	}
}

func TestMatchesAny(t *testing.T) {
	filePath := "test.log"
	f := Flags{
		Match:  "error|warning",
		Ignore: "ignore",
	}

	caches := make(map[string]*cache.Cache)
	caches[filePath] = cache.New(cache.NoExpiration, cache.NoExpiration)
	watcher, err := NewWatcher(filePath, f, caches[filePath], nil)
	assert.NoError(t, err)
	defer watcher.Close()

	tests := []struct {
		name     string
		line     string
		expected bool
	}{
		{
			name:     "matches error",
			line:     "This is an error message",
			expected: true,
		},
		{
			name:     "matches warning",
			line:     "This is a warning message",
			expected: true,
		},
		{
			name:     "no match",
			line:     "This is an info message",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := watcher.matchesAny(watcher.regexMatch, []byte(tt.line))
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEscapedPipeInActualMatching(t *testing.T) {
	// Test that escaped pipes work correctly in actual log scanning
	// The pattern should match literal "error|warning" but not "errorXwarning"
	content := `line1
error|warning found
line2
errorXwarning found
line3
error or warning
`
	filePath, err := setupTempFile(content)
	assert.NoError(t, err)
	defer os.Remove(filePath)

	// Pattern with escaped pipe - should match literal "error|warning"
	matchPattern := `error\|warning`

	f := Flags{
		Match:  matchPattern,
		Ignore: "",
	}

	caches := make(map[string]*cache.Cache)
	caches[filePath] = cache.New(cache.NoExpiration, cache.NoExpiration)
	watcher, err := NewWatcher(filePath, f, caches[filePath], nil)
	watcher.incrementScanCount()
	assert.NoError(t, err)
	defer watcher.Close()

	result, err := watcher.Scan()
	assert.NoError(t, err)
	// Should only match "error|warning found" line, not the others
	assert.Equal(t, 1, result.ErrorCount, "Should match only the line with literal pipe")
	assert.Contains(t, result.FirstLine, "error|warning found")
}

func TestUnescapedPipeInActualMatching(t *testing.T) {
	// Test that unescaped pipes work as OR operator
	content := `line1
error found
line2
warning found
line3
info found
`
	filePath, err := setupTempFile(content)
	assert.NoError(t, err)
	defer os.Remove(filePath)

	// Pattern with unescaped pipe - should match "error" OR "warning"
	matchPattern := `error|warning`

	f := Flags{
		Match:  matchPattern,
		Ignore: "",
	}

	caches := make(map[string]*cache.Cache)
	caches[filePath] = cache.New(cache.NoExpiration, cache.NoExpiration)
	watcher, err := NewWatcher(filePath, f, caches[filePath], nil)
	watcher.incrementScanCount()
	assert.NoError(t, err)
	defer watcher.Close()

	result, err := watcher.Scan()
	assert.NoError(t, err)
	// Should match both "error found" and "warning found", but not "info found"
	assert.Equal(t, 2, result.ErrorCount, "Should match lines with either error or warning")
}

func TestLongPatternPerformance(t *testing.T) {
	// Create a very long pattern with many alternatives
	var longPattern string
	for i := 0; i < 100; i++ {
		if i > 0 {
			longPattern += "|"
		}
		longPattern += "error" + string(rune('A'+i%26))
	}

	content := "line1\nerrorA\nline2\nerrorB\nline3\n"
	filePath, err := setupTempFile(content)
	assert.NoError(t, err)
	defer os.Remove(filePath)

	f := Flags{
		Match:  longPattern,
		Ignore: "",
	}

	caches := make(map[string]*cache.Cache)
	caches[filePath] = cache.New(cache.NoExpiration, cache.NoExpiration)
	watcher, err := NewWatcher(filePath, f, caches[filePath], nil)
	watcher.incrementScanCount()
	assert.NoError(t, err)
	defer watcher.Close()

	// Verify that the pattern was split
	assert.Greater(t, len(watcher.regexMatch), 1, "Expected long pattern to be split")

	// Verify that scanning still works correctly
	result, err := watcher.Scan()
	assert.NoError(t, err)
	assert.Greater(t, result.ErrorCount, 0, "Should find matches with split pattern")
}

func BenchmarkLogRotation(b *testing.B) {
	content := `line1
error:1
line2`
	filePath, err := setupTempFile(content)
	if err != nil {
		b.Fatal(err)
	}
	defer os.Remove(filePath)

	matchPattern := `error:1`
	ignorePattern := `ignore`

	f := Flags{
		Match:  matchPattern,
		Ignore: ignorePattern,
	}

	caches := make(map[string]*cache.Cache)
	caches[filePath] = cache.New(cache.NoExpiration, cache.NoExpiration)
	watcher, err := NewWatcher(filePath, f, caches[filePath], nil)
	watcher.incrementScanCount()
	if err != nil {
		b.Fatal(err)
	}
	defer watcher.Close()

	result, err := watcher.Scan()
	if err != nil {
		b.Fatal(err)
	}
	if result.ErrorCount != 1 || result.FirstLine != "error:1" || result.LastLine != "error:1" {
		b.Fatalf("Unexpected results: count=%d, first=%s, last=%s", result.ErrorCount, result.FirstLine, result.LastLine)
	}

	err = os.WriteFile(filePath, []byte("error:1\n"), 0644) // nolint: gosec
	if err != nil {
		b.Fatal(err)
	}

	watcher.lastLineNum = 0

	for i := 0; i < b.N; i++ {
		_, err := watcher.Scan()
		if err != nil {
			b.Fatal(err)
		}
	}
}
