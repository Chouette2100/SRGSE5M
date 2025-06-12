// Copyright © 2022-2025 chouette2100@gmail.com
// Released under the MIT license
// https://opensource.org/licenses/mit-license.php
/*
指定した時刻に指定したイベント、配信者の獲得ポイントを取得します。

デーモンとして動かすことを前提としています。
スケジュールはDB(event, eventuser)から読み込み、取得した獲得ポイントはDB（points）に書き込みます

これは前バージョンと実行時パラメーター、入力ファイルの形式をあわせてあります。現在のバージョンではFDetailは機能しません。

EvalPoints2　Folder Interval Mod HH_Detail FTitle FDetail

	Folder		[5|6]			Ex. 6			作業用フォルダーの識別子
	Interval	5..30			Ex. 30			データを取得する間隔、60の公約数
	Mod			0..Interval-1	Ex. 2			IntervalのMod分前にデータを取得する（Interval=30、MOｄ=2であれば、28分、58分）
	HH_Detail	0..23			Ex. 0 または 4		HH_Detailと同一時刻（時間）の最初のデータ取得時に貢献ポイントランキングを取得する。00時は無条件に取得。
	FTitle		[0|1]			Ex.　0			（Intervalが0のとき＝一回のみデータ取得のとき）1であれば配信者名リストを出力する
	FDetail		N/A

	※　Intervalが0でないときは、開始時に配信者名リストを出力します。


*/

package main

import (
	//	"crypto/aes"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	// "strconv"
	"sync"
	"syscall"
	"time"

	//	"strings"
	//	. "log"
	//	"bufio"
	//	"io"

	//	"runtime"

	//	"net/http"

	// "database/sql"

	_ "github.com/go-sql-driver/mysql"

	"github.com/go-gorp/gorp"

	//	"encoding/json"
	//	"github.com/360EntSecGroup-Skylar/excelize"

	//	. "MyModule/ShowroomCGIlib"
	"SRGSE5M/GSE5Mlib"
	"SRGSE5M/SRDBlib"

	// "github.com/dustin/go-humanize"

	"github.com/Chouette2100/exsrapi/v2"
	"github.com/Chouette2100/srapi/v2"
	"github.com/Chouette2100/srdblib/v2"
)

