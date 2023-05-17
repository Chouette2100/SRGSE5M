# SRGSE5M

SHOWROOMのサーバから獲得ポイントのデータを取得してデータベースに格納します。

取得の対象となるイベント・配信者、取得の時間間隔等はWebサーバ/CGIで設定します。

[Github - SHOWROOM イベント 獲得ポイント一覧](https://github.com/Chouette2100/SRCGI)

Webサーバ/CGIと設定や取得データを共有するデータベースのスキーマは以下のものです。

[CreateDB.sql](https://github.com/Chouette2100/SRCGI/blob/main/CreateDB.sql)

DBとそのログイン情報は ServerConfig.yml で設定します。
DB名、ログイン、パスワードをこのこのファイルに直接書くか環境変数から取得するように設定します。

一時的なデータは scoremap.txt に保存されており、データ取得の合間にこのプログラムを再起動すれば
取得したデータには連続性（＝配信中に再起動しても再起動の前後の配信は一枠とみなされる、というような意味）があります。

実行に特権は必要ありません。VSCodeでのデバッグもできます。

実際に使用するときはデーモンとして起動するのが"事故"が起きにくいと思います。

ビルドと実行

Releases から最新の（Webサーバ/CGIに適合した）バージョンをダウンロードし、

~/go/src/SRGSE5M

に展開します。

以下

$ cd ~/go/src/SRGSE5M

$ go mod init

$ go mod tidy

$ go build SRGSE5M.go

としてビルドします。

なお、このソースの作成は go version go1.20.4 linux/amd64 で行っていますが、
Linux,Windows,FreeBSDの最近のバージョンで、コンパイラがgo 1.17以後あたりであれば環境に依存することはないと思います。
クロスコンパイルも問題ありません。

Macでの動作も問題ないと思いますが、Macに関してはまったく検証をしていません。


