package content

import (
	"embed"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/lovyou-ai/site/views"
)

//go:embed reference/primitives.md
var primitivesRaw []byte

//go:embed reference/agent-primitives.md
var agentPrimitivesRaw []byte

var (
	layerHeading = regexp.MustCompile(`^## Layer (\d+): (.+)`)
	groupHeading = regexp.MustCompile(`^### Group \d+ — (.+)`)
	primHeading  = regexp.MustCompile(`^#### (.+)`)
	gapLine      = regexp.MustCompile(`^\*\*Gap from Layer \d+:\*\* (.+)`)
	transLine    = regexp.MustCompile(`^\*\*Transition:\*\* (.+)`)
	tableRow     = regexp.MustCompile(`^\| \*\*(.+?)\*\* \| (.+) \|$`)
)

// LoadLayers parses primitives.md into Layer structs with their primitives.
func LoadLayers() []views.Layer {
	lines := strings.Split(string(primitivesRaw), "\n")

	var layers []views.Layer
	var curLayer *views.Layer
	var curGroup string
	var curPrim *views.Primitive
	var primDesc []string
	inPrimitive := false

	flushPrimitive := func() {
		if curPrim != nil && curLayer != nil {
			curLayer.Primitives = append(curLayer.Primitives, *curPrim)
			curPrim = nil
		}
		primDesc = nil
		inPrimitive = false
	}

	flushLayer := func() {
		flushPrimitive()
		if curLayer != nil {
			layers = append(layers, *curLayer)
			curLayer = nil
		}
	}

	for _, line := range lines {
		line = strings.TrimRight(line, "\r")

		// Layer heading (skip Layer 0 — parsed separately from summary table).
		if m := layerHeading.FindStringSubmatch(line); m != nil {
			flushLayer()
			n, _ := strconv.Atoi(m[1])
			if n == 0 {
				continue
			}
			curLayer = &views.Layer{Number: n, Name: m[2]}
			curGroup = ""
			continue
		}

		// Group heading.
		if m := groupHeading.FindStringSubmatch(line); m != nil {
			flushPrimitive()
			curGroup = m[1]
			continue
		}

		// Primitive heading.
		if m := primHeading.FindStringSubmatch(line); m != nil {
			flushPrimitive()
			inPrimitive = true
			name := m[1]
			curPrim = &views.Primitive{
				Name:  name,
				Slug:  slugify(name),
				Group: curGroup,
			}
			if curLayer != nil {
				curPrim.Layer = curLayer.Number
				curPrim.LayerName = curLayer.Name
			}
			continue
		}

		if curLayer != nil && !inPrimitive {
			// Layer metadata lines.
			if m := gapLine.FindStringSubmatch(line); m != nil {
				curLayer.Gap = m[1]
			}
			if m := transLine.FindStringSubmatch(line); m != nil {
				curLayer.Transition = m[1]
			}
		}

		if curPrim != nil {
			// Table rows for primitive spec.
			if m := tableRow.FindStringSubmatch(line); m != nil {
				key := m[1]
				val := strings.TrimSpace(m[2])
				switch key {
				case "Subscribes to":
					curPrim.SubscribesTo = val
				case "Emits":
					curPrim.Emits = val
				case "Depends on":
					curPrim.DependsOn = val
				case "State":
					curPrim.State = val
				case "Intelligent", "Mechanical", "Both":
					curPrim.Intelligent = key + ": " + val
				}
				continue
			}
			// Skip empty lines and table headers.
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || trimmed == "| | |" || trimmed == "|---|---|" {
				continue
			}
			// Non-table, non-heading text is description or notes.
			if !strings.HasPrefix(trimmed, "##") && !strings.HasPrefix(trimmed, "---") && !strings.HasPrefix(trimmed, "**Full specification") {
				primDesc = append(primDesc, trimmed)
				if curPrim.Description == "" {
					curPrim.Description = trimmed
				}
			}
		}
	}
	flushLayer()

	// Parse Layer 0 separately — it's a summary table in primitives.md, not #### entries.
	layer0 := parseLayer0(lines)
	if layer0 != nil {
		// Prepend layer 0.
		layers = append([]views.Layer{*layer0}, layers...)
	}

	return layers
}

