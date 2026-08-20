# run-on-business-day

`run-on-business-day` は、日本の営業日（土日・祝日・年末年始以外）にのみ指定されたシェルコマンドを実行するための、前段ラッパー（CLIツール）です。

cron や systemd timer のような既存のスケジューラから呼び出されることを想定しており、休業日にはコマンドを実行せずに終了します。
単一のネイティブバイナリとしてゼロ依存で動作し、判定のためのパース処理を省いたコンパイル済みのハードコードマップを用いることで、オーバーヘッドのない極めて高速な起動と判定を実現しています。

## 特徴
- 完全オフライン動作 — 外部設定ファイルやAPI通信などは一切不要
- 高速判定 — 祝日データは事前にGoソースコードとして自動生成され、バイナリ内に直接ハードコードされています
- 年末年始対応 — 12月31日、および1月1日〜3日は無条件で休業日としてスキップします
- JST固定 — 実行中のサーバーのタイムゾーン(UTC等)に関わらず、必ず `Asia/Tokyo` (日本時間) 基準で日付を判定します

## インストール

### GithubのReleaseからダウンロード
依存関係のない単一のバイナリのため、実行環境に合わせてビルド済みの実行可能ファイルを、システムのパス（PATH）が通っているディレクトリに配置するだけで利用可能です。

#### Linux の場合

以下のコマンドを実行するとダウンロード・配置・実行権限付与まで一括で行えます。

```bash
# Linux amd64 の場合
curl -fsSL https://github.com/usaagi/run-on-business-day/releases/latest/download/run-on-business-day-linux-amd64 \
  -o /tmp/run-on-business-day && \
  sudo install -m 755 /tmp/run-on-business-day /usr/local/bin/run-on-business-day
```

```bash
# Linux arm64 の場合
curl -fsSL https://github.com/usaagi/run-on-business-day/releases/latest/download/run-on-business-day-linux-arm64 \
  -o /tmp/run-on-business-day && \
  sudo install -m 755 /tmp/run-on-business-day /usr/local/bin/run-on-business-day
```

#### Windows の場合

以下のコマンドをPowerShellで実行すると、ダウンロードとリネーム・配置を一括で行えます。

```powershell
# C:\tools\ に配置する場合
Invoke-WebRequest -Uri "https://github.com/usaagi/run-on-business-day/releases/latest/download/run-on-business-day-windows-amd64.exe" -OutFile "C:\tools\run-on-business-day.exe"
```

配置後、Windowsの環境変数 `Path` に `C:\tools\` を追加してください。

最後にターミナルを再起動し `run-on-business-day --check` を実行できれば配置完了です。


## アップデート

`update` サブコマンドで、GitHub Releases から最新バージョンに自動更新できます。

```bash
run-on-business-day update
```

- バイナリはユーザーがリネームしていても、実行ファイルと同じ名前・パスに上書きされます
- `/usr/local/bin/` など書き込み権限が必要な場所に配置している場合は `sudo run-on-business-day update` で実行してください
- ダウンロードしたバイナリは、リリースに添付された `SHA256SUMS` と照合してから配置されます。照合できない場合・一致しない場合は更新を中止します

### アップデートの頻度

毎月1日に自動で内閣府の祝日CSVを取得し、**前回から内容に変化があった場合のみ**新しいバージョンが公開されます。変化がない月はリリースされません。

バージョンは `v<祝日データの最終年>.<連番>` の形式です（例: `v2027.6`）。

### バージョンに含まれる祝日データの範囲

祝日データは**リリース時点の年以降**のみが埋め込まれます。過去年のデータは判定に不要なため取り除かれます。

- 例: 2026年に公開されたリリースには、2026年1月1日以降で内閣府CSVに収録されている祝日が含まれます
- 内閣府CSVは通常、翌年分までしか公開されないため、実質的に「今年と来年」の祝日が入ります
- 年をまたいで古いバイナリを使い続けると、新しい年の祝日が不足する場合があります。`run-on-business-day update` を定期的に実行してください


## 使い方

```bash
run-on-business-day [options] -- <shell command>
```

### 引数
- `--`: ツール自身のオプションと実行コマンドを明確に分割するための標準セパレータ（区切り文字）。
- `<shell command>`: 実際に実行したいコマンドとその引数。

> **注意**: パイプ（`|`）やリダイレクト（`>`）などのシェル機能を利用したい場合は、明示的に `sh -c '...'` などの形式でコマンド文字列として渡してください。

### 実行例
```bash
# 営業日であれば /srv/app ディレクトリに移動し、python main.py を実行する
run-on-business-day -C /srv/app -- python main.py

