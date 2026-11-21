package main

import (
	"bytes"
	"encoding/json/v2"
	"strings"
	"testing"
)

// TestEmitJSON_CarriesAllIssueClasses pins the --json wire shape: every issue
// class the text path gates on (broken refs, nav, arity, dead module citations)
// is present in the machine output, and any non-empty class flips valid=false.
// Found as C27 follow-through: the dead-citation class was text-log + exit
// code only, invisible to JSON/CI-annotation consumers.
func TestEmitJSON_CarriesAllIssueClasses(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	err := emitJSON(
		&buf,
		3, 100,
		[]brokenRef{{File: "a.md", Line: 1, Pkg: "event", Symbol: "New"}},
		[]string{},
		[]string{},
		[]navIssue{{File: "a.md", Line: 2, Msg: "broken anchor"}},
		[]navIssue{{File: "a.md", Line: 3, Msg: "wrong arity"}},
		[]navIssue{{File: "a.md", Line: 4, Msg: "cites deleted module"}},
		&resolver{clauses: map[string]string{}},
	)
	if err != nil {
		t.Fatalf("emitJSON: %v", err)
	}

	var doc struct {
		Valid         bool       `json:"valid"`
		NavIssues     []navIssue `json:"navIssues"`
		ArityIssues   []navIssue `json:"arityIssues"`
		ModPathIssues []navIssue `json:"modPathIssues"`
		Broken        []struct{} `json:"broken"`
		Warnings      []string   `json:"warnings"`
		Ambiguities   []string   `json:"ambiguities"`
	}
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("decode summary: %v\nraw: %s", err, buf.String())
	}

	if doc.Valid {
		t.Error("a run with a dead-citation issue must report valid=false")
	}

	if len(doc.ModPathIssues) != 1 || doc.ModPathIssues[0].Msg != "cites deleted module" {
		t.Errorf("modPathIssues must carry the dead-citation class, got %+v", doc.ModPathIssues)
	}

	if len(doc.NavIssues) != 1 || len(doc.ArityIssues) != 1 || len(doc.Broken) != 1 {
		t.Errorf("other classes must survive the change, got nav=%d arity=%d broken=%d",
			len(doc.NavIssues), len(doc.ArityIssues), len(doc.Broken))
	}
}

// TestEmitJSON_EmptyClassesRenderAsArrays pins the nil-to-empty-array
// normalization: downstream consumers index these fields without nil checks.
func TestEmitJSON_EmptyClassesRenderAsArrays(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	err := emitJSON(
		&buf,
		1, 10,
		nil, nil, nil, nil, nil, nil,
		&resolver{clauses: map[string]string{}},
	)
	if err != nil {
		t.Fatalf("emitJSON: %v", err)
	}

	for _, field := range []string{
		`"broken":[]`, `"navIssues":[]`, `"arityIssues":[]`,
		`"modPathIssues":[]`, `"warnings":[]`, `"ambiguities":[]`,
	} {
		if !strings.Contains(buf.String(), field) {
			t.Errorf("summary must normalize nil %s to [], got:\n%s", field, buf.String())
		}
	}

	if !strings.Contains(buf.String(), `"valid":true`) {
		t.Errorf("a clean run must report valid=true, got:\n%s", buf.String())
	}
}
