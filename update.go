package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	repoOwner = "usaagi"
	repoName  = "run-on-business-day"
	apiURL    = "https://api.github.com/repos/" + repoOwner + "/" + repoName + "/releases/latest"
)

type githubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func assetName() string {
	base := repoName + "-" + runtime.GOOS + "-" + runtime.GOARCH
	if runtime.GOOS == "windows" {
		return base + ".exe"
	}
	return base
}

func runUpgrade() {
	fmt.Println("最新バージョンを確認しています...")

	resp, err := http.Get(apiURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: GitHub APIへの接続に失敗しました: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Fprintf(os.Stderr, "Error: GitHub APIがステータス %d を返しました\n", resp.StatusCode)
		os.Exit(1)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		fmt.Fprintf(os.Stderr, "Error: レスポンスの解析に失敗しました: %v\n", err)
		os.Exit(1)
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")
	if latestVersion == version {
		fmt.Printf("すでに最新バージョン (%s) です。\n", version)
		os.Exit(0)
	}

	fmt.Printf("新しいバージョンが見つかりました: %s → %s\n", version, latestVersion)

	// アセットを探す
	target := assetName()
	var downloadURL string
	for _, a := range release.Assets {
		if a.Name == target {
			downloadURL = a.BrowserDownloadURL
			break
		}
	}
	if downloadURL == "" {
		fmt.Fprintf(os.Stderr, "Error: お使いの環境 (%s/%s) 向けのバイナリ '%s' がリリースに見つかりません\n", runtime.GOOS, runtime.GOARCH, target)
		os.Exit(1)
	}

	// ダウンロード
	fmt.Printf("ダウンロード中: %s\n", target)
	dlResp, err := http.Get(downloadURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: ダウンロードに失敗しました: %v\n", err)
		os.Exit(1)
	}
	defer dlResp.Body.Close()

	if dlResp.StatusCode != 200 {
		fmt.Fprintf(os.Stderr, "Error: ダウンロードがステータス %d を返しました\n", dlResp.StatusCode)
		os.Exit(1)
	}

	// 一時ファイルに書き込み
	execPath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: 実行ファイルのパスを取得できません: %v\n", err)
		os.Exit(1)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: シンボリックリンクの解決に失敗しました: %v\n", err)
		os.Exit(1)
	}

	dir := filepath.Dir(execPath)
	tmpFile, err := os.CreateTemp(dir, ".run-on-business-day-update-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: 一時ファイルの作成に失敗しました: %v\n", err)
		os.Exit(1)
	}
	tmpPath := tmpFile.Name()

	_, err = io.Copy(tmpFile, dlResp.Body)
	tmpFile.Close()
	if err != nil {
		os.Remove(tmpPath)
		fmt.Fprintf(os.Stderr, "Error: ダウンロードデータの書き込みに失敗しました: %v\n", err)
		os.Exit(1)
	}

	// 実行権限を付与 (Windows以外)
	if runtime.GOOS != "windows" {
		if err := os.Chmod(tmpPath, 0755); err != nil {
			os.Remove(tmpPath)
			fmt.Fprintf(os.Stderr, "Error: 実行権限の設定に失敗しました: %v\n", err)
			os.Exit(1)
		}
	}

	// 旧バイナリを .old にリネーム → 新バイナリをユーザーの実行ファイル名で配置
	oldPath := execPath + ".old"
	os.Remove(oldPath) // 前回の .old が残っていれば削除

	if err := os.Rename(execPath, oldPath); err != nil {
		os.Remove(tmpPath)
		fmt.Fprintf(os.Stderr, "Error: 旧バイナリのリネームに失敗しました: %v\n", err)
		fmt.Fprintf(os.Stderr, "Hint: 書き込み権限が必要な場所に配置している場合は sudo で実行してください\n")
		os.Exit(1)
	}

	if err := os.Rename(tmpPath, execPath); err != nil {
		// ロールバック
		os.Rename(oldPath, execPath)
		os.Remove(tmpPath)
		fmt.Fprintf(os.Stderr, "Error: 新バイナリの配置に失敗しました: %v\n", err)
		os.Exit(1)
	}

	// .old を削除 (失敗しても問題ない)
	os.Remove(oldPath)

	fmt.Printf("アップデート完了: %s → %s\n", version, latestVersion)
}
