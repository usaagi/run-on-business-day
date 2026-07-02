package businessday

import "time"

var jst = time.FixedZone("JST", 9*60*60)

// IsBusinessDay は指定された日時(t)が日本の営業日かどうかを判定します
// 非営業日: 土日、祝日（syukujitsuMap）、および 年末年始（12-31, 01-01, 01-02, 01-03）
func IsBusinessDay(t time.Time) bool {
	// 1. 土日判定
	weekday := t.Weekday()
	if weekday == time.Saturday || weekday == time.Sunday {
		return false
	}

	// 2. 年末年始判定 (12-31, 01-01, 01-02, 01-03)
	monthDay := t.Format("01-02")
	if monthDay == "12-31" || monthDay == "01-01" || monthDay == "01-02" || monthDay == "01-03" {
		return false
	}

	// 3. 祝日判定 (syukujitsu_data.go で自動生成されたマップを使用)
	dateStr := t.Format("2006-01-02")
	if _, isHoliday := syukujitsuMap[dateStr]; isHoliday {
		return false
	}

	// 上記以外は営業日
	return true
}

// NowJST は現在時刻を JST (UTC+9) として返します
func NowJST() time.Time {
	return time.Now().In(jst)
}