/*
	EvalPoints2A02 2019/04/30
		イベントが終わっている、イベント参加をとりやめた、SHOWROOMをやめた、などの対応
	EvalPoints2A03 2019/06/22
		ランキングイベントとレベルイベントの判別処理を追加した。

	Ver. RU10A0	MySQL版
	Ver. RU10A1	重複データの間引き
	Ver. RU20A0	終了後確定値の取り込み、スキーマ変更（前回データ保存）など（2021.06.10）
	Ver. RU20B0	現配信、前回配信情報の表示の改善（2021.07.14）
	Ver. RU20C0	comment.txtの出力方法の改善（2021.11.27）
	Ver. RU20C1	なにも変えてないはずだが....
	Ver. RU20C2	ShowroomCGIlibからのimportを明示的に書くようにした。
	Ver. RU20D0	MySQL8.0に対応する。DB情報をファイルから読み込む。
	Ver. RU20D1	comment.txtをWindows以外でも作成する（ubuntuでの視聴を考慮）
	Ver. RU20E0	貢献ポイントランキング取得（GetPointsCont）で使用するデータ取得タイムテーブルを作成する。
	Ver. RU20E1	貢献ポイントランキング取得の条件を追加する（eventuser.iscntrbpoints = 'Y'のとき）
	Ver. RU20E2	配信開始時刻の取得を追加する。
	Ver. RU20E3	配信開始時刻、配信継続期間で開始時刻から5分引く処理を取り除く。
	Ver. RU20E4	詳細ランクと次ランクまでのポイントを取得し、保存する。ShowroomCGIlib 0101D2に適合するバージョン。
	Ver. RU20E5	イベント終了直後に獲得ポイントデータのコピー作成と同時に最終的な貢献ポイントランキングを取得する。
	Ver. RU20F0	配信の開始、終了の判断を獲得ポイントの変化にAPIによる配信状態を加味する。
	Ver. RU20F1	終了処理の誤り（Provisionalの状態を通り越してConfirmedの状態に移行してしまう）を修正する。
	Ver. RU20G0	ライブラリShowroomCGIlibをサブディレクトリに移動しGSE5Mlibとする。
	Ver. RU20G1	イベント登録直後にQtimeが"01/01 00:00--00:00"となる場合は表示しない。
	Ver. RU20G2	設定ファイルをyaml形式に変更する。
				timetableにデータ取得時刻を保存するときstime、etimeも書き込むようにする。
	Ver. RU20G3	イベント終了まで5分を切ったら配信終了とみなす。 <== 作成した処理は実態にあっていなかった。
	Ver. RU20G4	配信中にイベントが終了したら貢献ポイントを取得する。
	Ver. RU20G5	上に関連しイベントが終了して10分以内はデータの取得を行う。
	Ver. RU20G6 イベント終了直前で配信が終了したとみなす処理を除く。イベント終了後のtimetable作成でのイベント名を正しくする。
	Ver. RU20G7 makePQ()の使用をやめて展開する。
	Ver. RU20H0 makePQ()の使用をやめて展開する。異常終了対策としてGoPointsAll()をgoroutineとして使用する。
	Ver. RU20H1 GetConfirmedに移行する条件の誤りを修正する。
	Ver. RU20H2 イベント終了時CopyScore()における貢献ポイントランキングの不要な取得を取り除く。
	Ver. RU20H3 イベント終了時CopyScorey()でのProvisional作成時刻を終了時刻＋1秒から＋59秒に変更する。
	Ver. RU20H4 イベント終了時GetConfirmed()でのConfirmed更新時刻を終了時刻＋1秒から＋59秒に変更する。
	Ver. RU20H5 イベント終了時のGetConfirmed()の実行を715分後から810分後に変更する。
	Ver. RU20J0 配信終了の条件を「ポイントの変化なし・配信終了」から、「ポイントの変化なし・配信終了がそれぞれ２回続く」に変更する。
	Ver. RU20K0 起動後一定時間で処理を終了するオプション（TimeLimit）を追加する。
	Ver. RU20K1 crontabから起動する時刻に停止するように変更する。
	Ver. RU20K2 0時に無条件に停止しないように0時を24時として扱う。
	Ver. RU20K3 import GSE5Mlibとし、所在はgo.modで指定する。
	Ver. 020AK00 GetPointsALL()でのDBの更新にトランザクションを用いる。
	Ver. 020AK01 ルームが対象ではないイベントに参加しているときはscoremap[id]の存在チェックを行う。
	Ver. 020AK02 デッドロック対策としてdeleteでwhere句にeventidを追加する。
	Ver. 020AL00 最終結果確定時にMakePointPerSlot()を実行する。これにともないSRDBlibを導入する。
	Ver. 020AM00 できるだけ早く確定情報を取得する。
	Ver. 020AN00 できるだけ早く確定情報を取得する（フェーズ移行の条件の見直し）
	Ver. 020AP03 イベント終了時、CopyScore()の前にGetPointsALL()を実行する（DontGetScoreを導入する）
	Ver. 020AP04 ログ出力を減らすため"Dup=%d .... deleted."のログ出力を削除する。
	Ver. 020AQ04 最終処理（GetConfirmed()）でeventuserに存在しないルームを補う。ログ出力はファイルのみとする。
	Ver. 020AQ05 GetPointsAll() での scoremap[i]の存在をチェックするようにする（チェック後の変数dupを使う）
	Ver. 020AQ06 GetSchedule()でエラーが発生した場合は処理を打ち切る。
	Ver. 020AQ07 InserIntoOrUpdatePoints()のeventuserに対するselect文のwhereの抜けを補う。
	Ver. 020AQ08 最終処理でeventuserに存在しないがuserに存在しなければ補う。
	Ver. 020AQ11 データがなくmax(ts) from pointsがnullとなった場合はその旨出力して後続の処理を行わない。
	Ver. 020AR00 Intervalminが0のときは5とする。Intervaminが0だと剰余を求めるときゼロ割りが起きる。
	Ver. 021AA00 gorpを導入するとともに srdblib を共通パッケージに変更する（第一ステップ）
	Ver. 021AB00 2時間ごとのuserテーブルの更新を停止する。
	Ver. 021AC00 50位以内のルームの獲得ポイントの取得にはGetEventsRankingByApi()を使う。
	Ver. 021AD00 block_id=0のブロックイベントに対応する。これはすべてのブロックイベントを含むイベント全体を示す。
	Ver. 021AD01 block_id=0のとき、51位以下のルームには順位をつけないようにする。
		 021AE00 GetEventsRankingByApi()はイベント開催中と終了後で使い分けられるようにする。
	Ver. 021AE01 CopyScore()でイベント終了時刻＋58秒以前のデータはイベント終了前のデータとみなす。
	Ver. 021AE02	GetIsOnliveByAPI()の内部外部でエラー処理を追加する。
	Ver. 021AE03	GetIsOnliveByAPI()でエラーが起きたときはisonlive=false, startedat = time.Now()とする(暫定対応)
	Ver. 021AE04	GetIsOnliveByAPI()での配信状態、配信開始時刻のチェックは行わない。
	Ver. 021AG01	Ver. 021AF00 bloc_id=0のイベントに対する処理を追加する
	Ver. 021AG02	ScanActive()とGetPointsAll()にPrintExf()を導入する
	Ver. 021AG03	ScanActive()とGetPointsAll()にPrintExf()を導入する(出力形式を変更する)
	Ver. 021AG04	GetPointsAll() イベントを途中で変えた50位より下位のルームを除外する
	Ver. 021AH00	GetPointsAll()をsrapi.GetPointAllByApi()に変更する。ログ出力の形式をイベントIDを中心に統一する。
	Ver. 021AJ00	毎分処理すべきタスクを確実に終了するためロジックを変更する。
	Ver. 021AJ01	exsrapi.FuncNameOfThisFunction()の仕様変更にともなってログ出力を修正する
	Ver. 021AK00	map を sync.Map に変更する
	Ver. 021AL00	GetPointsAll()を分離する。指定順位範囲にあるルームは自動的にeventuserに追加する。
	Ver. 021AL01	指定順位範囲にあるルームは自動的にeventuserに追加する。動作監視のためのログ出力を追加する
	Ver. 021AL03	ログ出力を修正する（処理中のイベントのeventidを表示する。登録直後で取得対象がないときも取得対象のチェックを行う。
	Ver. 021AL04	レベルイベントで獲得ポイントが0のルームを除外する
	Ver. 021AL05	レベルイベントで獲得ポイントが0のルームを除外する(2)
	Ver. 021AL06	ログ出力を datetime eventid userno の形に統一する
	Ver. 021AM00	GetPointsAll()でLengthをlenghthとする（Goルーチンとなっているところでやってはいけない）
	Ver. 021AN00	履歴にないルームのpointが0のときはpointを保存しない
	Ver. 021AN01	eventuserにすでに登録されてルームはポイントデータ取得対象とする、cntrblistをidlistの同様に拡張する。
	Ver. 021AN02	GetPointsAll()のログ出力をeventid id=userno ..... の形に変更する。
	Ver. 021AN05	GetPointsAll()でUpinsEventuser()はInsertIntoPoints()の直後に行う
	Ver. 021AN06	ScanActive()でのcmapはGetSchedule()で取得する、wevenuserの使用はeventuserを使うようにする。
	Ver. 021AP01	IsOnLiveのチェック処理を復活する。
	Ver. 021AP04	イベントの参加を取り消した場合の判断はトランザクションの内部で行う。バグがかなりあった。
	Ver. 021AQ01	開催前のイベントは処理の対象としないものとする
	Ver. 021AR00	GetPointsAll()の二分割を準備する
	Ver. 021AR05	獲得ポイントデータを記録する閾値を設定し、記録するルーム数を制御する。
	Ver. 021AR06	獲得ポイントデータを記録する閾値を設定し、記録するルーム数を制御する(パラメータはファイルに格納する)
	Ver. 021AR07	獲得ポイントデータを記録する閾値を設定し、記録するルーム数を制御する(パラメータはeventテーブルに格納する)
	Ver. 021AR08	GetSchedule()でtoorder が　0　のイベントは処理の対象から除く（toorderが0のデータはSRGCEでテスト用に作ることがある）
	Ver. 021AS00	scoremapのキーを"userno"から"userno eventid"に変更する。これにより、イベント終了時のGetPointsAll()での重複データの削除を行う。
	Ver. 021AS01	ScanActive()でMakeComment()の呼び出しをやめる（直接的にはstormapの扱いが誤っているがMakeComment()は必要性がないから）
	Ver. 021AT00	イベント終了時の処理をデータ作成後最初に行う
	Ver. 021AT01	GetSchedule()でイベント終了の時刻を終了時刻＋1分にする。
	Ver. 021AT02	通常起こりうる事象に対するエラーメッセージを抑制する。
	Ver. 021AT03	SaveScoremap()でscoremapとeventmapの不要なデータを削除する。
	Ver. 021AT04	exsrapi.FuncNameOfThisFunction()の引数の変更に伴う変更を行う
	---------- V2.0,0 --------------------------------
	Ver. 021AU00	thpoint = max(thinit, thdelta * hh) とする
	Ver. 021AU01	GetConfirmed()で結果発表後のイベントページのレイアウトが変更されたため、その対応を行う
					（イベント結果ポイントが取得できなくなっていた）
	Ver. 021AV00	イベント終了時のGetConfirmed()はこのプログラム内では行わない。
	Ver. 021AW00	https://www.showroom-live.com/event/room_listがなくなったため、代替手段を作る。
	Ver. 021AX00	srdblibをv2.3.2に変更する。
	Ver. 021AY00	レベルイベントのルーム取得にGetEventQuestRoomsByApi()を使用する。
	Ver. 021AY01	GetEventQuestRoomsByApi()でエラーが発生したときは獲得ポイント取得の処理を打ち切る。
	Ver. 021AY02	ScanActive()にpanicをrecoverする処理を追加する。
	Ver. 021AZ00	シグナルを捕捉して終了するようにする(グレイスフルシャットダウン)
	Ver. 021AZ01	シグナルを検出したときのメッセージを実態に合わせる。main.goをmain.goとInsertIntoPoints.goに分離する。
	Ver. 021AZ02	srdblib.Dberrをすべてerrとする

	課題
		登録済みの開催予定イベントの配信者がそれを取り消し、別のイベントに参加した場合scoremapを使用した処理に問題が生じる

*/

