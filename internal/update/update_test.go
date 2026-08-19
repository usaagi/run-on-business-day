package update

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestAssetName(t *testing.T) {
	got := assetName()
	want := repoName + "-" + runtime.GOOS + "-" + runtime.GOARCH
	if runtime.GOOS == "windows" {
		want += ".exe"
	}
	if got != want {
		t.Errorf("assetName() = %q, want %q", got, want)
	}
}

func TestFindAssetURL(t *testing.T) {
	assets := []githubAsset{
		{Name: "run-on-business-day-linux-amd64", BrowserDownloadURL: "https://example.com/linux-amd64"},
		{Name: "run-on-business-day-windows-amd64.exe", BrowserDownloadURL: "https://example.com/windows-amd64"},
	}

	tests := []struct {
		name     string
		target   string
		wantURL  string
		wantOK   bool
		useEmpty bool
	}{
		{name: "一致するアセットあり", target: "run-on-business-day-linux-amd64", wantURL: "https://example.com/linux-amd64", wantOK: true},
		{name: "一致するアセットなし", target: "run-on-business-day-darwin-amd64", wantURL: "", wantOK: false},
		{name: "空スライス", target: "run-on-business-day-linux-amd64", wantURL: "", wantOK: false, useEmpty: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input := assets
			if tc.useEmpty {
				input = nil
			}
			gotURL, gotOK := findAssetURL(input, tc.target)
			if gotURL != tc.wantURL || gotOK != tc.wantOK {
				t.Errorf("findAssetURL() = (%q, %v), want (%q, %v)", gotURL, gotOK, tc.wantURL, tc.wantOK)
			}
		})
	}
}

func TestParseChecksums(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  map[string]string
	}{
		{
			name:  "sha256sum の標準的な出力",
			input: "aaaa  run-on-business-day-linux-amd64\nbbbb  run-on-business-day-linux-arm64\n",
			want: map[string]string{
				"run-on-business-day-linux-amd64": "aaaa",
				"run-on-business-day-linux-arm64": "bbbb",
			},
		},
		{
			name:  "バイナリモードのアスタリスク付き",
			input: "cccc *run-on-business-day-windows-amd64.exe\n",
			want:  map[string]string{"run-on-business-day-windows-amd64.exe": "cccc"},
		},
		{
			name:  "大文字のハッシュは小文字に正規化される",
			input: "ABCD  run-on-business-day-linux-amd64\n",
			want:  map[string]string{"run-on-business-day-linux-amd64": "abcd"},
		},
		{
			name:  "空行と不正な行は無視される",
			input: "\n   \ndddd  ok-file\nこれは不正な行です\n",
			want:  map[string]string{"ok-file": "dddd"},
		},
		{
			name:  "CRLF 改行",
			input: "eeee  ok-file\r\n",
			want:  map[string]string{"ok-file": "eeee"},
		},
		{
			name:  "空入力",
			input: "",
			want:  map[string]string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseChecksums([]byte(tc.input))
			if len(got) != len(tc.want) {
				t.Fatalf("件数が異なります: got %d (%v), want %d (%v)", len(got), got, len(tc.want), tc.want)
			}
			for name, want := range tc.want {
				if got[name] != want {
					t.Errorf("%q: got %q, want %q", name, got[name], want)
				}
			}
		})
	}
}

func TestFileSHA256(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		// SHA-256 の既知値
		{name: "空ファイル", content: "", want: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
		{name: "abc", content: "abc", want: "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "target.bin")
			if err := os.WriteFile(path, []byte(tc.content), 0600); err != nil {
				t.Fatalf("テスト用ファイルの作成に失敗: %v", err)
			}
			got, err := fileSHA256(path)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("fileSHA256() = %q, want %q", got, tc.want)
			}
		})
	}

	t.Run("存在しないファイル", func(t *testing.T) {
		if _, err := fileSHA256(filepath.Join(t.TempDir(), "missing.bin")); err == nil {
			t.Error("存在しないファイルでエラーが返りませんでした")
		}
	})
}

func TestVerifyChecksum(t *testing.T) {
	path := filepath.Join(t.TempDir(), "target.bin")
	if err := os.WriteFile(path, []byte("abc"), 0600); err != nil {
		t.Fatalf("テスト用ファイルの作成に失敗: %v", err)
	}
	const abcSum = "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"

	tests := []struct {
		name    string
		want    string
		wantErr bool
	}{
		{name: "一致", want: abcSum, wantErr: false},
		{name: "大文字の期待値でも一致", want: "BA7816BF8F01CFEA414140DE5DAE2223B00361A396177A9CB410FF61F20015AD", wantErr: false},
		{name: "不一致", want: "0000000000000000000000000000000000000000000000000000000000000000", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := verifyChecksum(path, tc.want)
			if (err != nil) != tc.wantErr {
				t.Errorf("verifyChecksum() error = %v, wantErr = %v", err, tc.wantErr)
			}
		})
	}
}
