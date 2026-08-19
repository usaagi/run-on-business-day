package update

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

// newChecksumServer は SHA256SUMS とバイナリを配信するテスト用サーバを立て、
// (SHA256SUMS の URL, バイナリの URL) を返します。
func newChecksumServer(t *testing.T, assetName string, body []byte) (sumsURL, binURL string) {
	t.Helper()

	sum := sha256.Sum256(body)
	sums := hex.EncodeToString(sum[:]) + "  " + assetName + "\n"

	mux := http.NewServeMux()
	mux.HandleFunc("/SHA256SUMS", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(sums))
	})
	mux.HandleFunc("/asset", func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	})
	mux.HandleFunc("/notfound", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	return srv.URL + "/SHA256SUMS", srv.URL + "/asset"
}

func TestFetchChecksums(t *testing.T) {
	body := []byte("dummy binary contents")
	sumsURL, _ := newChecksumServer(t, "run-on-business-day-linux-amd64", body)

	t.Run("正常に取得・解析できる", func(t *testing.T) {
		got, err := fetchChecksums(http.DefaultClient, sumsURL)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		sum := sha256.Sum256(body)
		want := hex.EncodeToString(sum[:])
		if got["run-on-business-day-linux-amd64"] != want {
			t.Errorf("got %q, want %q", got["run-on-business-day-linux-amd64"], want)
		}
	})

	t.Run("404 はエラーになる", func(t *testing.T) {
		notFound := sumsURL[:len(sumsURL)-len("/SHA256SUMS")] + "/notfound"
		if _, err := fetchChecksums(http.DefaultClient, notFound); err == nil {
			t.Error("404 でエラーが返りませんでした")
		}
	})
}

func TestVerifyDownload(t *testing.T) {
	const target = "run-on-business-day-linux-amd64"
	body := []byte("dummy binary contents")
	sumsURL, binURL := newChecksumServer(t, target, body)

	writeTemp := func(t *testing.T, content []byte) string {
		t.Helper()
		path := filepath.Join(t.TempDir(), "downloaded.bin")
		if err := os.WriteFile(path, content, 0600); err != nil {
			t.Fatalf("テスト用ファイルの作成に失敗: %v", err)
		}
		return path
	}

	assets := []githubAsset{
		{Name: target, BrowserDownloadURL: binURL},
		{Name: checksumsAssetName, BrowserDownloadURL: sumsURL},
	}

	t.Run("内容が一致すれば成功する", func(t *testing.T) {
		if err := verifyDownload(assets, target, writeTemp(t, body)); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("内容が改ざんされていれば失敗する", func(t *testing.T) {
		err := verifyDownload(assets, target, writeTemp(t, []byte("tampered contents")))
		if err == nil {
			t.Fatal("改ざんされたファイルが検証を通過しました")
		}
		if !strings.Contains(err.Error(), "チェックサムが一致しません") {
			t.Errorf("想定外のエラーです: %v", err)
		}
	})

	t.Run("SHA256SUMS が無ければ失敗する", func(t *testing.T) {
		without := []githubAsset{{Name: target, BrowserDownloadURL: binURL}}
		if err := verifyDownload(without, target, writeTemp(t, body)); err == nil {
			t.Error("SHA256SUMS 不在で検証が通過しました")
		}
	})

	t.Run("SHA256SUMS に対象のエントリが無ければ失敗する", func(t *testing.T) {
		err := verifyDownload(assets, "run-on-business-day-linux-arm64", writeTemp(t, body))
		if err == nil {
			t.Error("エントリ不在で検証が通過しました")
		}
	})
}
