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

`run-on-business-day-windows-amd64.exe` のファイルを任意のフォルダ（例: `C:\tools\` など）に `run-on-business-day.exe` にリネームして配置し、Windowsの環境変数 `Path` にそのフォルダを追加してください。

最後にターミナルで `run-on-business-day --check` を実行できれば配置完了です。


## アップデート

`update` サブコマンドで、GitHub Releases から最新バージョンに自動更新できます。

```bash
run-on-business-day update
```

- 現在のバージョンと最新リリースを比較し、同じであれば何もせず終了します
- バイナリはユーザーがリネームしていても、実行ファイルと同じ名前・パスに上書きされます
- 更新後は古いバイナリを削除し、新しいバイナリのみが残ります
- `/usr/local/bin/` など書き込み権限が必要な場所に配置している場合は `sudo run-on-business-day update` で実行してください


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
# 平日であれば /srv/app ディレクトリに移動し、python main.py を実行する
run-on-business-day -C /srv/app -- python main.py

# 現在のディレクトリでそのまま実行し、シェル機能（リダイレクト）を利用する
run-on-business-day -- sh -c 'backup.sh > out.log 2>&1'
```

### オプション
| オプション | 説明 |
| :--- | :--- |
| `-C`, `--cwd` | コマンド実行前に指定したディレクトリに移動（cd）します。 |
| `--force` | 営業日の判定を無視し、休日・祝日であっても強制的にコマンドを実行します。 |
| `--check` | 営業日かどうかの判定結果のみを出力し、実際のコマンドは実行せずに終了します。 |

### 終了コード

#### 実行モード（コマンド指定時: `run-on-business-day -- <command>`）
- `0` : 営業日でコマンド実行成功、または非営業日でスキップしても成功扱いです。
- `1` : 内部エラー（引数不足、ディレクトリ変更失敗、コマンド起動失敗など）
- `その他`: 指定されたコマンドの終了コードをそのまま返します（営業日に実行された場合のみ）

#### 判定モード（コマンド指定なし: `run-on-business-day`）
- `0` : 営業日
- `10`: 非営業日（休業日・祝日・年末年始）

**bash での利用例：**
```bash
# 営業日かどうかで処理を分岐
if run-on-business-day; then
  echo "営業日です"
else
  echo "非営業日です"
fi
```

#### チェックモード（`run-on-business-day --check`）
- `0` : 営業日（出力: `business day`）
- `10`: 非営業日（出力: `non-business day`）

---

### 祝日データの自動更新

本ツールは毎年3月10日に自動で新しいバージョンがリリースされます。例えば、2027年3月10日には `v2028.0` が自動で公開されます。

バージョンに含まれる祝日データの範囲:
- `v2028.0` は `1955年1月1日` 〜 `2028年12月31日` までの祝日データを含みます
- 各バージョンは、1955年からそのバージョンの年の12月31日までの祝日データが埋め込まれています

毎年自動でリリースされるため、定期的に更新することで、常に最新の祝日データを利用できます。


## systemd での利用

本ツールは cron や systemd timer などのスケジューラから呼び出されることを想定しています。systemd の `.service` ファイルや Podman Quadlet の `.container` ファイルで利用する際の設定例を以下に示します。

ExecCondition は終了コード `0` なら ExecStart を実行します。`1-254` では実行されませんがエラーにはなりません。`255` のみがエラーになり扱いになります。

### Podman Quadlet (.container ファイル)

Podman Quadlet で定期実行するコンテナの起動前に営業日チェックを行う場合(.container)：

```ini
[Service]
ExecCondition=/usr/local/bin/run-on-business-day
```

営業日（exit code 0）のときのみコンテナが起動します。非営業日（exit code 10）のときはコンテナ起動がスキップされます。

### systemd service (.service ファイル)

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

営業日のみ指定されたコマンドを実行します。非営業日は何もせずに終了します。


## ビルド方法

本ツールの祝日判定ロジックは、内閣府が公開する「国民の祝日」CSVデータを元に、コンパイル時（ビルド実行時）の年以降のデータのみを抽出してGoソースコードに自動的に埋め込んでいます。

祝日CSVはビルド時に自動ダウンロードされるため、手動で配置する必要はありません。

### 動作前提
- Go の実行環境 (`go` コマンド)
- `just` コマンドランナー (Makefileの代替) ※任意ですが推奨

### ビルド手順（Windows PowerShell等の環境でも可）

プロジェクトルートで以下のコマンドを実行します。これにより、内閣府から祝日CSVを自動ダウンロードし、Goソースコード生成 (`syukujitsu_data.go`) と各種プラットフォーム向けのコンパイルが全自動で行われます。

```bash
just build-all
```

> 備考：`dist/` ディレクトリ内に、Linux向け(`amd64`, `arm64`) と Windows向け(`.exe`) のバイナリが出力されます。環境にあったものをサーバーの `/usr/local/bin/` 等に配置してご利用ください。
