package main

import (
	"bytes"
	"flag"
	"testing"
)

func TestParseOptions(t *testing.T) {
	t.Run("--check", func(t *testing.T) {
		var stderr bytes.Buffer
		opts, err := parseOptions([]string{"--check"}, &stderr)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !opts.check {
			t.Errorf("check = false, want true")
		}
	})

	t.Run("--force と -- 以降のコマンド引数", func(t *testing.T) {
		var stderr bytes.Buffer
		opts, err := parseOptions([]string{"--force", "--", "echo", "hi"}, &stderr)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !opts.force {
			t.Errorf("force = false, want true")
		}
		want := []string{"echo", "hi"}
		if len(opts.cmdArgs) != len(want) {
			t.Fatalf("cmdArgs = %v, want %v", opts.cmdArgs, want)
		}
		for i := range want {
			if opts.cmdArgs[i] != want[i] {
				t.Errorf("cmdArgs[%d] = %q, want %q", i, opts.cmdArgs[i], want[i])
			}
		}
	})

	t.Run("-C で workingDir 指定", func(t *testing.T) {
		var stderr bytes.Buffer
		opts, err := parseOptions([]string{"-C", "/tmp", "--", "ls"}, &stderr)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if opts.workingDir != "/tmp" {
			t.Errorf("workingDir = %q, want %q", opts.workingDir, "/tmp")
		}
	})

	t.Run("--cwd で workingDir 指定", func(t *testing.T) {
		var stderr bytes.Buffer
		opts, err := parseOptions([]string{"--cwd", "/tmp"}, &stderr)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if opts.workingDir != "/tmp" {
			t.Errorf("workingDir = %q, want %q", opts.workingDir, "/tmp")
		}
	})

	t.Run("--version", func(t *testing.T) {
		var stderr bytes.Buffer
		opts, err := parseOptions([]string{"--version"}, &stderr)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !opts.showVersion {
			t.Errorf("showVersion = false, want true")
		}
	})

	t.Run("不正なフラグはエラーを返す", func(t *testing.T) {
		var stderr bytes.Buffer
		_, err := parseOptions([]string{"--bogus"}, &stderr)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
	})

	t.Run("--help は flag.ErrHelp を返す", func(t *testing.T) {
		var stderr bytes.Buffer
		_, err := parseOptions([]string{"--help"}, &stderr)
		if err != flag.ErrHelp {
			t.Fatalf("err = %v, want flag.ErrHelp", err)
		}
	})

	t.Run("update サブコマンドは cmdArgs に残る", func(t *testing.T) {
		var stderr bytes.Buffer
		opts, err := parseOptions([]string{"update"}, &stderr)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(opts.cmdArgs) != 1 || opts.cmdArgs[0] != "update" {
			t.Errorf("cmdArgs = %v, want [update]", opts.cmdArgs)
		}
	})

	t.Run("引数なしは全フィールドゼロ値", func(t *testing.T) {
		var stderr bytes.Buffer
		opts, err := parseOptions([]string{}, &stderr)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if opts.force || opts.check || opts.showVersion || opts.workingDir != "" || len(opts.cmdArgs) != 0 {
			t.Errorf("opts = %+v, want zero values", opts)
		}
	})
}

func TestRunShowVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	opts := &options{showVersion: true}

	code := run(opts, &stdout, &stderr)

	if code != exitOK {
		t.Errorf("code = %d, want %d", code, exitOK)
	}
	if stdout.String() != version+"\n" {
		t.Errorf("stdout = %q, want %q", stdout.String(), version+"\n")
	}
}
