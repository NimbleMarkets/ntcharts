package main

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func kindsOf(t *testing.T, line string) map[string]tokenKind {
	t.Helper()
	tokens, _ := tokenizeWGSL(line, false)
	var rebuilt strings.Builder
	out := map[string]tokenKind{}
	for _, tok := range tokens {
		rebuilt.WriteString(tok.text)
		out[tok.text] = tok.kind
	}
	if rebuilt.String() != line {
		t.Fatalf("tokens do not round-trip: %q != %q", rebuilt.String(), line)
	}
	return out
}

func TestTokenizeWGSLClassifies(t *testing.T) {
	k := kindsOf(t, "fn shade(uv: vec2<f32>) -> vec3<f32> { let c=cos(a); return 0.5*c; }")
	want := map[string]tokenKind{
		"fn": tokKeyword, "let": tokKeyword, "return": tokKeyword,
		"shade": tokCall, "cos": tokCall,
		"vec2": tokType, "vec3": tokType, "f32": tokType,
		"0.5": tokNumber,
		"uv":  tokText, "c": tokText,
	}
	for text, kind := range want {
		if got, ok := k[text]; !ok || got != kind {
			t.Errorf("%q = %v (present=%v), want %v", text, got, ok, kind)
		}
	}
}

func TestTokenizeWGSLAttributesNumbersComments(t *testing.T) {
	k := kindsOf(t, "@compute @workgroup_size(8,8,1) // per tile")
	if k["@compute"] != tokAttr || k["@workgroup_size"] != tokAttr {
		t.Errorf("attributes: %v", k)
	}
	if k["8"] != tokNumber || k["1"] != tokNumber {
		t.Errorf("integer literals: %v", k)
	}
	if k["// per tile"] != tokComment {
		t.Errorf("line comment: %v", k)
	}
	k = kindsOf(t, "pixels[i]=rgb.x|(rgb.y<<8u)|(255u<<24u);")
	if k["8u"] != tokNumber || k["255u"] != tokNumber || k["24u"] != tokNumber {
		t.Errorf("suffixed literals: %v", k)
	}
	if k["rgb"] != tokText || k["x"] != tokText {
		t.Errorf("member access should be plain text: %v", k)
	}
}

func TestTokenizeWGSLBlockCommentSpansLines(t *testing.T) {
	tokens, open := tokenizeWGSL("let a=1; /* start", false)
	if !open || tokens[len(tokens)-1].kind != tokComment || tokens[len(tokens)-1].text != "/* start" {
		t.Fatalf("block comment did not open: %+v open=%v", tokens, open)
	}
	tokens, open = tokenizeWGSL("end */ let b=2;", true)
	if open || tokens[0].kind != tokComment || tokens[0].text != "end */" {
		t.Fatalf("block comment did not close: %+v open=%v", tokens, open)
	}
	found := false
	for _, tok := range tokens {
		if tok.text == "let" && tok.kind == tokKeyword {
			found = true
		}
	}
	if !found {
		t.Fatal("code after the block comment was not tokenized")
	}
}

func TestHighlightWGSLTruncatesToWidth(t *testing.T) {
	line := "let ring=length(vec2<f32>(length(q.xy)-0.85,q.z))-0.18; // a long comment that runs on"
	out, _ := highlightWGSL(line, false, 24)
	if w := ansi.StringWidth(out); w > 24 {
		t.Fatalf("highlighted width %d exceeds 24", w)
	}
	if !strings.HasSuffix(ansi.Strip(out), "…") {
		t.Fatalf("truncated line lacks ellipsis: %q", ansi.Strip(out))
	}
	short, _ := highlightWGSL("let x=1;", false, 24)
	if ansi.Strip(short) != "let x=1;" {
		t.Fatalf("short line altered: %q", ansi.Strip(short))
	}
}