# 営業日であれば現在のディレクトリでそのまま実行し、シェル機能（リダイレクト）を利用する
run-on-business-day -- sh -c 'backup.sh > out.log 2>&1'
```

### オプション
| オプション | 説明 |
| :--- | :--- |
| `-C`, `--cwd` | コマンド実行前に指定したディレクトリに移動（cd）します。 |
| `--force` | 営業日の判定を無視し、休日・祝日であっても強制的にコマンドを実行します。 |
| `--check` | 営業日かどうかの判定結果のみを出力し、実際のコマンドは実行せずに終了します。 |
| `--version` | 現在のバージョンを表示します。 |
| `--help` | ヘルプメッセージを表示します。 |

### 終了コード

#### 実行モード（コマンド指定時: `run-on-business-day -- <command>`）
- `0` : 営業日でコマンド実行成功、または非営業日でスキップしても成功扱いです。
- `1` : 内部エラー（引数不足、ディレクトリ変更失敗、コマンド起動失敗など）
- `その他`: 指定されたコマンドの終了コードをそのまま返します（営業日に実行された場合のみ）

#### 判定モード（コマンド指定なし: `run-on-business-day`）
- `0` : 営業日
- `10`: 非営業日（休業日・祝日・年末年始）

#### チェックモード（`run-on-business-day --check`）
- `0` : 営業日（出力: `business day`）
- `10`: 非営業日（出力: `non-business day`）

---

## 他の利用方法 (bash, systemd)

### bash での利用例
```bash
# 営業日かどうかで処理を分岐
if run-on-business-day; then
  echo "営業日です"
else
  echo "非営業日です"
fi
```

### systemd
本ツールは cron や systemd timer などのスケジューラから呼び出されることを想定しています。systemd の `.service` ファイルや Podman Quadlet の `.container` ファイルで利用する際の設定例を以下に示します。

ExecCondition は終了コード `0` なら ExecStart を実行します。`1-254` では実行されませんがエラーにはなりません(ログが汚れない) `255` のみがエラー扱いになります。

#### Podman Quadlet (.container ファイル)

Podman Quadlet で定期実行するコンテナの起動前に営業日チェックを行う場合(.container)：

```ini
[Service]
ExecCondition=/usr/local/bin/run-on-business-day
```

営業日のときのみコンテナが起動します。非営業日のときはコンテナ起動がスキップされます。

#### systemd service (.service ファイル)

systemd service でコマンドを実行する場合(.service)：

```ini
[Service]
WorkingDirectory=/home/xxx/myapp
ExecCondition=/usr/local/bin/run-on-business-day
ExecStart=uv run main.py
```

又は

```ini
[Service]
ExecStart=/usr/local/bin/run-on-business-day -C /home/xxx/myapp -- uv run main.py
```


## ビルド方法

本ツールの祝日判定ロジックは、内閣府が公開する「国民の祝日」CSVデータを元に、コード生成実行時の年以降のデータのみを抽出してGoソースコードに自動的に埋め込んでいます。

生成された `internal/businessday/syukujitsu_data.go` はリポジトリにコミットされており、リリースビルドはこのコミット済みファイルをそのまま使います。手元で最新化したい場合は `just generate`（内部で `just download-csv` が必要）を実行してください。

### 動作前提
- Go の実行環境 (`go` コマンド)
- `just` コマンドランナー (Makefileの代替) ※任意ですが推奨

### ビルド手順（Windows PowerShell等の環境でも可）

プロジェクトルートで以下のコマンドを実行します。これにより、内閣府から祝日CSVを自動ダウンロードし、Goソースコード生成 (`syukujitsu_data.go`) と各種プラットフォーム向けのコンパイルが全自動で行われます。

```bash
just build-all
```

> 備考：`dist/` ディレクトリ内に、Linux向け(`amd64`, `arm64`) と Windows向け(`.exe`) のバイナリが出力されます。環境にあったものをサーバーの `/usr/local/bin/` 等に配置してご利用ください。

### リリース手順（メンテナ向け）

祝日データの更新は毎月1日のワークフロー (`.github/workflows/holiday-data-update.yml`) が自動で行います。手動でリリースする場合は次の手順を踏みます。

```bash
# 1. 祝日データを最新化し、差分があればコミットする
just download-csv
just generate
git add internal/businessday/syukujitsu_data.go
git commit -m "chore: update holiday data"

# 2. タグを打って push する（v<データ最終年>.<連番>）
git tag -a v2027.6 -m "Holiday data update for 2027"
git push origin main
git push origin v2027.6
```

タグの push により `.github/workflows/release.yml` が起動し、3プラットフォーム向けのバイナリと `SHA256SUMS` を添付したリリースが作成されます。

> リリースバイナリはタグが指すコミットの `syukujitsu_data.go` をそのままビルドします。タグを打つ前に手順1を済ませてください。
