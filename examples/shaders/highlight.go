package main

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// A small WGSL tokenizer for the source pane. It knows enough to color
// comments, keywords, types, numbers, attributes and calls; everything
// else is plain text. It is not a parser and never rejects input.

type tokenKind int

const (
	tokText tokenKind = iota
	tokComment
	tokKeyword
	tokType
	tokNumber
	tokAttr
	tokCall
)

type token struct {
	kind tokenKind
	text string
}

var wgslKeywords = map[string]bool{
	"fn": true, "let": true, "var": true, "const": true, "return": true, "if": true, "else": true,
	"for": true, "loop": true, "while": true, "break": true, "continue": true, "struct": true,
	"switch": true, "case": true, "default": true, "discard": true, "true": true, "false": true,
	"uniform": true, "storage": true, "read": true, "read_write": true, "private": true, "workgroup": true,
}

var wgslTypes = map[string]bool{
	"f32": true, "f16": true, "u32": true, "i32": true, "bool": true,
	"array": true, "atomic": true, "ptr": true, "sampler": true,
}

func isWGSLType(word string) bool {
	return wgslTypes[word] || strings.HasPrefix(word, "vec") || strings.HasPrefix(word, "mat") || strings.HasPrefix(word, "texture_")
}

func isIdentStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}
func isIdent(c byte) bool { return isIdentStart(c) || (c >= '0' && c <= '9') }
func isDigit(c byte) bool { return c >= '0' && c <= '9' }

// tokenizeWGSL splits one source line into tokens whose texts concatenate
// back to the line. inBlock says whether the line starts inside a block
// comment; the returned bool says whether one is still open at the end.
func tokenizeWGSL(line string, inBlock bool) ([]token, bool) {
	var toks []token
	var plain strings.Builder
	emit := func(kind tokenKind, text string) {
		if text != "" {
			toks = append(toks, token{kind, text})
		}
	}
	flush := func() {
		emit(tokText, plain.String())
		plain.Reset()
	}
	i := 0
	for i < len(line) {
		if inBlock {
			flush()
			end := strings.Index(line[i:], "*/")
			if end < 0 {
				emit(tokComment, line[i:])
				return toks, true
			}
			emit(tokComment, line[i:i+end+2])
			i += end + 2
			inBlock = false
			continue
		}
		c := line[i]
		rest := line[i:]
		switch {
		case strings.HasPrefix(rest, "//"):
			flush()
			emit(tokComment, rest)
			return toks, false
		case strings.HasPrefix(rest, "/*"):
			flush()
			end := strings.Index(rest[2:], "*/")
			if end < 0 {
				emit(tokComment, rest)
				return toks, true
			}
			emit(tokComment, rest[:end+4])
			i += end + 4
		case c == '@' && i+1 < len(line) && isIdentStart(line[i+1]):
			flush()
			j := i + 1
			for j < len(line) && isIdent(line[j]) {
				j++
			}
			emit(tokAttr, line[i:j])
			i = j
		case isDigit(c):
			flush()
			j := i
			for j < len(line) {
				d := line[j]
				exponentSign := (d == '+' || d == '-') && (line[j-1] == 'e' || line[j-1] == 'E')
				if !isDigit(d) && d != '.' && d != 'e' && d != 'E' && !exponentSign {
					break
				}
				j++
			}
			if j < len(line) && strings.IndexByte("uifh", line[j]) >= 0 {
				j++
			}
			emit(tokNumber, line[i:j])
			i = j
		case isIdentStart(c):
			flush()
			j := i
			for j < len(line) && isIdent(line[j]) {
				j++
			}
			word := line[i:j]
			kind := tokText
			switch {
			case wgslKeywords[word]:
				kind = tokKeyword
			case isWGSLType(word):
				kind = tokType
			case j < len(line) && line[j] == '(':
				kind = tokCall
			}
			emit(kind, word)
			i = j
		default:
			plain.WriteByte(c)
			i++
		}
	}
	flush()
	return toks, inBlock
}

var tokenStyles = map[tokenKind]lipgloss.Style{
	tokComment: muted,
	tokKeyword: lipgloss.NewStyle().Foreground(lipgloss.Color("#c792ea")),
	tokType:    lipgloss.NewStyle().Foreground(lipgloss.Color("#82aaff")),
	tokNumber:  lipgloss.NewStyle().Foreground(lipgloss.Color("#f78c6c")),
	tokAttr:    lipgloss.NewStyle().Foreground(lipgloss.Color("#ffcb6b")),
	tokCall:    lipgloss.NewStyle().Foreground(lipgloss.Color("#70ead1")),
}

// highlightWGSL renders one line with colors, truncated to width cells with
// an ellipsis. It threads the block-comment state like tokenizeWGSL.
func highlightWGSL(line string, inBlock bool, width int) (string, bool) {
	tokens, open := tokenizeWGSL(line, inBlock)
	var sb strings.Builder
	for _, tok := range tokens {
		if style, ok := tokenStyles[tok.kind]; ok {
			sb.WriteString(style.Render(tok.text))
		} else {
			sb.WriteString(tok.text)
		}
	}
	return ansi.Truncate(sb.String(), width, "…"), open
}

const sourceDivider = "── common.wgsl · shared prelude ──"

// sourceLines returns the preset's own shader body, a divider, and then the
// shared prelude with the splice marker replaced by a note. The body comes
// first so it stays visible when the pane clips.
func sourceLines(index int) []string {
	body, err := shaderFiles.ReadFile("shaders/" + presets[index].name + ".wgsl")
	if err != nil {
		panic(err)
	}
	common, err := shaderFiles.ReadFile("shaders/common.wgsl")
	if err != nil {
		panic(err)
	}
	prelude := strings.Replace(string(common), "// SHADER_BODY", "// … the shader body above is spliced in here …", 1)
	lines := strings.Split(strings.TrimRight(string(body), "\n"), "\n")
	lines = append(lines, sourceDivider)
	return append(lines, strings.Split(strings.TrimRight(prelude, "\n"), "\n")...)
}

// renderSourcePane renders numbered, highlighted source for the preset,
// clipped to rows lines and width cells.
func renderSourcePane(index, width, rows int) string {
	lines := sourceLines(index)
	shown := lines
	if len(shown) > rows {
		shown = shown[:max(0, rows-1)]
	}
	out := make([]string, 0, rows)
	inBlock := false
	for i, line := range shown {
		var text string
		if line == sourceDivider {
			text = muted.Render(ansi.Truncate(line, width-4, ""))
		} else {
			text, inBlock = highlightWGSL(line, inBlock, width-4)
		}
		out = append(out, muted.Render(fmt.Sprintf("%3d ", i+1))+text)
	}
	if len(shown) < len(lines) {
		out = append(out, muted.Render(fmt.Sprintf("    (+%d more lines)", len(lines)-len(shown))))
	}
	return lipgloss.NewStyle().Width(width).Height(rows).Render(strings.Join(out, "\n"))
}
