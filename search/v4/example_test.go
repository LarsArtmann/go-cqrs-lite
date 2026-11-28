package search_test

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/search/v4"
)

// Note is the record type the examples search.
type Note struct {
	ID     string
	Title  string
	Body   string
	Status string
}

func Example() {
	notes := search.Slice[Note]{
		Items: []Note{
			{ID: "n1", Title: "Big invoice", Body: "The March invoice for Acme", Status: "open"},
			{ID: "n2", Title: "Standup", Body: "Talk about the big invoice", Status: "open"},
			{ID: "n3", Title: "Holiday", Body: "Nothing to do", Status: "done"},
		},
		Identify: func(n Note) string { return n.ID },
		Match: func(n Note, filters url.Values) bool {
			return filters == nil || filters.Get("status") == "" || filters.Get("status") == n.Status
		},
	}

	spec := search.Spec[Note]{
		Fields: []search.Field[Note]{
			{Name: "title", Weight: 1, Value: func(n Note) string { return n.Title }},
			{Name: "body", Weight: 0.2, Value: func(n Note) string { return n.Body }},
		},
	}
	index, err := search.New(spec, search.NoEngine(), notes)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	result, err := index.Search(context.Background(), search.Request{Query: "big invoice"})
	if err != nil {
		fmt.Println("error:", err)

		return
	}
	fmt.Println("via:", result.Via, "total:", result.TotalCount)
	for _, hit := range result.Data {
		fmt.Println(hit.ID, marked(hit.Snippet))
	}
	// Output:
	// via: scan total: 2
	// n1 *Big* *invoice*
	// n2 Talk about the *big* *invoice*
}

// marked renders segments the way a template would: text as is, marked runs
// between asterisks.
func marked(segments []search.Segment) string {
	out := ""
	var outSb63 strings.Builder
	for _, segment := range segments {
		if segment.Marked {
			outSb63.WriteString("*" + segment.Text + "*")

			continue
		}
		outSb63.WriteString(segment.Text)
	}
	out += outSb63.String()

	return out
}
