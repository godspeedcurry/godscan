package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestExpandUF(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		out  []string
	}{
		{"separate", []string{"godscan", "-uf", "urls.txt"}, []string{"godscan", "-f", "urls.txt"}},
		{"equals", []string{"godscan", "-uf=urls.txt"}, []string{"godscan", "-f", "urls.txt"}},
		{"compact", []string{"godscan", "-ufurls.txt"}, []string{"godscan", "-f", "urls.txt"}},
		{"nochange", []string{"godscan", "-u", "https://a.com"}, []string{"godscan", "-u", "https://a.com"}},
	}
	for _, tt := range tests {
		expanded := expandUF(tt.in)
		if len(expanded) != len(tt.out) {
			t.Fatalf("%s: len mismatch got %v want %v", tt.name, expanded, tt.out)
		}
		for i := range expanded {
			if expanded[i] != tt.out[i] {
				t.Fatalf("%s: idx %d got %q want %q", tt.name, i, expanded[i], tt.out[i])
			}
		}
	}
}

func TestNormalizeOutputPaths(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	tmp := t.TempDir()
	outDir := filepath.Join(tmp, "out")
	viper.Set("output-dir", outDir)
	viper.Set("output", "result.log")

	normalizeOutputPaths()

	if _, err := os.Stat(outDir); err != nil {
		t.Fatalf("output dir not created: %v", err)
	}
	wantOut := filepath.Join(outDir, "result.log")
	if got := viper.GetString("output"); got != wantOut {
		t.Fatalf("output path got %s want %s", got, wantOut)
	}

	// absolute path should stay absolute
	absOut := filepath.Join(tmp, "abs.log")
	viper.Set("output", absOut)
	normalizeOutputPaths()
	if got := viper.GetString("output"); got != absOut {
		t.Fatalf("abs output path changed: %s", got)
	}
}
