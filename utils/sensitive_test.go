package utils

import "testing"

func TestFilterLowValueEmptyBearer(t *testing.T) {
	in := []SensitiveData{
		{Content: `Authorization: "Bearer "`},
		{Content: `Authorization="Bearer "`},
		{Content: `Authorization: "Bearer abc123"`},
		{Content: `AUTHORIZATION    Bearer   `},
		{Content: `Authorization: Bearer token-xyz`},
	}
	out := filterLowValue(in)
	if len(out) != 2 {
		t.Fatalf("expected 2 entries kept, got %d", len(out))
	}
	if out[0].Content != `Authorization: "Bearer abc123"` {
		t.Fatalf("unexpected first entry: %q", out[0].Content)
	}
	if out[1].Content != `Authorization: Bearer token-xyz` {
		t.Fatalf("unexpected second entry: %q", out[1].Content)
	}
}
