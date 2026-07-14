package diff

import (
	"strings"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/pajamasi726/mocking-box/internal/writeset"
)

var noiseCols = []string{"*.created_at", "*.updated_at"}

func dec(s string) decimal.Decimal {
	d, err := decimal.NewFromString(s)
	if err != nil {
		panic(err)
	}
	return d
}

func TestPathMatches(t *testing.T) {
	cases := []struct {
		pattern string
		path    []string
		want    bool
	}{
		{"data.updated_at", []string{"data", "updated_at"}, true},
		{"**.updated_at", []string{"a", "b", "updated_at"}, true},
		{"**.updated_at", []string{"updated_at"}, true},
		{"data.*.id", []string{"data", "0", "id"}, true},
		{"data.updated_at", []string{"data", "balance"}, false},
		{"*.id", []string{"a", "b", "id"}, false},
	}
	for _, c := range cases {
		if got := PathMatches(c.pattern, c.path); got != c.want {
			t.Errorf("PathMatches(%q, %v) = %v, want %v", c.pattern, c.path, got, c.want)
		}
	}
}

func TestStripNoiseNested(t *testing.T) {
	body := map[string]any{
		"wallet_id":  float64(1),
		"updated_at": "2026-07-14T10:00:00",
		"items": []any{
			map[string]any{"id": float64(1), "created_at": "x"},
			map[string]any{"id": float64(2), "created_at": "y"},
		},
	}
	cleaned := StripNoise(body, []string{"**.updated_at", "**.created_at"})
	want := map[string]any{
		"wallet_id": float64(1),
		"items": []any{
			map[string]any{"id": float64(1)},
			map[string]any{"id": float64(2)},
		},
	}
	if canonical(cleaned) != canonical(want) {
		t.Errorf("got %s, want %s", canonical(cleaned), canonical(want))
	}
}

func TestResponseDiffIgnoresNoiseAndKeyOrder(t *testing.T) {
	old := `{"wallet_id": 1, "balance": 55000, "updated_at": "2026-07-14T10:00:00"}`
	new := `{"updated_at": "2026-07-14T10:00:03", "balance": 55000, "wallet_id": 1}`
	diffs := DiffResponses(200, old, 200, new, nil, nil, Options{NoisePaths: []string{"**.updated_at"}})
	if len(diffs) != 0 {
		t.Errorf("expected no diffs, got %+v", diffs)
	}
}

func TestResponseDiffCatchesRealChange(t *testing.T) {
	diffs := DiffResponses(200, `{"balance": 55000}`, 200, `{"balance": 56000}`, nil, nil, Options{})
	if len(diffs) != 1 || diffs[0].Path != "balance" {
		t.Errorf("expected one balance diff, got %+v", diffs)
	}
}

func TestResponseDiffStatus(t *testing.T) {
	diffs := DiffResponses(200, `{}`, 404, `{}`, nil, nil, Options{})
	found := false
	for _, d := range diffs {
		if d.Path == "status" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected status diff, got %+v", diffs)
	}
}

func chargeChangesOld() []writeset.RowChange {
	return []writeset.RowChange{
		{
			Table: "demo.wallet", Op: "UPDATE",
			Before: map[string]any{"id": int64(1), "user_id": int64(101), "balance": dec("50000"),
				"updated_at": "2026-07-14 10:00:00", "created_at": "2026-01-01 00:00:00"},
			After: map[string]any{"id": int64(1), "user_id": int64(101), "balance": dec("55000"),
				"updated_at": "2026-07-14 10:00:01", "created_at": "2026-01-01 00:00:00"},
			Query: "UPDATE wallet SET balance = balance + 5000 WHERE id = 1",
		},
		{
			Table: "demo.wallet_history", Op: "INSERT",
			After: map[string]any{"id": int64(1), "wallet_id": int64(1), "type": "CHARGE",
				"amount": dec("5000"), "balance_after": dec("55000"),
				"created_at": "2026-07-14 10:00:01"},
		},
	}
}

func chargeChangesNew() []writeset.RowChange {
	return []writeset.RowChange{
		{ // different execution order
			Table: "demo.wallet_history", Op: "INSERT",
			After: map[string]any{"id": int64(1), "wallet_id": int64(1), "type": "CHARGE",
				"amount": dec("5000"), "balance_after": dec("55000"),
				"created_at": "2026-07-14 10:00:07"},
		},
		{
			Table: "demo.wallet", Op: "UPDATE",
			Before: map[string]any{"id": int64(1), "user_id": int64(101), "balance": dec("50000"),
				"updated_at": "2026-07-14 10:00:00", "created_at": "2026-01-01 00:00:00"},
			After: map[string]any{"id": int64(1), "user_id": int64(101), "balance": dec("55000"),
				"updated_at": "2026-07-14 10:00:07", "created_at": "2026-01-01 00:00:00"},
			Query: "UPDATE wallet SET updated_at = NOW(), balance = 55000 WHERE id = 1",
		},
	}
}

func TestWritesetMatchDespiteDifferentSQLAndOrder(t *testing.T) {
	oldWS := NormalizeWriteset(chargeChangesOld(), noiseCols, nil)
	newWS := NormalizeWriteset(chargeChangesNew(), noiseCols, nil)
	if diffs := DiffWritesets(oldWS, newWS); len(diffs) != 0 {
		t.Errorf("expected match, got %+v", diffs)
	}
}