// parseLayer0 extracts Foundation primitives from the summary table.
func parseLayer0(lines []string) *views.Layer {
	layer := &views.Layer{
		Number:     0,
		Name:       "Foundation",
		Gap:        "None — this is the base layer.",
		Transition: "Nothing → Something",
	}

	// Layer 0 primitives are listed in a table: | Group | Primitives | Domain |
	inLayer0 := false
	l0Row := regexp.MustCompile(`^\| (\d+) — (.+?) \| (.+?) \| (.+?) \|$`)

	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		if strings.HasPrefix(line, "## Layer 0:") {
			inLayer0 = true
			continue
		}
		if inLayer0 && strings.HasPrefix(line, "## Layer") {
			break
		}
		if !inLayer0 {
			continue
		}

		if m := l0Row.FindStringSubmatch(line); m != nil {
			groupName := strings.TrimSpace(m[2])
			names := strings.TrimSpace(m[3])
			domain := strings.TrimSpace(m[4])

			for _, name := range strings.Split(names, ", ") {
				name = strings.TrimSpace(name)
				if name == "" {
					continue
				}
				layer.Primitives = append(layer.Primitives, views.Primitive{
					Name:        name,
					Slug:        slugify(name),
					Layer:       0,
					LayerName:   "Foundation",
					Group:       groupName,
					Description: domain,
				})
			}
		}
	}

	if len(layer.Primitives) == 0 {
		return nil
	}
	return layer
}

// LoadAgentPrimitives parses agent-primitives.md into primitives.
func LoadAgentPrimitives() []views.Primitive {
	lines := strings.Split(string(agentPrimitivesRaw), "\n")

	var prims []views.Primitive
	var category string

	// Agent primitives are in 7-column tables under ### Category (N) headings.
	catHeading := regexp.MustCompile(`^### (Structural|Operational|Relational|Modal) \((\d+)\)`)
	agentRow := regexp.MustCompile(`^\| \*\*(.+?)\*\* \| (.+?) \| (.+?) \| (.+?) \| (.+?) \| (.+?) \| (.+?) \|$`)

	for _, line := range lines {
		line = strings.TrimRight(line, "\r")

		if m := catHeading.FindStringSubmatch(line); m != nil {
			category = m[1]
			continue
		}

		if m := agentRow.FindStringSubmatch(line); m != nil {
			prims = append(prims, views.Primitive{
				Name:        m[1],
				Slug:        "agent-" + slugify(m[1]),
				Layer:       -1, // agent primitive, not a layer primitive
				LayerName:   "Agent",
				Group:       category,
				Description: m[2],
				// Store dimensional analysis in spec fields.
				SubscribesTo: "Direction: " + m[3],
				Emits:        "Timing: " + m[4],
				DependsOn:    "Mutability: " + m[5],
				State:        "Agency: " + m[6],
				Intelligent:  "Awareness: " + m[7],
			})
		}
	}

	return prims
}

func slugify(name string) string {
	s := strings.ToLower(name)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "/", "-")
	// Remove non-alphanumeric except hyphens.
	var b strings.Builder
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			b.WriteRune(c)
		}
	}
	return b.String()
}

// LoadGrammars reads composition grammar markdown files.
func LoadGrammars() ([]views.RefPage, error) {
	entries, err := grammarsFS.ReadDir("reference/grammars")
	if err != nil {
		return nil, fmt.Errorf("read grammars dir: %w", err)
	}

	var pages []views.RefPage
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}

		raw, err := grammarsFS.ReadFile("reference/grammars/" + e.Name())
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", e.Name(), err)
		}

		page, err := parseGrammarPage(e.Name(), raw)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", e.Name(), err)
		}
		pages = append(pages, page)
	}

	return pages, nil
}

//go:embed reference/grammars/*.md
var grammarsFS embed.FS
