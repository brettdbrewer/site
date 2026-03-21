package content

import (
	"bytes"
	"embed"
	"fmt"
	"sort"
	"strings"

	"github.com/lovyou-ai/site/views"
	"github.com/yuin/goldmark"
)

//go:embed reference/*.md
var referenceFS embed.FS

// LoadReference reads all embedded reference markdown and returns them in order.
func LoadReference() ([]views.RefPage, error) {
	entries, err := referenceFS.ReadDir("reference")
	if err != nil {
		return nil, fmt.Errorf("read reference dir: %w", err)
	}

	md := goldmark.New()
	var pages []views.RefPage

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}

		raw, err := referenceFS.ReadFile("reference/" + e.Name())
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", e.Name(), err)
		}

		page, err := parseRefPage(md, e.Name(), raw)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", e.Name(), err)
		}
		pages = append(pages, page)
	}

	sort.Slice(pages, func(i, j int) bool {
		return pages[i].Order < pages[j].Order
	})

	return pages, nil
}

func parseRefPage(md goldmark.Markdown, filename string, raw []byte) (views.RefPage, error) {
	lines := strings.SplitN(string(raw), "\n", -1)

	// Title from first # heading.
	var title string
	for _, l := range lines {
		if strings.HasPrefix(l, "# ") {
			title = strings.TrimPrefix(l, "# ")
			break
		}
	}

	// Summary: first non-empty line after the title that isn't a heading or separator.
	var summary string
	pastTitle := false
	for _, l := range lines {
		if strings.HasPrefix(l, "# ") {
			pastTitle = true
			continue
		}
		if !pastTitle {
			continue
		}
		l = strings.TrimSpace(l)
		if l == "" || l == "---" || strings.HasPrefix(l, "#") {
			continue
		}
		summary = l
		if len(summary) > 200 {
			summary = summary[:200] + "..."
		}
		break
	}

	// Order: parse from filename prefix (00-, 01-, etc.)
	order := 99
	slug := strings.TrimSuffix(filename, ".md")
	if len(slug) > 2 && slug[2] == '-' {
		n := 0
		for _, c := range slug[:2] {
			if c >= '0' && c <= '9' {
				n = n*10 + int(c-'0')
			}
		}
		order = n
		slug = slug[3:] // strip "NN-"
	} else if slug == "agent-primitives" {
		order = 14
	}

	// Convert body to HTML (skip title line).
	bodyStart := 0
	for i, l := range lines {
		if strings.HasPrefix(l, "# ") {
			bodyStart = i + 1
			break
		}
	}
	bodyMD := strings.Join(lines[bodyStart:], "\n")

	var buf bytes.Buffer
	if err := md.Convert([]byte(bodyMD), &buf); err != nil {
		return views.RefPage{}, fmt.Errorf("convert markdown: %w", err)
	}

	return views.RefPage{
		Slug:    slug,
		Title:   title,
		Summary: summary,
		Order:   order,
		Body:    buf.String(),
	}, nil
}
