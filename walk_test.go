package fspath

import (
	"net/http"
	"os"
	"testing"
)

func TestWalk(t *testing.T) {
	expected := []string{
		"bar/bar.txt",
		"foo/baz/baz.txt",
		"foo/foo/fred.txt",
		"foo/foo.txt",
		"foo.txt",
		"qux.txt",
	}
	results := make([]string, 0, len(expected))
	fs := http.Dir("fs")
	err := Walk(fs, ".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		results = append(results, path)
		return nil
	})
	if err != nil {
		t.Error(err)
	}
	if len(expected) != len(results) {
		t.Error("Expected", len(expected), "files, got", len(results))
		return
	}
	for i, name := range expected {
		if name != results[i] {
			t.Error("Expected", name, "got", results[i])
		}
	}
}
