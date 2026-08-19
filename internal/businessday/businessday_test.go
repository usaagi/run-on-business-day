package businessday

import (
	"testing"
	"time"
)

// stringToTime はテスト用のユーティリティ関数（JSTとしてパース）
func stringToTime(t *testing.T, dateStr string) time.Time {
	jst, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Fatalf("Asia/Tokyo のロードに失敗: %v", err)
	}
	// 年-月-日 のみ指定し時刻は0時とする
	parsed, err := time.ParseInLocation("2006-01-02", dateStr, jst)
	if err != nil {
		t.Fatalf("日時のパースに失敗: %v", err)
	}
	return parsed
}

func TestIsBusinessDay(t *testing.T) {
	// ここでは、syukujitsu_data.go（2026年以降のデータ）が含まれてコンパイルされることを前提にテスト
	tests := []struct {
		name     string
		dateStr  string
		expected bool // true: 営業日, false: 休業日（土日祝・年末年始）
	}{
		// 平日
		{"平日（月曜日）", "2026-04-13", true}, // 2026/04/13 は月曜で祝日でもない
		{"平日（金曜日）", "2026-04-17", true}, // 2026/04/17 は金曜

		// 土日
		{"土曜日", "2026-04-18", false}, // 2026/04/18 は土曜
		{"日曜日", "2026-04-19", false}, // 2026/04/19 は日曜

		// 年末年始（12/31〜1/3）
		// 1/1は祝日・年末年始の両方の条件に合致する
		{"年末（大晦日）", "2025-12-31", false}, // 年は関係なく 12-31 なので false
		{"年始（1月2日）", "2026-01-02", false},
		{"年始（1月3日）", "2026-01-03", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			targetTime := stringToTime(t, tc.dateStr)
			result := IsBusinessDay(targetTime)
			if result != tc.expected {
				t.Errorf("日付: %s, 期待値: %v, 結果: %v", tc.dateStr, tc.expected, result)
			}
		})
	}
}

// TestIsBusinessDay_Holiday は祝日マップによる判定を、生成データの内容に依存せず検証します。
// syukujitsu_data.go は毎年再生成され過去年のデータが削除されるため、
// 実在の祝日をハードコードするとテストが将来壊れます。
func TestIsBusinessDay_Holiday(t *testing.T) {
	original := syukujitsuMap
	t.Cleanup(func() { syukujitsuMap = original })

	// 2099-04-29 は水曜、2099-05-04 は月曜。いずれも年末年始ルールに該当しない
	syukujitsuMap = map[string]struct{}{
		"2099-04-29": {},
		"2099-05-04": {},
	}

	tests := []struct {
		name     string
		dateStr  string
		expected bool
	}{
		{"マップに存在する日は非営業日（水曜）", "2099-04-29", false},
		{"マップに存在する日は非営業日（月曜）", "2099-05-04", false},
		{"マップに存在しない平日は営業日", "2099-04-30", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			targetTime := stringToTime(t, tc.dateStr)
			if got := IsBusinessDay(targetTime); got != tc.expected {
				t.Errorf("日付: %s, 期待値: %v, 結果: %v", tc.dateStr, tc.expected, got)
			}
		})
	}
}

// TestSyukujitsuMapIsSane は生成された祝日データ自体の健全性を検証します。
// CSV の取得に失敗して空データのままビルドされる事故を検知するための番人です。
func TestSyukujitsuMapIsSane(t *testing.T) {
	if len(syukujitsuMap) < 10 {
		t.Fatalf("祝日データが %d 件しかありません。CSV の取得または生成に失敗している可能性があります", len(syukujitsuMap))
	}

	for dateStr := range syukujitsuMap {
		if _, err := time.Parse("2006-01-02", dateStr); err != nil {
			t.Errorf("祝日データのキーが日付として不正です: %q", dateStr)
		}
	}
}

func TestNowJST(t *testing.T) {
	now := NowJST()
	_, offset := now.Zone()
	if offset != 9*60*60 {
		t.Errorf("JST のオフセットが不正です: %d秒 (期待値: %d秒)", offset, 9*60*60)
	}
}
