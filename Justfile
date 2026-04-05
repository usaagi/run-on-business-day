# run-on-business-day 開発・ビルド用 Justfile

# Windows (PowerShell) 環境でも動作するようにシェルを指定
set shell := ["powershell.exe", "-c"]

# デフォルトのタスク（引数なしで実行した場合は一覧を表示）
default:
    @just --list

# -----------------------------------------------------------------------------
# コード生成・テスト
# -----------------------------------------------------------------------------

# 内閣府から最新の祝日CSVデータを取得する
download-csv:
    @echo "=> 内閣府のサイトから祝日CSVデータをダウンロードしています..."
    Invoke-WebRequest -Uri "https://www8.cao.go.jp/chosei/shukujitsu/syukujitsu.csv" -OutFile "syukujitsu.csv"

# CSVからGoの祝日ソースコード(`syukujitsu_data.go`)を生成する
generate:
    @echo "=> CSVから祝日データを抽出し、syukujitsu_data.goを生成しています..."
    go run tools/csv2go/main.go

# すべての自動テストを実行する
test: generate
    @echo "=> テストを実行しています..."
    go test -v ./...

# -----------------------------------------------------------------------------
# ビルドコマンド (各種プラットフォーム向け)
# -----------------------------------------------------------------------------

# Linux用 (一般的なamd64アーキテクチャ・WSL用) にビルドする
build-linux: generate
    @echo "=> Linux (amd64/WSL) 向けにビルドしています..."
    $env:GOOS="linux"; $env:GOARCH="amd64"; go build -ldflags "-s -w" -o dist/run-on-business-day .

# Linux用 (ARM64: Raspberry Pi, AWS Gravitonなど) にビルドする
build-linux-arm: generate
    @echo "=> Linux (arm64) 向けにビルドしています..."
    $env:GOOS="linux"; $env:GOARCH="arm64"; go build -ldflags "-s -w" -o dist/run-on-business-day-arm64 .

# Windows用 (amd64) にビルドする
build-windows: generate
    @echo "=> Windows (amd64) 向けにビルドしています..."
    $env:GOOS="windows"; $env:GOARCH="amd64"; go build -ldflags "-s -w" -o dist/run-on-business-day.exe .

# すべてのプラットフォーム向けに一括でビルドする
build-all: build-linux build-linux-arm build-windows
    @echo "=> 全プラットフォーム向けのビルドが完了しました。出力先: ./dist"

# 生成物をクリーンアップする
clean:
    @echo "=> ビルド生成物を削除しています..."
    rm -rf dist/
    rm -f run-on-business-day run-on-business-day.exe
