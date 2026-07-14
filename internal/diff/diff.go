// Package diff implements the noise-aware response diff and write-set diff.
package diff

import (
	"encoding/base64"
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/pajamasi726/mocking-box/internal/writeset"
)

// Verdicts.
const (
	Match        = "MATCH"
	ResponseDiff = "RESPONSE_DIFF"
	WritesetDiff = "WRITESET_DIFF"
	BothDiff     = "BOTH_DIFF"
	// Noise: every difference matched the baseline (self-check) fingerprints —
	// a capture artifact, not a behavior change of the new stack.
	Noise = "NOISE"
	Error = "ERROR"
)

type Difference struct {
	Kind string `json:"kind"` // "response" | "writeset"
	Path string `json:"path"`
	Old  any    `json:"old"`
	New  any    `json:"new"`
}

const absent = "<absent>"

// ---------------------------------------------------------------------------
// value normalization (binlog & JSON values -> comparable primitives)
// ---------------------------------------------------------------------------

func NormValue(v any) any {
	switch x := v.(type) {
	case decimal.Decimal:
		// scale-insensitive: 50000 == 50000.0
		if x.Equal(x.Truncate(0)) {
			return x.IntPart()
		}
		f, _ := x.Float64()
		return f
	case []byte:
		if isPrintable(x) {
			return string(x)
		}
		return base64.StdEncoding.EncodeToString(x)
	case float64:
		// JSON numbers arrive as float64; canonicalize integral values
		if x == float64(int64(x)) {
			return int64(x)
		}
		return x
	case int:
		return int64(x)
	case int8:
		return int64(x)
	case int16:
		return int64(x)
	case int32:
		return int64(x)
	case uint:
		return int64(x)
	case uint8:
		return int64(x)
	case uint16:
		return int64(x)
	case uint32:
		return int64(x)
	case uint64:
		return int64(x)
	case float32:
		return NormValue(float64(x))
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, val := range x {
			out[k] = NormValue(val)
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, val := range x {
			out[i] = NormValue(val)
		}
		return out
	default:
		return v
	}
}

func isPrintable(b []byte) bool {
	for _, c := range b {
		if c < 0x09 {
			return false
		}
	}
	return true
}

func canonical(v any) string {
	b, err := json.Marshal(NormValue(v)) // encoding/json sorts map keys
	if err != nil {
		return "!" + strconv.Quote(err.Error())
	}
	return string(b)
}

func equal(a, b any) bool { return canonical(a) == canonical(b) }

// ---------------------------------------------------------------------------
// noise path matching for JSON bodies
//   pattern segments: literal | * (exactly one) | ** (zero or more)
// ---------------------------------------------------------------------------

func PathMatches(pattern string, path []string) bool {
	return match(strings.Split(pattern, "."), path)
}

func match(pat, path []string) bool {
	if len(pat) == 0 {
		return len(path) == 0
	}
	head, rest := pat[0], pat[1:]
	if head == "**" {
		for i := 0; i <= len(path); i++ {
			if match(rest, path[i:]) {
				return true
			}
		}
		return false
	}
	if len(path) == 0 {
		return false
	}
	if head == "*" || head == path[0] {
		return match(rest, path[1:])
	}
	return false
}

// SortRule sorts arrays whose path matches Path before comparison, so mapper-
// dependent ordering (Jackson vs MyBatis result order, …) doesn't read as a diff.
// By is an element key to sort by, or "$canonical" to sort by the element's
// canonical JSON.
type SortRule struct {
	Path string `json:"path" yaml:"path"`
	By   string `json:"by" yaml:"by"`
}

// SortArrays returns a copy of v with matching arrays canonically sorted.
func SortArrays(v any, rules []SortRule) any { return sortArrays(v, rules, nil) }

func sortArrays(v any, rules []SortRule, path []string) any {
	switch x := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, val := range x {
			out[k] = sortArrays(val, rules, append(append([]string{}, path...), k))
		}
		return out
	case []any:
		items := make([]any, len(x))
		for i, val := range x {
			items[i] = sortArrays(val, rules, append(append([]string{}, path...), strconv.Itoa(i)))
		}
		for _, rule := range rules {
			if !PathMatches(rule.Path, path) {
				continue
			}
			sortKey := func(e any) string {
				if rule.By != "" && rule.By != "$canonical" {
					if m, ok := e.(map[string]any); ok {
						return canonical(m[rule.By])
					}
				}
				return canonical(e)
			}
			sort.SliceStable(items, func(i, j int) bool { return sortKey(items[i]) < sortKey(items[j]) })
			break // first matching rule wins
		}
		return items
	default:
		return v
	}
}

// StripNoise returns a copy of v with noise-matching keys removed.
func StripNoise(v any, patterns []string) any { return stripNoise(v, patterns, nil) }

