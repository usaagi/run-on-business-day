package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

var version = "dev" // 埋め込まれたバージョン (ビルド時に -ldflags "-X main.version=..." で上書きされる)

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

func main() {
	// 引数解析
	forceFlag := flag.Bool("force", false, "営業日判定を無視して常に実行する")
	checkFlag := flag.Bool("check", false, "営業日かどうかの判定のみを行い、コマンドは実行しない")
	versionFlag := flag.Bool("version", false, "バージョンを表示する")
	var workingDir string
	flag.StringVar(&workingDir, "C", "", "コマンド実行前に指定したディレクトリに移動する (short)")
	flag.StringVar(&workingDir, "cwd", "", "コマンド実行前に指定したディレクトリに移動する")
	flag.Parse()

	// --version フラグの処理
	if *versionFlag {
		fmt.Println(version)
		os.Exit(0)
	}

	cmdArgs := flag.Args()

	// JST (UTC+9) として現在時刻を取得
	jst := time.FixedZone("JST", 9*60*60)
	nowJST := time.Now().In(jst)

	// 営業日判定とスキップ処理
	isBusinessDay := IsBusinessDay(nowJST)

	// --check オプションの挙動: 判定結果のみ出力してコマンドは実行しない
	if *checkFlag {
		if isBusinessDay {
			fmt.Println("business day")
			os.Exit(0)
		} else {
			fmt.Println("non-business day")
			// 仕様通り、非営業日はコード10
			os.Exit(10)
		}
	}

	// コマンド引数がない場合は判定モード
	if len(cmdArgs) == 0 {
		// 判定モード: 営業日なら0、非営業日なら10
		if !isBusinessDay {
			os.Exit(10)
		}
		os.Exit(0)
	}

	// コマンド指定時、非営業日はスキップ
	if !isBusinessDay {
		if !*forceFlag {
			// 非営業日であり、--forceも指定されていない場合はスキップ（正常終了）
			os.Exit(0)
		}
	}

	// 作業ディレクトリへ移動 (オプションが指定されている場合のみ)
	if workingDir != "" {
		err = os.Chdir(workingDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to change working directory to '%s': %v\n", workingDir, err)
			os.Exit(1)
		}
	}

	// サブプロセスの準備
	cmdName := cmdArgs[0]
	var execArgs []string
	if len(cmdArgs) > 1 {
		execArgs = cmdArgs[1:]
	}

	cmd := exec.Command(cmdName, execArgs...)

	// 親プロセスの入出力をそのまま子プロセスに繋ぐ (透過的)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// シグナルハンドリングの準備 (SIGINT, SIGTERM を子プロセスに転送する)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for sig := range sigChan {
			if cmd.Process != nil {
				cmd.Process.Signal(sig)
			}
		}
	}()

	// コマンドを実行して終了を待機 (同期)
	err = cmd.Run()

	// シグナル転送を停止
	signal.Stop(sigChan)

	// コマンドの終了コードを判定してそのまま返す
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			os.Exit(exitError.ExitCode())
		}
		// 起動失敗やパス見つからずなど
		fmt.Fprintf(os.Stderr, "Error: Execution failed: %v\n", err)
		os.Exit(1)
	}

	// 正常終了 (そのままコード0で抜ける)
	os.Exit(0)
}
