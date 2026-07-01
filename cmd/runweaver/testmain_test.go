package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "runweaver-cli-test-bin-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)
	if err := os.Setenv("RUNWEAVER_BIN", writeRunWeaverTestBin(dir)); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

func writeRunWeaverTestBin(dir string) string {
	path := filepath.Join(dir, "runweaver")
	content := "#!/bin/sh\nif [ \"$1\" = \"help\" ]; then\nprintf '%s\\n' 'start --repo <path>' 'status --repo <path>' 'context query --task <text>'\nexit 0\nfi\nexit 0\n"
	if runtime.GOOS == "windows" {
		path += ".cmd"
		content = "@echo off\r\nif \"%1\"==\"help\" (\r\necho start --repo ^<path^>\r\necho status --repo ^<path^>\r\necho context query --task ^<text^>\r\nexit /b 0\r\n)\r\nexit /b 0\r\n"
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		panic(err)
	}
	return path
}