func stripNoise(v any, patterns []string, path []string) any {
	switch x := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(x))
	keys:
		for k, val := range x {
			child := append(append([]string{}, path...), k)
			for _, p := range patterns {
				if PathMatches(p, child) {
					continue keys
				}
			}
			out[k] = stripNoise(val, patterns, child)
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, val := range x {
			out[i] = stripNoise(val, patterns, append(append([]string{}, path...), strconv.Itoa(i)))
		}
		return out
	default:
		return v
	}
}

// ---------------------------------------------------------------------------
// response diff
// ---------------------------------------------------------------------------

func tryJSON(body string) (any, bool) {
	var v any
	if err := json.Unmarshal([]byte(body), &v); err != nil {
		return nil, false
	}
	switch v.(type) {
	case map[string]any, []any:
		return v, true
	}
	return v, false
}

func diffJSON(old, new any, path []string, out *[]Difference) {
	joined := func() string {
		if len(path) == 0 {
			return "$"
		}
		return strings.Join(path, ".")
	}
	oldMap, oldIsMap := old.(map[string]any)
	newMap, newIsMap := new.(map[string]any)
	if oldIsMap && newIsMap {
		keys := map[string]bool{}
		for k := range oldMap {
			keys[k] = true
		}
		for k := range newMap {
			keys[k] = true
		}
		sorted := make([]string, 0, len(keys))
		for k := range keys {
			sorted = append(sorted, k)
		}
		sort.Strings(sorted)
		for _, k := range sorted {
			child := append(append([]string{}, path...), k)
			ov, oOK := oldMap[k]
			nv, nOK := newMap[k]
			switch {
			case !oOK:
				*out = append(*out, Difference{"response", strings.Join(child, "."), absent, NormValue(nv)})
			case !nOK:
				*out = append(*out, Difference{"response", strings.Join(child, "."), NormValue(ov), absent})
			default:
				diffJSON(ov, nv, child, out)
			}
		}
		return
	}
	oldArr, oldIsArr := old.([]any)
	newArr, newIsArr := new.([]any)
	if oldIsArr && newIsArr {
		if len(oldArr) != len(newArr) {
			*out = append(*out, Difference{"response", joined() + ".length", len(oldArr), len(newArr)})
		}
		n := min(len(oldArr), len(newArr))
		for i := 0; i < n; i++ {
			diffJSON(oldArr[i], newArr[i], append(append([]string{}, path...), strconv.Itoa(i)), out)
		}
		return
	}
	if !equal(old, new) {
		*out = append(*out, Difference{"response", joined(), NormValue(old), NormValue(new)})
	}
}

// Options controls response comparison.
type Options struct {
	NoisePaths     []string
	CompareHeaders []string
	SortArrays     []SortRule
}

func DiffResponses(
	oldStatus int, oldBody string,
	newStatus int, newBody string,
	oldHeaders, newHeaders map[string]string,
	opts Options,
) []Difference {
	var diffs []Difference
	if oldStatus != newStatus {
		diffs = append(diffs, Difference{"response", "status", oldStatus, newStatus})
	}
	for _, h := range opts.CompareHeaders {
		ov, nv := oldHeaders[h], newHeaders[h]
		if ov != nv {
			diffs = append(diffs, Difference{"response", "header." + h, ov, nv})
		}
	}
	oldParsed, oldIsJSON := tryJSON(oldBody)
	newParsed, newIsJSON := tryJSON(newBody)
	if oldIsJSON || newIsJSON {
		normalize := func(v any) any {
			return SortArrays(StripNoise(v, opts.NoisePaths), opts.SortArrays)
		}
		diffJSON(normalize(oldParsed), normalize(newParsed), nil, &diffs)
	} else if oldBody != newBody {
		diffs = append(diffs, Difference{"response", "body", oldBody, newBody})
	}
	return diffs
}

// ---------------------------------------------------------------------------
// write-set diff
// ---------------------------------------------------------------------------

// WriteEntry is the canonical representation of one row change.
type WriteEntry struct {
	Table   string         `json:"table"`
	Op      string         `json:"op"`
	PK      any            `json:"pk"`
	Values  map[string]any `json:"values,omitempty"`  // INSERT / DELETE
	Changed map[string]any `json:"changed,omitempty"` // UPDATE: col -> {before, after}
}

func colIsNoise(table, column string, patterns []string) bool {
	bare := table
	if i := strings.LastIndex(table, "."); i >= 0 {
		bare = table[i+1:]
	}
	for _, pat := range patterns {
		pt, pc, found := strings.Cut(pat, ".")
		if !found {
			continue
		}
		if (pt == "*" || pt == bare || pt == table) && (pc == "*" || pc == column) {
			return true
		}
	}
	return false
}

