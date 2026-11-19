# MINYO-GO スポット

> POC 用の独立 repo。2026-11-16 に bonsai/neo-minyo-land の `minyo-go/` から切り出して複製したもの。企画の正本は neo-minyo-land 側（docs/PLAN 系と RESEARCH-PLAN）、ここは歌map の実装とスポット台帳の実験場。

「ひたすら移動し、地理・特産・数で記憶する」#1 旅する民謡ゲームの、歌のある場所の台帳。

## ルール（2026-11-16 確定）

- **対象は幅広く**：和歌・民謡・子守唄・童謡・演歌。種別は区別して登録する
- 1曲につき、ゆかりの場所を登録する（1曲に複数の場所があってもよい。例：伝承地と歌碑）
- 歌碑があれば歌碑をGET地点にし、歌い継がれている場所ならその場所をGET地点にする
- 30km圏内でメダルGET
- **歌map（map.html）のメダルは、通常は30km圏内でのみGETできる。テストモードでは距離制限なしで取得できる**
- **歌との関係が確認できない場所は登録しない**（数を無理に維持しない）。「歌碑がある」だけでなく、どの歌とどう結びついているかも残す
- 座標は、安全な公開チェックイン地点が現地確認できるまで null（推測の中心点は使わない）

## ファイル

| ファイル | 中身 |
|---|---|
| `spots-verified.json` | 歌とのつながりを確認できた場所。現在16件（大阪9／奈良3／和歌山4） |
| `spots-candidates.json` | 調査プール 30件。祭り・施設中心で歌とのつながりが未確認のものが多い。確認できたものだけ verified へ昇格させる |
| [`map.html`](map.html) | **歌map**：verified を地図で見るビュー。種別で色分けしたピン＋一覧、現在地からの距離とメダルGET（30km制限／テストモード）、スポット追加フォーム付き。ピンの座標は表示用の目安（Wikipedia・公式ページ出典）で、台帳の座標 null ポリシーは変えない |

## 歌mapからの追加分マージ運用

歌mapの「スポット追加」で足した分はブラウザの localStorage に溜まる。「追加分の書き出し」で JSON を書き出し、その `records` を `spots-verified.json` の `records` へマージして push すると台帳に取り込まれる（取り込み時に根拠と座標を確認する）。

## 種別の内訳（16件）

| 種別 | 件数 | 例 |
|---|---|---|
| 和歌 | 6 | 三室山遊歩道（家持）、玉津島神社、片男波公園、信貴山下駅前（憶良）、千里石ぶみの丘、海ゆかば歌碑 |
| 民謡 | 3 | 淀川三十石船船唄（高槻＝伝承地／南天満公園＝歌碑）、野崎小唄 |
| 子守唄 | 2 | 根来の子守唄、天満の子守歌 |
| 童謡 | 3 | きらきらぼし、はとぽっぽ（新宮駅前／五月山公園） |
| 演歌 | 2 | 王将（通天閣）、月の法善寺横町（法善寺） |

根拠の強さは `evidence_level` で区別している（official_documented＝自治体・公式／news_documented＝報道／secondary_documented＝歌碑データベース等の二次情報）。二次情報のものは現地確認で格上げする。

## Go + SQLite API（MVP実装）

Go 1.23 と pure-Go SQLite driver `modernc.org/sqlite` を使用する。DBはこのリポジトリのアプリ内部で完結し、MINYO LAND共通DBには接続しない。

### 起動

```sh
go mod tidy
go run ./cmd/server -import-spots spots-verified.json
go run ./cmd/server -import-quizzes quizzes.json
go run ./cmd/server -addr :8080
# ブラウザで http://localhost:8080/?api= を開く
```

Goサーバーは `/` と `/map.html` で既存の地図UIも配信する。`?api=` を付けると地図が同じサーバーのクイズ・スタンプAPIに接続する。クイズデータは、各スポットにつき3問を用意したJSONを `-import-quizzes` で取り込む。

```sh
go run ./cmd/server -import-quizzes quizzes.json
```

JSON形式：

```json
{
  "questions": [
    {
      "key": "osaka-verified-001-q1",
      "spot_id": "osaka-verified-001",
      "prompt": "問題文",
      "choices": ["選択肢A", "選択肢B", "選択肢C"],
      "answer_index": 0,
      "sort_order": 1
    }
  ]
}
```

同じ `spot_id` には必ず3問を登録する。正解番号はDB内に保存し、クイズ取得APIのレスポンスには含めない。クイズ原稿データはまだリポジトリに登録されていない。出典確認済みの原稿を用意してから取り込む。

### API

| Method | Path | 説明 |
|---|---|---|
| GET | `/healthz` | DBヘルスチェック |
| GET | `/api/spots` | スポット一覧 |
| GET | `/api/spots/{id}/quiz` | 3問のクイズ（正解は非公開） |
| POST | `/api/attempts` | 3問の回答判定・スタンプ付与 |
| GET | `/api/users/{id}/stamps` | ユーザーの取得済みスタンプ |
| GET | `/api/users/{id}/progress` | 全スポット数・取得数・未取得一覧 |

通常モードではサーバーがスポット座標との距離を計算し、30km以内の場合だけ回答を受け付ける。座標が未確認（NULL）のスポットは通常モードで取得できない。テストモードはサーバー環境変数 `MINYO_GO_ALLOW_TEST_MODE=true` で明示的に許可した場合だけ、リクエストの `test_mode: true` を受け付ける。

### 開発・テスト

```sh
go test ./...
go vet ./...
```

現段階の `user_id` はMVP用の識別子であり、認証機構ではない。本番公開前に認証・ユーザー本人性の検証を追加する。SQLiteはWAL + busy_timeout + synchronous=NORMALで運用し、PRAGMAはmoderncのDSN `_pragma` パラメータで物理接続ごとに適用し、DB接続プールは最大8接続としている。SQLiteの書き込み自体は単一writerなので、WALは読み書きの共存を改善するものであり、書き込みを並列化するものではない。