const version = "021AZ02"

const Maxroom = 10
const ConfirmedAt = 59 //	イベント終了時刻からこの秒数経った時刻に最終結果を格納する。

//	var Thmap map[string][2]int

/*
type Parameters struct {
	EventID   string
	Interval  int
	Mod       int
	HH_Detail int
	FTitle    int
	ExcelType string
	DB        *sql.DB
	eventmap  *map[string]int
}
*/

type LastScore struct {
	Eventid string
	Score   int
	Rank    int
	ts      time.Time
	Dup     int
	Sum0    int
	Tstart0 time.Time
	Tend    time.Time
	//	Sum1    int
	Tstart1   time.Time
	Continued int // 更新（＝（1時間おきに）配信を再スタート）したときの再スタートの回数
	//	Tend1   time.Time
	Qstatus   string
	Qtime     string
	NoOffline int // isonlive でない状態が何回続いたか？
}

type Gschedule struct {
	Eventid   string
	Ieventid  int
	Starttime time.Time
	Endtime   time.Time
	//	Eventno     int
	Intervalmin int
	Modmin      int
	Modsec      int
	Fromorder   int
	Toorder     int
	Cmap        int
	Thinit      int
	Thdelta     int
	Beforestart bool
	Method      string
	Done        bool
}

type Gschedulelist []Gschedule

