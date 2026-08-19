package update

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	repoOwner = "usaagi"
	repoName  = "run-on-business-day"
	apiURL    = "https://api.github.com/repos/" + repoOwner + "/" + repoName + "/releases/latest"

	// checksumsAssetName はリリースに添付されるチェックサム一覧のアセット名
	checksumsAssetName = "SHA256SUMS"
)

var (
	apiClient      = &http.Client{Timeout: 30 * time.Second}
	downloadClient = &http.Client{Timeout: 10 * time.Minute}
)

// errOldBinaryRenameFailed は旧バイナリの退避リネームが失敗したことを示すセンチネルエラー
var errOldBinaryRenameFailed = errors.New("old binary rename failed")

type githubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// Run は GitHub Releases から最新バージョンを取得し、自バイナリを置き換えます
func Run(currentVersion string, stdout, stderr io.Writer) error {
	fmt.Fprintln(stdout, "最新バージョンを確認しています...")

	release, err := fetchLatestRelease(apiClient)
	if err != nil {
		return err
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")
	if latestVersion == currentVersion {
		fmt.Fprintf(stdout, "すでに最新バージョン (%s) です。\n", currentVersion)
		return nil
	}

	fmt.Fprintf(stdout, "新しいバージョンが見つかりました: %s → %s\n", currentVersion, latestVersion)

	target := assetName()
	downloadURL, ok := findAssetURL(release.Assets, target)
	if !ok {
		return fmt.Errorf("お使いの環境 (%s/%s) 向けのバイナリ '%s' がリリースに見つかりません", runtime.GOOS, runtime.GOARCH, target)
	}

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("実行ファイルのパスを取得できません: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("シンボリックリンクの解決に失敗しました: %w", err)
	}
	dir := filepath.Dir(execPath)

	fmt.Fprintf(stdout, "ダウンロード中: %s\n", target)
	tmpPath, err := downloadAsset(downloadClient, downloadURL, dir)
	if err != nil {
		return err
	}

	// 実行権限を付与 (Windows以外)
	if runtime.GOOS != "windows" {
		if err := os.Chmod(tmpPath, 0755); err != nil {
			os.Remove(tmpPath)
			return fmt.Errorf("実行権限の設定に失敗しました: %w", err)
		}
	}

	if err := replaceExecutable(execPath, tmpPath); err != nil {
		if errors.Is(err, errOldBinaryRenameFailed) {
			fmt.Fprintln(stderr, "Hint: 書き込み権限が必要な場所に配置している場合は sudo で実行してください")
		}
		return err
	}

	fmt.Fprintf(stdout, "アップデート完了: %s → %s\n", currentVersion, latestVersion)
	return nil
}

func fetchLatestRelease(client *http.Client) (*githubRelease, error) {
	resp, err := client.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("GitHub APIへの接続に失敗しました: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub APIがステータス %d を返しました", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("レスポンスの解析に失敗しました: %w", err)
	}
	return &release, nil
}

func assetName() string {
	base := repoName + "-" + runtime.GOOS + "-" + runtime.GOARCH
	if runtime.GOOS == "windows" {
		return base + ".exe"
	}
	return base
}

func findAssetURL(assets []githubAsset, name string) (string, bool) {
	for _, a := range assets {
		if a.Name == name {
			return a.BrowserDownloadURL, true
		}
	}
	return "", false
}

func downloadAsset(client *http.Client, url, destDir string) (string, error) {
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("ダウンロードに失敗しました: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ダウンロードがステータス %d を返しました", resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp(destDir, ".run-on-business-day-update-*")
	if err != nil {
		return "", fmt.Errorf("一時ファイルの作成に失敗しました: %w", err)
	}
	tmpPath := tmpFile.Name()

	_, err = io.Copy(tmpFile, resp.Body)
	tmpFile.Close()
	if err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("ダウンロードデータの書き込みに失敗しました: %w", err)
	}

	return tmpPath, nil
}

// replaceExecutable は旧バイナリを .old に退避し、新バイナリを配置します。
// 新バイナリの配置に失敗した場合は .old から旧バイナリへロールバックします。
func replaceExecutable(execPath, tmpPath string) error {
	oldPath := execPath + ".old"
	os.Remove(oldPath) // 前回の .old が残っていれば削除

	if err := os.Rename(execPath, oldPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("旧バイナリのリネームに失敗しました: %w: %w", errOldBinaryRenameFailed, err)
	}

	if err := os.Rename(tmpPath, execPath); err != nil {
		// ロールバック
		os.Rename(oldPath, execPath)
		os.Remove(tmpPath)
		return fmt.Errorf("新バイナリの配置に失敗しました: %w", err)
	}

	// .old を削除 (失敗しても問題ない)
	os.Remove(oldPath)

	return nil
}

// parseChecksums は `<sha256>  <ファイル名>` 形式のテキストを
// ファイル名 -> ハッシュ値(小文字16進) のマップに変換します。
// 2フィールドに分解できない行は無視します。
func parseChecksums(data []byte) map[string]string {
	result := make(map[string]string)

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) != 2 {
			continue
		}
		// sha256sum のバイナリモードマーカー "*" を取り除く
		name := strings.TrimPrefix(fields[1], "*")
		result[name] = strings.ToLower(fields[0])
	}

	return result
}

// fileSHA256 は指定ファイルの SHA-256 ハッシュを小文字16進文字列で返します
func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// verifyChecksum はファイルの SHA-256 ハッシュが期待値と一致するか検証します
func verifyChecksum(path, want string) error {
	got, err := fileSHA256(path)
	if err != nil {
		return fmt.Errorf("ハッシュ値の計算に失敗しました: %w", err)
	}
	if !strings.EqualFold(got, want) {
		return fmt.Errorf("チェックサムが一致しません (期待値: %s, 実際: %s)", want, got)
	}
	return nil
}