// NormalizeWriteset produces a canonical, order-insensitive write-set:
// changed non-noise columns only for UPDATEs, rows keyed by their `id`.
func NormalizeWriteset(changes []writeset.RowChange, noiseColumns, ignoreTables []string) []WriteEntry {
	ignored := map[string]bool{}
	for _, t := range ignoreTables {
		ignored[t] = true
	}

	var out []WriteEntry
	for _, ch := range changes {
		bare := ch.Table
		if i := strings.LastIndex(bare, "."); i >= 0 {
			bare = bare[i+1:]
		}
		if ignored[bare] || ignored[ch.Table] {
			continue
		}

		clean := func(row map[string]any) map[string]any {
			if row == nil {
				return nil
			}
			m := map[string]any{}
			for k, v := range row {
				if !colIsNoise(ch.Table, k, noiseColumns) {
					m[k] = NormValue(v)
				}
			}
			return m
		}

		entry := WriteEntry{Table: bare, Op: ch.Op}
		pkSource := ch.After
		if ch.Op == "DELETE" {
			pkSource = ch.Before
		}
		if pkSource != nil {
			entry.PK = NormValue(pkSource["id"])
		}

		switch ch.Op {
		case "UPDATE":
			before, after := clean(ch.Before), clean(ch.After)
			changed := map[string]any{}
			for k := range union(before, after) {
				if !equal(before[k], after[k]) {
					changed[k] = map[string]any{"before": before[k], "after": after[k]}
				}
			}
			if len(changed) == 0 { // only noise columns changed
				continue
			}
			entry.Changed = changed
		case "INSERT":
			entry.Values = clean(ch.After)
		default: // DELETE
			entry.Values = clean(ch.Before)
		}
		out = append(out, entry)
	}

	sort.Slice(out, func(i, j int) bool { return canonical(out[i]) < canonical(out[j]) })
	return out
}

func union(a, b map[string]any) map[string]bool {
	keys := map[string]bool{}
	for k := range a {
		keys[k] = true
	}
	for k := range b {
		keys[k] = true
	}
	return keys
}

func (e WriteEntry) payload() map[string]any {
	if e.Changed != nil {
		return e.Changed
	}
	return e.Values
}

// DiffWritesets pairs entries by (table, op, pk) and reports divergences.
func DiffWritesets(oldWS, newWS []WriteEntry) []Difference {
	key := func(e WriteEntry) string {
		return e.Table + "|" + e.Op + "|" + canonical(e.PK)
	}
	label := func(e WriteEntry) string {
		return e.Table + "[" + e.Op + " pk=" + canonical(e.PK) + "]"
	}

	oldBy := map[string][]WriteEntry{}
	for _, e := range oldWS {
		oldBy[key(e)] = append(oldBy[key(e)], e)
	}
	newBy := map[string][]WriteEntry{}
	for _, e := range newWS {
		newBy[key(e)] = append(newBy[key(e)], e)
	}

	keys := map[string]bool{}
	for k := range oldBy {
		keys[k] = true
	}
	for k := range newBy {
		keys[k] = true
	}
	sorted := make([]string, 0, len(keys))
	for k := range keys {
		sorted = append(sorted, k)
	}
	sort.Strings(sorted)

	var diffs []Difference
	for _, k := range sorted {
		olds, news := oldBy[k], newBy[k]
		n := min(len(olds), len(news))
		for i := 0; i < n; i++ {
			o, nw := olds[i], news[i]
			if !equal(o.payload(), nw.payload()) {
				diffPayload(label(o), o.payload(), nw.payload(), &diffs)
			}
		}
		for _, extra := range olds[n:] {
			diffs = append(diffs, Difference{"writeset", label(extra), extra.payload(), absent})
		}
		for _, extra := range news[n:] {
			diffs = append(diffs, Difference{"writeset", label(extra), absent, extra.payload()})
		}
	}
	return diffs
}

func diffPayload(label string, old, new map[string]any, out *[]Difference) {
	for k := range union(old, new) {
		ov, oOK := old[k]
		nv, nOK := new[k]
		if !oOK {
			ov = absent
		}
		if !nOK {
			nv = absent
		}
		if !equal(ov, nv) {
			*out = append(*out, Difference{"writeset", label + "." + k, ov, nv})
		}
	}
	sort.Slice(*out, func(i, j int) bool { return (*out)[i].Path < (*out)[j].Path })
}

func VerdictOf(responseDiffs, writesetDiffs []Difference) string {
	switch {
	case len(responseDiffs) > 0 && len(writesetDiffs) > 0:
		return BothDiff
	case len(responseDiffs) > 0:
		return ResponseDiff
	case len(writesetDiffs) > 0:
		return WritesetDiff
	default:
		return Match
	}
}
