package main

import (
	"encoding/json/v2"
	"testing"
)

func TestStripJSONComments_LineComments(t *testing.T) {
	t.Parallel()

	input := []byte(`{
  // This is a line comment
  "preset": "production",
  "min-severity": "warning" // trailing comment
}`)
	stripped := stripJSONComments(input)

	var m map[string]any
	if err := json.Unmarshal(stripped, &m); err != nil {
		t.Fatalf("stripped result is not valid JSON: %v\n%s", err, stripped)
	}

	if m["preset"] != "production" {
		t.Errorf("preset = %v, want production", m["preset"])
	}
	if m["min-severity"] != "warning" {
		t.Errorf("min-severity = %v, want warning", m["min-severity"])
	}
}

func TestStripJSONComments_BlockComments(t *testing.T) {
	t.Parallel()

	input := []byte(`{
  /* Block comment
     spanning multiple lines */
  "preset": "library",
  "rules": {
    /* Disable D002 */
    "disable": ["D002"]
  }
}`)
	stripped := stripJSONComments(input)

	var m map[string]any
	if err := json.Unmarshal(stripped, &m); err != nil {
		t.Fatalf("stripped result is not valid JSON: %v\n%s", err, stripped)
	}

	if m["preset"] != "library" {
		t.Errorf("preset = %v, want library", m["preset"])
	}
}

func TestStripJSONComments_UrlsInStrings(t *testing.T) {
	t.Parallel()

	input := []byte(`{
  // Comment before
  "url": "https://example.com/path",
  "path2": "http://other.com"
}`)
	stripped := stripJSONComments(input)

	var m map[string]any
	if err := json.Unmarshal(stripped, &m); err != nil {
		t.Fatalf("stripped result is not valid JSON: %v\n%s", err, stripped)
	}

	if m["url"] != "https://example.com/path" {
		t.Errorf("url = %v, want https://example.com/path", m["url"])
	}
	if m["path2"] != "http://other.com" {
		t.Errorf("path2 = %v, want http://other.com", m["path2"])
	}
}

func TestStripJSONComments_SlashInsideString(t *testing.T) {
	t.Parallel()

	input := []byte(`{
  "regex": "a/b/c",
  "comment": "//not-a-comment"
}`)
	stripped := stripJSONComments(input)

	var m map[string]any
	if err := json.Unmarshal(stripped, &m); err != nil {
		t.Fatalf("stripped result is not valid JSON: %v\n%s", err, stripped)
	}

	if m["regex"] != "a/b/c" {
		t.Errorf("regex = %v, want a/b/c", m["regex"])
	}
	if m["comment"] != "//not-a-comment" {
		t.Errorf("comment = %v, want //not-a-comment", m["comment"])
	}
}

func TestStripJSONComments_NoComments(t *testing.T) {
	t.Parallel()

	input := []byte(`{"preset":"production","min-severity":"warning"}`)
	stripped := stripJSONComments(input)

	var m map[string]any
	if err := json.Unmarshal(stripped, &m); err != nil {
		t.Fatalf("stripped result is not valid JSON: %v\n%s", err, stripped)
	}

	if m["preset"] != "production" {
		t.Errorf("preset = %v, want production", m["preset"])
	}
}

func TestStripJSONComments_EscapedQuotes(t *testing.T) {
	t.Parallel()

	input := []byte(`{
  // Comment
  "text": "He said \"hello //world\""
}`)
	stripped := stripJSONComments(input)

	var m map[string]any
	if err := json.Unmarshal(stripped, &m); err != nil {
		t.Fatalf("stripped result is not valid JSON: %v\n%s", err, stripped)
	}

	if m["text"] != "He said \"hello //world\"" {
		t.Errorf("text = %v, want He said \"hello //world\"", m["text"])
	}
}
