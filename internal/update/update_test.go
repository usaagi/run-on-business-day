package update

import (
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