//	var eventmap map[string]int

// **重要**	構造体のmapを作りたいときはかならずポインターのmapにする。
var eventmap map[string]*GSE5Mlib.Event_Inf

// var snmap map[int]string
var snmap sync.Map

// **重要**	構造体のmapを作りたいときはかならずポインターのmapにする。
//	var scoremap map[int]*LastScore

// 以下のエラーに対する対策のため（goroutineでmapを使うと起きることがある）
// fatal error: concurrent map iteration and map write
var scoremap sync.Map

//	https://tech-up.hatenablog.com/entry/2019/01/05/212630
//	var db *sql.DB
//	var err error

// シャットダウンに関連する状態をまとめた構造体
type AppShutdownManager struct {
	Ctx    context.Context
	Cancel context.CancelFunc // トップレベルでのみ使うことが多いが、構造体に含めることも可能
	Wg     *sync.WaitGroup
	// 他のリソース（DB接続など）を含めることも可能
	// DB *gorp.DbMap // 例
}

// CloseResources はシャットダウン処理とリソース解放を行います。
// defer で呼び出されることを想定しています。
func (sm *AppShutdownManager) CloseResources() {
	log.Println("Closing resources...")

	// コンテキストをキャンセルし、新しいgoroutineの起動を停止
	// シグナル受信などで既に呼ばれている可能性もあるが、冪等なので問題ない
	sm.Cancel()
	log.Println("Context cancelled.")

	// WaitGroupの完了を待つのは、通常main関数で行います。
	// ここでWaitすると、CloseResourcesがブロックされてしまい、
	// main関数がWaitする前にリソース解放が完了しない可能性があります。
	// そのため、Waitはmain関数に任せるのが一般的です。

	// 他のリソース解放処理
	// if sm.DB != nil {
	// 	sm.DB.Db.Close() // gorpのDB接続をクローズ
	// 	fmt.Println("Database connection closed.")
	// }
	// 他のリソース解放処理
	log.Println("Resources closed.")
}

