package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"run-on-business-day/internal/businessday"
	"run-on-business-day/internal/update"
)

var version = "dev" // 埋め込まれたバージョン (ビルド時に -ldflags "-X main.version=..." で上書きされる)

const (
	exitOK             = 0
	exitError          = 1
	exitNonBusinessDay = 10
)

// options は解析済みの CLI 引数を保持します
type options struct {
	force       bool
	check       bool
	showVersion bool
	workingDir  string
	cmdArgs     []string
}

func main() {
	opts, err := parseOptions(os.Args[1:], os.Stderr)
	if err != nil {
		if err == flag.ErrHelp {
			os.Exit(exitOK)
		}
		os.Exit(2)
	}

	os.Exit(run(opts, os.Stdout, os.Stderr))
}

// parseOptions は CLI 引数を解析して options を返します
func parseOptions(argv []string, stderr io.Writer) (*options, error) {
	fs := flag.NewFlagSet("run-on-business-day", flag.ContinueOnError)
	fs.SetOutput(stderr)

	opts := &options{}
	fs.BoolVar(&opts.force, "force", false, "営業日判定を無視して常に実行する")
	fs.BoolVar(&opts.check, "check", false, "営業日かどうかの判定のみを行い、コマンドは実行しない")
	fs.BoolVar(&opts.showVersion, "version", false, "バージョンを表示する")
	fs.StringVar(&opts.workingDir, "C", "", "コマンド実行前に指定したディレクトリに移動する (short)")
	fs.StringVar(&opts.workingDir, "cwd", "", "コマンド実行前に指定したディレクトリに移動する")

	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: %s [options] [command [args...]]\n\n", "run-on-business-day")
		fmt.Fprintf(stderr, "Subcommands:\n")
		fmt.Fprintf(stderr, "  update\tGitHub Releases から最新バージョンに自己更新する\n\n")
		fmt.Fprintf(stderr, "Options:\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(argv); err != nil {
		return nil, err
	}

	opts.cmdArgs = fs.Args()
	return opts, nil
}

// run はモード分岐を行い、終了コードを返します
func run(opts *options, stdout, stderr io.Writer) int {
	// --version フラグの処理
	if opts.showVersion {
		fmt.Fprintln(stdout, version)
		return exitOK
	}

	// update サブコマンド
	if len(opts.cmdArgs) > 0 && opts.cmdArgs[0] == "update" {
		if err := update.Run(version, stdout, stderr); err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return exitError
		}
		return exitOK
	}

	// 営業日判定
	isBusinessDay := businessday.IsBusinessDay(businessday.NowJST())

	// --check オプションの挙動: 判定結果のみ出力してコマンドは実行しない
	if opts.check {
		if isBusinessDay {
			fmt.Fprintln(stdout, "business day")
			return exitOK
		}
		fmt.Fprintln(stdout, "non-business day")
		return exitNonBusinessDay
	}

	// コマンド引数がない場合は判定モード
	if len(opts.cmdArgs) == 0 {
		if !isBusinessDay {
			return exitNonBusinessDay
		}
		return exitOK
	}

	// コマンド指定時、非営業日はスキップ
	if !isBusinessDay && !opts.force {
		// 非営業日であり、--forceも指定されていない場合はスキップ（正常終了）
		return exitOK
	}

	// 作業ディレクトリへ移動 (オプションが指定されている場合のみ)
	if opts.workingDir != "" {
		if err := os.Chdir(opts.workingDir); err != nil {
			fmt.Fprintf(stderr, "Error: Failed to change working directory to '%s': %v\n", opts.workingDir, err)
			return exitError
		}
	}

	return executeCommand(opts.cmdArgs[0], opts.cmdArgs[1:], stderr)
}

// executeCommand はサブプロセスを実行し、その終了コードを返します
func executeCommand(name string, args []string, stderr io.Writer) int {
	cmd := exec.Command(name, args...)

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
	err := cmd.Run()

	// シグナル転送を停止
	signal.Stop(sigChan)

	// コマンドの終了コードを判定してそのまま返す
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		// 起動失敗やパス見つからずなど
		fmt.Fprintf(stderr, "Error: Execution failed: %v\n", err)
		return exitError
	}

	// 正常終了 (そのままコード0で抜ける)
	return exitOK
}