func TestWritesetDetectsMissingInsert(t *testing.T) {
	oldWS := NormalizeWriteset(chargeChangesOld(), noiseCols, nil)
	newWS := NormalizeWriteset(chargeChangesNew()[1:], noiseCols, nil) // history INSERT lost
	diffs := DiffWritesets(oldWS, newWS)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %+v", diffs)
	}
	if diffs[0].New != absent {
		t.Errorf("expected new side absent, got %+v", diffs[0])
	}
	if VerdictOf(nil, diffs) != WritesetDiff {
		t.Errorf("expected WRITESET_DIFF verdict")
	}
}

func TestWritesetDetectsWrongValue(t *testing.T) {
	oldWS := NormalizeWriteset(chargeChangesOld(), noiseCols, nil)
	bad := chargeChangesNew()
	bad[1].After["balance"] = dec("56000") // wrong computation
	newWS := NormalizeWriteset(bad, noiseCols, nil)
	diffs := DiffWritesets(oldWS, newWS)
	found := false
	for _, d := range diffs {
		if strings.Contains(d.Path, "balance") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected balance diff, got %+v", diffs)
	}
}

func TestNoiseOnlyUpdateIsDropped(t *testing.T) {
	touch := []writeset.RowChange{{
		Table: "demo.wallet", Op: "UPDATE",
		Before: map[string]any{"id": int64(1), "balance": dec("50000"), "updated_at": "2026-07-14 10:00:00"},
		After:  map[string]any{"id": int64(1), "balance": dec("50000"), "updated_at": "2026-07-14 10:00:05"},
	}}
	if ws := NormalizeWriteset(touch, noiseCols, nil); len(ws) != 0 {
		t.Errorf("expected empty write-set, got %+v", ws)
	}
}

func TestDecimalScaleInsensitive(t *testing.T) {
	a := []writeset.RowChange{{Table: "demo.wallet", Op: "UPDATE",
		Before: map[string]any{"id": int64(1), "balance": dec("50000")},
		After:  map[string]any{"id": int64(1), "balance": dec("55000")}}}
	b := []writeset.RowChange{{Table: "demo.wallet", Op: "UPDATE",
		Before: map[string]any{"id": int64(1), "balance": dec("50000.0")},
		After:  map[string]any{"id": int64(1), "balance": dec("55000.0")}}}
	if diffs := DiffWritesets(NormalizeWriteset(a, nil, nil), NormalizeWriteset(b, nil, nil)); len(diffs) != 0 {
		t.Errorf("expected scale-insensitive match, got %+v", diffs)
	}
}

func TestIgnoredTables(t *testing.T) {
	changes := []writeset.RowChange{{
		Table: "demo._replay_marker", Op: "INSERT",
		After: map[string]any{"id": int64(9), "rid": int64(1)},
	}}
	if ws := NormalizeWriteset(changes, nil, []string{"_replay_marker"}); len(ws) != 0 {
		t.Errorf("expected ignored, got %+v", ws)
	}
}

func TestVerdicts(t *testing.T) {
	resp := DiffResponses(200, `{"a":1}`, 200, `{"a":2}`, nil, nil, Options{})
	if VerdictOf(resp, nil) != ResponseDiff {
		t.Errorf("expected RESPONSE_DIFF")
	}
	if VerdictOf(nil, nil) != Match {
		t.Errorf("expected MATCH")
	}
}

func TestSortArraysByKey(t *testing.T) {
	// mapper-dependent list order: same rows, different order
	old := `{"items": [{"id": 1, "v": "a"}, {"id": 2, "v": "b"}, {"id": 3, "v": "c"}]}`
	new := `{"items": [{"id": 3, "v": "c"}, {"id": 1, "v": "a"}, {"id": 2, "v": "b"}]}`

	// without a sort rule: order difference reads as a diff
	if diffs := DiffResponses(200, old, 200, new, nil, nil, Options{}); len(diffs) == 0 {
		t.Fatalf("expected order-sensitive diff without sort rule")
	}
	// with a sort rule: equivalent
	opts := Options{SortArrays: []SortRule{{Path: "items", By: "id"}}}
	if diffs := DiffResponses(200, old, 200, new, nil, nil, opts); len(diffs) != 0 {
		t.Errorf("expected match with sort rule, got %+v", diffs)
	}
}

func TestSortArraysCanonicalFallbackAndNested(t *testing.T) {
	old := `{"data": {"tags": ["b", "a", "c"]}}`
	new := `{"data": {"tags": ["c", "b", "a"]}}`
	opts := Options{SortArrays: []SortRule{{Path: "**", By: "$canonical"}}}
	if diffs := DiffResponses(200, old, 200, new, nil, nil, opts); len(diffs) != 0 {
		t.Errorf("expected match with canonical sort, got %+v", diffs)
	}
	// real value difference still detected after sorting
	bad := `{"data": {"tags": ["c", "b", "x"]}}`
	if diffs := DiffResponses(200, old, 200, bad, nil, nil, opts); len(diffs) == 0 {
		t.Errorf("expected diff for changed element")
	}
}

func TestSortArraysFirstRuleWins(t *testing.T) {
	old := `{"items": [{"id": 2, "rank": 1}, {"id": 1, "rank": 2}]}`
	new := `{"items": [{"id": 1, "rank": 2}, {"id": 2, "rank": 1}]}`
	opts := Options{SortArrays: []SortRule{
		{Path: "items", By: "id"},
		{Path: "**", By: "rank"}, // must not shadow the first rule
	}}
	if diffs := DiffResponses(200, old, 200, new, nil, nil, opts); len(diffs) != 0 {
		t.Errorf("expected match, got %+v", diffs)
	}
}