/*
func GetEventInfo() {

		return
	}
*/
func main() {

	cmt0 := "=========="
	fncname := exsrapi.FuncNameOfThisFunction(1) + "()"
	log.Println(cmt0, ">>>>>>>>>>>>>>>>>>", fncname, ">>>>>>>>>>>>>>>>>>>")
	defer exsrapi.PrintExf(cmt0, fncname)()

	// debugon := os.Getenv("DEBUG")

	//	eventmap = make(map[string]int)
	eventmap = make(map[string]*GSE5Mlib.Event_Inf)
	//	parameters.eventmap = &eventmap

	//	snmap = make(map[int]string)
	//	scoremap = map[int]*LastScore{}

	logfilename := version + "_" + GSE5Mlib.Version + "_" + SRDBlib.Version + "_" +
		srapi.Version + "_" + exsrapi.Version + "_" + srdblib.Version + "_" +
		time.Now().Format("20060102") + ".txt"
	logfile, err := os.OpenFile(logfilename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		panic("cannnot open logfile: " + logfilename + err.Error())
	}
	defer logfile.Close()
	log.SetOutput(logfile)
	//	log.SetOutput(io.MultiWriter(logfile, os.Stdout))

	log.Printf(" ****************************\n")
	log.Printf(" GetScoreEvery5Minutes version=%s %s\n", version, GSE5Mlib.Version)

	/*
		GSE5Mlib.Dbconfig, err = GSE5Mlib.LoadConfig("ServerConfig.yml")
		if err != nil {
			panic(err)
		}
		if GSE5Mlib.Dbconfig.TimeLimit == 0 {
			GSE5Mlib.Dbconfig.TimeLimit = 99999
		}
		log.Printf(" Dbconfig=%+v\n", GSE5Mlib.Dbconfig)

		status := GSE5Mlib.OpenDb()
		if status != 0 {
			return
		}
		defer SRDBlib.Db.Close()
	*/

	//	データベースとの接続をオープンする。
	var dbconfig *srdblib.DBConfig
	dbconfig, err = srdblib.OpenDb("DBConfig.yml")
	if err != nil {
		err = fmt.Errorf("srdblib.OpenDb() returned error. %w", err)
		log.Printf("%s\n", err.Error())
		return
	}
	if dbconfig.UseSSH {
		defer srdblib.Dialer.Close()
	}
	defer srdblib.Db.Close()

	log.Printf("********** Dbhost=<%s> Dbname = <%s> Dbuser = <%s> Dbpw = <%s>\n",
		(*dbconfig).DBhost, (*dbconfig).DBname, (*dbconfig).DBuser, (*dbconfig).DBpswd)

	//	gorpの初期設定を行う
	dial := gorp.MySQLDialect{Engine: "InnoDB", Encoding: "utf8mb4"}
	srdblib.Dbmap = &gorp.DbMap{Db: srdblib.Db, Dialect: dial, ExpandSliceArgs: true}

	srdblib.Dbmap.AddTableWithName(srdblib.User{}, "user").SetKeys(false, "Userno")
	srdblib.Dbmap.AddTableWithName(srdblib.Userhistory{}, "userhistory").SetKeys(false, "Userno", "Ts")
	srdblib.Dbmap.AddTableWithName(srdblib.Points{}, "points").SetKeys(false, "Eventid", "User_id", "Ts")

	//	srdblib.Dbmap.AddTableWithName(srdblib.Wuser{}, "wuser").SetKeys(false, "Userno")
	//	srdblib.Dbmap.AddTableWithName(srdblib.Userhistory{}, "wuserhistory").SetKeys(false, "Userno", "Ts")
	//	srdblib.Dbmap.AddTableWithName(srdblib.Event{}, "wevent").SetKeys(false, "Eventid")
	srdblib.Dbmap.AddTableWithName(srdblib.Eventuser{}, "eventuser").SetKeys(false, "Eventid", "Userno")
	srdblib.Dbmap.AddTableWithName(srdblib.Event{}, "event").SetKeys(false, "Eventid")

	//      cookiejarがセットされたHTTPクライアントを作る
	client, jar, err := exsrapi.CreateNewClient("ShowroomCGI")
	if err != nil {
		log.Printf("CreateNewClient: %s\n", err.Error())
		return
	}
	//      すべての処理が終了したらcookiejarを保存する。
	defer jar.Save()

	// -------------------------------------

	// 1. シグナル通知用のチャネルを作成
	// バッファリングされたチャネルにすることで、シグナル受信と処理の間に少し余裕を持たせます。
	sigCh := make(chan os.Signal, 1)
	// SIGINT (Ctrl+C) と SIGTERM を補足するように設定
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// 2. 新しいgoroutineの起動を制御するためのコンテキスト
	// context.WithCancel() でキャンセル可能なコンテキストを作成します。
	ctx, cancel := context.WithCancel(context.Background())
	// main関数が終了する際に確実にcancel()が呼ばれるようにdeferで設定
	// (シグナル受信時にもcancel()を呼びますが、二重呼び出しは問題ありません)
	// defer cancel() // (sm *AppShutdownManager) CloseResources()で呼び出されるので、ここでは不要)

	// 3. 実行中のgoroutineを追跡するための WaitGroup
	var wg sync.WaitGroup

	// ShutdownManager インスタンスを作成
	// DB接続などの初期化もここで行う
	sm := &AppShutdownManager{
		Ctx:    ctx,
		Cancel: cancel,
		Wg:     &wg,
		// DB: initDB(), // 例
	}
	// main関数が終了する際に、リソース解放処理を確実に実行
	// これにより、シグナル受信、エラー終了、正常終了のいずれの場合でも呼ばれる
	defer sm.CloseResources()

	// -------------------------------------

	// デーモンをrestartしたときデータの継続性を確保するためのデータを読み込む
	RestoreScoremap()
	//	Thmap = ReadThpoint()

	go func() {
		var gschedulelist Gschedulelist

		//	hh, _, ss := time.Now().Clock()
		_, _, ss := time.Now().Clock()
		if ss != 0 {
			// time.Sleep(time.Duration(61-ss) * time.Second)
			nxt := time.Now().Add(time.Duration(61-ss) * time.Second)
			// 毎分00秒になるまでウェイとする (例: 次の分の開始まで待つ)
			// Contextを考慮したSleep関数を使うか、Sleep後にContextチェックが必要
			log.Println("Waiting until next minute...")
			select {
			case <-sm.Ctx.Done(): // Sleep前のチェック
				log.Println("Context cancelled before minute wait, exiting.")
				return // または break outerloop_label
			case <-time.After(time.Until(nxt)): // Contextを考慮しないSleepの例
				// Sleepが完了
			}

		}
		st := time.Now()
		//	t := st
		_, mm, _ := st.Clock()
		log.Printf(" start time=%s\n", st.Format("2006-01-02 15:04:05"))

		status := 0

		for {
			select {
			case <-sm.Ctx.Done(): // 外側ループ開始時のチェック
				log.Println("Outer loop: Context cancelled, exiting.")
				return // または break outerloop_label
			default:
				// Contextはまだ有効
			}

			//  現時点で（確定データ取得を含む）獲得ポイントデータ取得が必要なイベントの一覧を作成する
			//	このデータは随時更新可能なので、毎回取得する
			gschedulelist, status = GetSchedule()
			if status != 0 {
				log.Printf("GetSchedule() status=%d\n", status)
				return
			}
			//	fmt.Printf("now=%s t=%s status=%d len=%d\n", time.Now().Format("2006/01/02 15:04:05"), t.Format("2006/01/02 15:04:05"), status, len(gschedulelist))

		outerloop:
			for {
				//  未処理のタスクがなくなるまで繰り返す

				select {
				case <-sm.Ctx.Done(): // 中間ループ開始時のチェック
					fmt.Println("Middle loop: Context cancelled, exiting.")
					return // または break outerloop_label
				default:
					// Contextはまだ有効
				}

				//  現在の分で実行が必要なタスクのなかから実行の秒がいちばん小さなタスクを見つける
				for {
					select {
					case <-sm.Ctx.Done(): // 内側ループ開始時のチェック (必要なら)
						log.Println("Inner loop: Context cancelled, exiting.")
						return // または break outerloop_label
					default:
						// Contextはまだ有効
					}
					nextsec := 99
					idx := -1

					for i := 0; i < len(gschedulelist); i++ {
						if gschedulelist[i].Done {
							//	すでに実行されたタスク
							continue
						}
						if mm%gschedulelist[i].Intervalmin == gschedulelist[i].Modmin {
							tnextsec := gschedulelist[i].Modsec
							if tnextsec < nextsec {
								nextsec = tnextsec
								idx = i
							}
						}
					}

					if idx > -1 {
						//  処理すべきタスクが存在する
						//	hh, mm, ss = time.Now().Clock()
						_, tmm, tss := time.Now().Clock()
						if tmm == mm && tss < nextsec {
							//  まだ処理すべき秒に達していない
							// time.Sleep(time.Duration(nextsec-tss) * time.Second)
							nxt := time.Now().Add(time.Duration(nextsec-tss) * time.Second)
							// 毎分00秒になるまでウェイとする (例: 次のタスクの開始まで待つ)
							// Contextを考慮したSleep関数を使うか、Sleep後にContextチェックが必要
							log.Println("Waiting until next minute...")
							select {
							case <-sm.Ctx.Done(): // Sleep前のチェック
								log.Println("Context cancelled before minute wait, exiting.")
								return // または break outerloop_label
							case <-time.After(time.Until(nxt)): // Contextを考慮しないSleepの例
								// Sleepが完了
							}

						}
						log.Printf("%s method=%s\n", gschedulelist[idx].Eventid, gschedulelist[idx].Method)

						switch gschedulelist[idx].Method {
						case "GetScore":
							//  獲得ポイント取得( GetPointsAll() called )
							// if debugon == "ON" {
							// 	ScanActive(client, gschedulelist[idx])
							// } else {
							// Contextがキャンセルされていないか最終チェックしてからgoroutine起動
							select {
							case <-sm.Ctx.Done():
								log.Println("Context cancelled before spawning func1, skipping.")
								break outerloop // 中間ループを抜ける
							default:
								sm.Wg.Add(1) // goroutine起動直前にAdd
								go func() {
									defer sm.Wg.Done() // goroutine終了時にDone
									log.Println("ScanActive goroutine started.")
									// func1 の実際の処理
									// Contextを渡して処理中にキャンセルをチェックすることも可能
									// processTask1(sm.Ctx, taskData)
									// time.Sleep(time.Second) // 処理をシミュレート
									ScanActive(client, gschedulelist[idx])
									log.Println("func1 goroutine finished.")
								}()
							}

							// }
						case "CopyScore":
							//	最終取得データのコピーを作成する（最終結果格納の準備）
							// if debugon == "ON" {
							// 	CopyScore(gschedulelist[idx])
							// } else {
							// Contextがキャンセルされていないか最終チェックしてからgoroutine起動
							select {
							case <-sm.Ctx.Done():
								log.Println("Context cancelled before spawning func1, skipping.")
								break outerloop // 中間ループを抜ける
							default:
								sm.Wg.Add(1) // goroutine起動直前にAdd
								go func() {
									defer sm.Wg.Done() // goroutine終了時にDone
									log.Println("CopyScore goroutine started.")
									// func1 の実際の処理
									// Contextを渡して処理中にキャンセルをチェックすることも可能
									// processTask1(sm.Ctx, taskData)
									// time.Sleep(time.Second) // 処理をシミュレート
									CopyScore(gschedulelist[idx])
									log.Println("func1 goroutine finished.")
								}()
							}

							// }
						// case "GetConfirmed":
						// 	//  最終結果の取得
						// 	if debugon == "ON" {
						// 		GetConfirmed(gschedulelist[idx])
						// 	} else {
						// 		go GetConfirmed(gschedulelist[idx])
						// 	}
						default:
						}
						gschedulelist[idx].Done = true
					} else {
						break outerloop
					}
				}
			}

			//	毎分00秒になるまで待つ
			_, tmm, tss := time.Now().Clock()
			w := 60 - tss
			if tmm == mm {
				if w > 30 && tmm%5 == 0 {
					time.Sleep(5 * time.Second)
					SaveScoremap()
					//	Thmap = ReadThpoint()
				}
				// time.Sleep(time.Duration(w) * time.Second)
				nxt := time.Now().Add(time.Duration(w) * time.Second)
				// 毎分00秒になるまでウェイとする (例: 次の分の開始まで待つ)
				// Contextを考慮したSleep関数を使うか、Sleep後にContextチェックが必要
				log.Println("Waiting until next minute...")
				select {
				case <-sm.Ctx.Done(): // Sleep前のチェック
					log.Println("Context cancelled before minute wait, exiting.")
					return // または break outerloop_label
				case <-time.After(time.Until(nxt)): // Contextを考慮しないSleepの例
					// Sleepが完了
				}
			}
			//	t = time.Now()
			//	hh, mm, _ = time.Now().Clock()
			_, mm, _ = time.Now().Clock()
			/*
				hh24 := mm
				if hh24 == 0 {
					hh24 = 24
				}
					if hh24%GSE5Mlib.Dbconfig.TimeLimit == 0 && mm == 0 {
						//	一定時間経ったら処理を終了する
						break outerloop
					}
			*/
		}
		//	log.Printf(" end time=%s\n", t.Format("2006-01-02 15:04:05"))
	}()

	// シグナル受信を待つ
	<-sigCh
	log.Println("\nシグナルを受信しました。")

	// シグナル受信をトリガーとして、ShutdownManager経由でキャンセルを呼び出す
	// これにより、Contextを監視しているgoroutineが終了を開始する
	// deferされたCloseResourcesでもCancelは呼ばれるが、シグナル受信時に
	// 即座にキャンセルをトリガーしたい場合はここで明示的に呼ぶ
	// (CloseResources内でCancelを呼ぶ設計の場合は、ここでの明示的な呼び出しは不要)
	// 今回はCloseResources内でCancelを呼ぶ設計なので、ここはコメントアウト
	// sm.Cancel()
	log.Println("シャットダウン処理を開始します。")

	// ShutdownManager経由でWaitを呼び、実行中のすべてのgoroutineが終了するのを待つ
	// Contextがキャンセルされた後、すべてのワーカーgoroutineがDone()を呼ぶのを待つ
	log.Println("実行中のすべてのgoroutineが終了するのを待っています...")
	sm.Wg.Wait()

	// Wait()から戻ったら、すべてのgoroutineが終了したことになります。
	// main関数が終了するため、defer sm.CloseResources() が呼ばれ、
	// リソース解放処理が行われます。
	log.Println("すべてのgoroutineが終了しました。")
	// main関数が終了すると、deferが実行され、プログラムが終了します。

	// 最後に次回再開時に必要な作業用データを保存する。
	SaveScoremap()

}
