package main

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

		// 祝日（syukujitsu_data.go 定義）
		// 内閣府データ上、2026年の元日、昭和の日、こどもの日などが存在する前提
		{"祝日（元日）", "2026-01-01", false},
		{"祝日（昭和の日）", "2026-04-29", false},
		{"祝日（みどりの日）", "2026-05-04", false},

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
