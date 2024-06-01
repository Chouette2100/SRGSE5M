/*!
Copyright © 2022 chouette.21.00@gmail.com
Released under the MIT license
https://opensource.org/licenses/mit-license.php
*/
/*
指定した時刻に指定したイベント、配信者の獲得ポイントを取得します。

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
	"fmt"
	"log"

	//	. "log"
	"strconv"

	//	"strings"
	"time"

	//	"bufio"
	//	"io"
	"os"

	//	"runtime"

	"database/sql"

	_ "github.com/go-sql-driver/mysql"

	"github.com/go-gorp/gorp"

	//	"encoding/json"
	//	"net/http"
	//	"github.com/360EntSecGroup-Skylar/excelize"

	//	. "MyModule/ShowroomCGIlib"
	"SRGSE5M/GSE5Mlib"
	"SRGSE5M/SRDBlib"

	"github.com/dustin/go-humanize"

	"github.com/Chouette2100/srdblib"
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

	課題
		登録済みの開催予定イベントの配信者がそれを取り消し、別のイベントに参加した場合scoremapを使用した処理に問題が生じる

*/

const version = "021AB00"

const Maxroom = 10
const ConfirmedAt = 59 //	イベント終了時刻からこの秒数経った時刻に最終結果を格納する。

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
	Continued int
	//	Tend1   time.Time
	Qstatus   string
	Qtime     string
	NoOffline int
}

type Gschedule struct {
	Eventid  string
	Ieventid int
	Endtime  time.Time
	//	Eventno     int
	Intervalmin int
	Modmin      int
	Modsec      int
	Beforestart bool
	Method      string
	Done        bool
}

type Gschedulelist []Gschedule

//	var eventmap map[string]int

// **重要**	構造体のmapを作りたいときはかならずポインターのmapにする。
var eventmap map[string]*GSE5Mlib.Event_Inf
var snmap map[int]string

// **重要**	構造体のmapを作りたいときはかならずポインターのmapにする。
var scoremap map[int]*LastScore

//	https://tech-up.hatenablog.com/entry/2019/01/05/212630
//	var db *sql.DB
//	var err error

/*
func InsertSampleTimeIntoTimeacqTable() (timestamp time.Time) {

	//	db := *parameters.DB

	/.
		rows, err := Db.Query("select auto_increment from information_schema.tables where table_name ='timeacq'")
		if err != nil {
			panic(err.Error())
		}
		defer rows.Close()

		//	var idx int
		for rows.Next() {
			err = rows.Scan(&idx)
			if err != nil {
				panic(err.Error())
			}
		}
	./

	//	create table timeacq (idx int auto_increment, t datetime,index(idx));
	/.
		log.Printf("db.Prepare()\n")
		stmt, err := db.Prepare("INSERT INTO timeacq(t) VALUES(?)")
		if err != nil {
			log.Fatal(err)
		}
		//	https://blog.suganoo.net/entry/2019/01/25/190200
		defer stmt.Close()

		log.Printf("row.Exec()\n")
		_, err = stmt.Exec(time.Now().Format("2006-01-02 15:04:05"))
	./
	//	log.Printf("db.Exec()\n")
	timestamp = time.Now()
	stimestamp := timestamp.Format("2006-01-02 15:04:05")
	_, err := Db.Exec("INSERT INTO timeacq (t) VALUES ('" + stimestamp + "')")

	if err != nil {
		log.Fatal(err)
	}

	return
}
*/

func InsertIntoPoints(
	tx *sql.Tx,
	timestamp time.Time,
	userno int,
	point, rank,
	gap int,
	eventid string,
	pstatus string,
	ptime string,
	qstatus string,
	qtime string,
) (
	status int,
) {

	status = 0

	//	log.Printf("InsertIntoPoints()　db.Prepare()\n")
	var stmt *sql.Stmt
	stmt, srdblib.Dberr = tx.Prepare("INSERT INTO points(ts, user_id, eventid, point, `rank`, gap, pstatus, ptime, qstatus, qtime) VALUES(?,?,?,?,?,?,?,?,?,?)")
	if srdblib.Dberr != nil {
		log.Printf("InsertIntoPoints() select err=[%s]\n", srdblib.Dberr.Error())
		status = -1
	}
	defer stmt.Close()

	//	log.Printf("InsertIntoPoints()　row.Exec("InsertIntoOrUpdate",...)\n")
	//	log.Printf("timestamp=%v, userno=%v, eventid=%v, point=%v, rank=%v, gap=%v, pstatus=%v, ptime=%v, qstatus=%v, qtime=%v\n", timestamp, userno, eventid, point, rank, gap, pstatus, ptime, qstatus, qtime)
	_, srdblib.Dberr = stmt.Exec(timestamp, userno, eventid, point, rank, gap, pstatus, ptime, qstatus, qtime)

	if srdblib.Dberr != nil {
		log.Printf("InsertIntoPoints() insert into points err=[%s]\n", srdblib.Dberr.Error())
		status = -1
	}

	//	===============================================
	sqlstmt := "update eventuser set point = ? where eventid = ? and userno = ?"
	_, srdblib.Dberr = tx.Exec(sqlstmt, point, eventid, userno)

	if srdblib.Dberr != nil {
		log.Printf("InsertIntoPoints() update eventuser err=[%s]\n", srdblib.Dberr.Error())
		status = -1
	}

	return
}

func InsertIntoOrUpdatePoints(
	timestamp time.Time,
	roominf GSE5Mlib.RoomInfo,
	rank int,
	gap int,
	eventid string,
	pstatus string,
	ptime string,
	qstatus string,
	qtime string,
) (
	status int,
) {

	status = 0

	nrow := 0
	sqlstmt := "select count(*) from points where ts = ? and eventid = ? and user_id= ?"
	srdblib.Dberr = srdblib.Db.QueryRow(sqlstmt, timestamp, eventid, roominf.Userno).Scan(&nrow)
	if srdblib.Dberr != nil {
		log.Printf("InsertIntoOrUpdatePoints() select err=[%s]\n", srdblib.Dberr.Error())
		status = -1
	}
	if nrow == 0 {
		//	log.Printf("InsertIntoOrUpdatePoints()　db.Prepare()\n")
		var stmt *sql.Stmt
		stmt, srdblib.Dberr = srdblib.Db.Prepare("INSERT INTO points(ts, user_id, eventid, point, `rank`, gap, pstatus, ptime, qstatus, qtime) VALUES(?,?,?,?,?,?,?,?,?,?)")
		if srdblib.Dberr != nil {
			log.Printf("InsertIntoOrUpdatePoints() select err=[%s]\n", srdblib.Dberr.Error())
			status = -1
		}
		defer stmt.Close()

		//	log.Printf("InsertIntoOrUpdatePoints()　row.Exec("InsertIntoOrUpdate",...)\n")
		_, srdblib.Dberr = stmt.Exec(timestamp, roominf.Userno, eventid, roominf.Point, rank, gap, pstatus, ptime, qstatus, qtime)

		if srdblib.Dberr != nil {
			log.Printf("InsertIntoOrUpdatePoints() select err=[%s]\n", srdblib.Dberr.Error())
			status = -1
		}
	} else {
		sqlstmt = "update points set point = ?, `rank`=?, gap=?, pstatus=?, ptime =?, qstatus=?, qtime=? where ts=? and eventid=? and user_id=?"
		_, srdblib.Dberr = srdblib.Db.Exec(sqlstmt, roominf.Point, rank, gap, pstatus, ptime, qstatus, qtime, timestamp, eventid, roominf.Userno)

		if srdblib.Dberr != nil {
			log.Printf("InsertIntoOrUpdatePoints() update points err=[%s]\n", srdblib.Dberr.Error())
			status = -1
		}
	}

	//	===============================================
	nrow = 0
	sqlstmt = "select count(*) from eventuser where eventid = ? and userno = ?"
	srdblib.Dberr = srdblib.Db.QueryRow(sqlstmt, eventid, roominf.Userno).Scan(&nrow)
	if srdblib.Dberr != nil {
		log.Printf("InsertIntoOrUpdatePoints() select err=[%s]\n", srdblib.Dberr.Error())
		status = -1
	}

	if nrow != 0 {
		sqlstmt = "update eventuser set point = ? where eventid = ? and userno = ?"
		_, srdblib.Dberr = srdblib.Db.Exec(sqlstmt, roominf.Point, eventid, roominf.Userno)

		if srdblib.Dberr != nil {
			log.Printf("InsertIntoOrUpdatePoints() update eventuser err=[%s]\n", srdblib.Dberr.Error())
			status = -1
		}

	} else {
		sqlstmt = "insert into eventuser (eventid, userno, istarget, iscntrbpoints, graph, color, point) values (?,?,?,?,?,?,?)"
		_, srdblib.Dberr = srdblib.Db.Exec(sqlstmt, eventid, roominf.Userno, "N", "N", "N", "white", roominf.Point)
		if srdblib.Dberr != nil {
			log.Printf("InsertIntoOrUpdatePoints() insert into eventuser err=[%s]\n", srdblib.Dberr.Error())
			status = -1
		}
	}

	nrow = 0
	sqlstmt = "select count(*) from user where userno = ?"
	srdblib.Dberr = srdblib.Db.QueryRow(sqlstmt, roominf.Userno).Scan(&nrow)
	if srdblib.Dberr != nil {
		log.Printf("InsertIntoOrUpdatePoints() select err=[%s]\n", srdblib.Dberr.Error())
		status = -1
	}

	log.Printf("eventid=%s userno=%d nrow=%d\n", eventid, roominf.Userno, nrow)

	if nrow == 0 {
		log.Printf(" roominf=%v\n", roominf)
		InsertIntoUser(timestamp, eventid, roominf)
	}

	return
}
func InsertIntoUser(tnow time.Time, eventid string, roominf GSE5Mlib.RoomInfo) (status int) {

	status = 0

	userno, _ := strconv.Atoi(roominf.ID)
	log.Printf("  *** InsertIntoUser() *** userno=%d\n", userno)

	log.Printf("insert into user(*new*) userno=%d rank=<%s> nrank=<%s> prank=<%s> level=%d, followers=%d\n",
		userno, roominf.Rank, roominf.Nrank, roominf.Prank, roominf.Level, roominf.Followers)

	sql := "INSERT INTO user(userno, userid, user_name, longname, shortname, genre, `rank`, nrank, prank, level, followers, ts, currentevent)"
	sql += " VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)"

	//	log.Printf("sql=%s\n", sql)
	stmt, err := srdblib.Db.Prepare(sql)
	if err != nil {
		log.Printf("InsertIntoUser() error() (INSERT/Prepare) err=%s\n", err.Error())
		status = -1
		return
	}
	defer stmt.Close()

	lenid := len(roominf.ID)
	_, err = stmt.Exec(
		userno,
		roominf.Account,
		roominf.Name,
		//	roominf.ID,
		roominf.Name,
		roominf.ID[lenid-2:lenid],
		roominf.Genre,
		roominf.Rank,
		roominf.Nrank,
		roominf.Prank,
		roominf.Level,
		roominf.Followers,
		tnow,
		eventid,
	)

	if err != nil {
		log.Printf("error(InsertIntoOrUpdateUser() INSERT/Exec) err=%s\n", err.Error())
		//	status = -2
		_, err = stmt.Exec(
			userno,
			roominf.Account,
			roominf.Account,
			roominf.ID,
			roominf.ID[lenid-2:lenid],
			roominf.Genre,
			roominf.Rank,
			roominf.Nrank,
			roominf.Prank,
			roominf.Level,
			roominf.Followers,
			tnow,
			eventid,
		)
		if err != nil {
			log.Printf("error(InsertIntoOrUpdateUser() INSERT/Exec) err=%s\n", err.Error())
			status = -2
		}
	}

	return

}

/*
func GetRoomInfoByAPI(room_id string) (
	genre string,
	rank string,
	nrank string,
	prank string,
	level int,
	followers int,
	fans int,
	fans_lst int,
	roomname string,
	roomurlkey string,
	startedat time.Time,
	status int,
) {

	status = 0

	//	https://qiita.com/takeru7584/items/f4ba4c31551204279ed2
	url := "https://www.showroom-live.com/api/room/profile?room_id=" + room_id

	resp, err := http.Get(url)
	if err != nil {
		//	一時的にデータが取得できない。
		//	resp.Body.Close()
		//		panic(err)
		status = -1
		return
	}
	defer resp.Body.Close()

	//	JSONをデコードする。
	//	次の記事を参考にさせていただいております。
	//		Go言語でJSONに泣かないためのコーディングパターン
	//		https://qiita.com/msh5/items/dc524e38073ed8e3831b

	var result interface{}
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&result); err != nil {
		//	panic(err)
		status = -2
		return
	}

	value, _ := result.(map[string]interface{})["follower_num"].(float64)
	followers = int(value)

	tnow := time.Now()
	fans = GetAciveFanByAPI(room_id, tnow.Format("200601"))
	yy := tnow.Year()
	mm := tnow.Month() - 1
	if mm < 0 {
		yy -= 1
		mm = 12
	}
	fans_lst = GetAciveFanByAPI(room_id, fmt.Sprintf("%04d%02d", yy, mm))

	genre, _ = result.(map[string]interface{})["genre_name"].(string)

	rank, _ = result.(map[string]interface{})["league_label"].(string)
	ranks, _ := result.(map[string]interface{})["show_rank_subdivided"].(string)
	rank = rank + " | " + ranks

	value, _ = result.(map[string]interface{})["next_score"].(float64)
	nrank = humanize.Comma(int64(value))
	value, _ = result.(map[string]interface{})["prev_score"].(float64)
	prank = humanize.Comma(int64(value))

	value, _ = result.(map[string]interface{})["room_level"].(float64)
	level = int(value)

	roomname, _ = result.(map[string]interface{})["room_name"].(string)

	roomurlkey, _ = result.(map[string]interface{})["room_url_key"].(string)

	//	配信開始時刻の取得
	value, _ = result.(map[string]interface{})["current_live_started_at"].(float64)
	startedat = time.Unix(int64(value), 0).Truncate(time.Second)
	//	log.Printf("current_live_stared_at %f %v\n", value, startedat)

	return

}

func GetAciveFanByAPI(room_id string, yyyymm string) (nofan int) {

	nofan = -1

	url := "https://www.showroom-live.com/api/active_fan/room?room_id=" + room_id + "&ym=" + yyyymm

	resp, err := http.Get(url)
	if err != nil {
		//	一時的にデータが取得できない。
		//	resp.Body.Close()
		//		panic(err)
		nofan = -1
		return
	}
	defer resp.Body.Close()

	//	JSONをデコードする。
	//	次の記事を参考にさせていただいております。
	//		Go言語でJSONに泣かないためのコーディングパターン
	//		https://qiita.com/msh5/items/dc524e38073ed8e3831b

	var result interface{}
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&result); err != nil {
		//	panic(err)
		nofan = -2
		return
	}

	value, _ := result.(map[string]interface{})["total_user_count"].(float64)
	nofan = int(value)

	return
}
*/

func DeleteFromPoints(tx *sql.Tx, eventid string, ts time.Time, user_id int) {

	sql := "delete from points where eventid = ? and ts= ? and user_id = ?"

	//	log.Printf("Db.Exec(\"delete ...\")\n")
	_, srdblib.Dberr = tx.Exec(sql, eventid, ts, user_id)

	if srdblib.Dberr != nil {
		log.Printf("DeleteFromPoints() select err=[%s]\n", srdblib.Dberr.Error())
		//	status = -1
	}

}

func InsertIntoTimeTable(
	eventid string,
	userno int,
	st1 time.Time,
	earnedp int,
	stime time.Time,
	etime time.Time,
) (
	status int,
) {

	status = 0

	log.Printf("InsertIntoTimeTable() called. eventid=%s, userno =%d st1=%v\n", eventid, userno, st1)

	var stmt *sql.Stmt
	sql := "INSERT INTO timetable(eventid, userid, sampletm1, stime, etime, target, earnedpoint, status)"
	sql += " VALUES(?,?,?,?,?,?,?,?)"
	stmt, srdblib.Dberr = srdblib.Db.Prepare(sql)
	if srdblib.Dberr != nil {
		log.Printf("InsertIntoPoints() prepare() err=[%s]\n", srdblib.Dberr.Error())
		status = -1
	}
	defer stmt.Close()

	_, srdblib.Dberr = stmt.Exec(eventid, userno, st1, stime, etime, -1, earnedp, 0)

	if srdblib.Dberr != nil {
		log.Printf("InsertIntoEventrank() exec() err=[%s]\n", srdblib.Dberr.Error())
		status = -1
	}

	return
}

/*
配信者のリストからそれぞれの獲得ポイントなどを取得する。
*/
func GetPointsAll(IdList []string, gschedule Gschedule, cntrblist []string) (status int) {

	status = 0

	//	wtdp := 2
	//	delay := time.Duration((wtdp+1)*gschedule.Intervalmin) * time.Minute

	//	timestamp := InsertSampleTimeIntoTimeacqTable()
	timestamp := time.Now().Truncate(time.Second)
	if timestamp.After(gschedule.Endtime.Add(time.Duration(gschedule.Intervalmin+1) * time.Minute)) {
		log.Printf("set rstatus = DontGetScore eventid=%s\n", gschedule.Eventid)
		sqlstmte := "update event set rstatus = ? where eventid = ?"
		_, srdblib.Dberr = srdblib.Db.Exec(sqlstmte, "DontGetScore", gschedule.Eventid)

		if srdblib.Dberr != nil {
			log.Printf("GetPointsAll() update event err=[%s]\n", srdblib.Dberr.Error())
		}
	}

	Length := len(IdList)

	if Length == 0 {
		return
	}

	var tx *sql.Tx
	tx, srdblib.Dberr = srdblib.Db.Begin()
	if srdblib.Dberr != nil {
		log.Printf("GetPointsAll() begin err=[%s]\n", srdblib.Dberr.Error())
		return -1
	}
	defer tx.Rollback()

	//	pstatus := "n/a"
	//	ptime := ""
	for i := 0; i < Length; i++ {

		//	var makePQ func()
		id, _ := strconv.Atoi(IdList[i])

		//	開催されていないイベントに対する設定を兼ねる変数定義
		point := 0
		rank := 1
		gap := 0
		eventid := gschedule.Eventid

		var isonlive bool
		var startedat time.Time

		if !gschedule.Beforestart {
			//	開催されているイベント
			point, rank, gap, eventid = GSE5Mlib.GetPointsByAPI(IdList[i])
			if eventid != gschedule.Eventid {
				//	イベントがデータ取得対象のイベントではない
				//	Ver. RU20G4	配信中にイベントが終了したら貢献ポイントを取得する。
				log.Printf(" eventid=%s isn't gschedule.Eventid(%s) .\n", eventid, gschedule.Eventid)
				dup := -9
				if _, ok := scoremap[id]; ok {
					dup = scoremap[id].Dup
				}
				log.Printf(" eventid=%s timestamp=%v gschedule.Endtime=%v scoremap[id].Dup=%d\n", eventid, timestamp, gschedule.Endtime, dup)
				if timestamp.After(gschedule.Endtime) {
					//	イベントが終了している。
					//	if scoremap[id].Dup == 0 {	// 該当scoremap[id]が存在しない場合異常終了する。
					if dup == 0 {
						//	配信中のイベント終了であるので貢献ランキングを取得する。
						//	RU20G6 InsertIntoTimeTable(eventid, id, timestamp.Add(15 * time.Minute), (*scoremap[id]).Sum0, (*scoremap[id]).Tstart0, gschedule.Endtime)
						if cntrblist[i] == "Y" {
							//	イベント配信者設定で貢献ポイントランキングを取得すると設定されている場合
							InsertIntoTimeTable(gschedule.Eventid, id, timestamp.Add(15*time.Minute), (*scoremap[id]).Sum0, (*scoremap[id]).Tstart0, gschedule.Endtime)
							scoremap[id].Dup = -1
						}
						//	makePQ()
						log.Printf(" eventid=%s id=%d !isonlive\n", eventid, id)
						if _, ok := scoremap[id]; !ok {
							log.Printf(" eventid=%s scoremap[%d] not found.\n", eventid, id)
							return
						}

						//	RU20G1
						//	(*scoremap[id]).Qtime = (*scoremap[id]).Tstart0.Add(-time.Duration(gschedule.Modmin*60+gschedule.Modsec)*time.Second).Format("01/02 15:04") + "--" + timestamp.Add(-time.Duration(gschedule.Modmin*60+gschedule.Modsec)*time.Second-delay).Format("15:04")
						ststart0 := (*scoremap[id]).Tstart0.Format("01/02 15:04")
						stend := (*scoremap[id]).Tend.Format("15:04")
						log.Printf(" eventid=%s ststart0 = [%s] stend = [%s]\n", eventid, ststart0, stend)
						if ststart0 == "01/01 00:00" {
							ststart0 = ""
						}
						if stend == "00:00" {
							stend = ""
						}
						log.Printf(" eventid=%s ststart0 = [%s] stend = [%s]\n", eventid, ststart0, stend)
						if ststart0 != "" || stend != "" {
							(*scoremap[id]).Qtime = ststart0 + "--" + stend
						} else {
							(*scoremap[id]).Qtime = ""
						}
						//	RU20G1	-----------------------------

						if (*scoremap[id]).Continued > 0 {
							(*scoremap[id]).Qtime += fmt.Sprintf("(C%d)", (*scoremap[id]).Continued)
						} else if (*scoremap[id]).Continued == -1 {
							(*scoremap[id]).Qtime += "(E)"
						} else if (*scoremap[id]).Continued < -1 {
							(*scoremap[id]).Qtime += "(U)"
						}

					}

				}
				//	Ver. RU20G4	-----------------------------------------------------------------
				continue
			}
			isonlive, startedat, _ = GSE5Mlib.GetIsOnliveByAPI(IdList[i])
			if _, ok := scoremap[id]; ok {
				if isonlive {
					scoremap[id].NoOffline = 0
				} else {
					scoremap[id].NoOffline++
				}
			} else {
				log.Printf(" eventid=%s scoremap[%d] not found.\n", eventid, id)
			}
		}

		//	id, _ := strconv.Atoi(IdList[i])
		pstatus := "n/a"
		ptime := ""
		if _, ok := scoremap[id]; ok {
			if (*scoremap[id]).Eventid != gschedule.Eventid {
				//	scoremap[]にあるイベントが取得対象のイベントと違う ＝  取得対象イベントでの初めてのデータ取得
				log.Printf(" eventid=%s %s *Chg*%8d%7d %s\n", eventid, timestamp.Format("15:04:05"), point, id, eventid)
				var score LastScore
				score.Eventid = gschedule.Eventid
				score.Score = point
				score.Rank = rank
				score.ts = timestamp
				score.Dup = 0
				score.Qtime = ""
				score.Qstatus = ""

				if isonlive {
					score.Tstart0 = startedat
					score.Tstart1 = startedat
					score.Tend = startedat.Add(10000 * time.Hour)
					score.Continued = -999
					ptime = startedat.Format("01/02 15:04:05")
					pstatus = "n/a"
				} else {
					score.Continued = 0
					ptime = ""
					pstatus = "="
				}

				scoremap[id] = &score

			} else if (*scoremap[id]).Score == point && scoremap[id].Rank == rank {
				//	獲得ポイントも順位も変化がないとき
				//	（順位の変化を獲得ポイントの変化と同一視するのは特定順位を目標とする場合があることを考慮しているため）

				if !isonlive {
					if scoremap[id].Dup == 1 && scoremap[id].NoOffline > 1 {
						//	同一の獲得ポイントが３回、オフラインが２回（以上）連続したとき
						if (*scoremap[id]).Sum0 > 0 {
							(*scoremap[id]).Qstatus = "+" + humanize.Comma(int64((*scoremap[id]).Sum0))
						} else if (*scoremap[id]).Sum0 < 0 {
							(*scoremap[id]).Qstatus = "-" + humanize.Comma(int64(-(*scoremap[id]).Sum0))
						}
						if (*scoremap[id]).Tend.After(timestamp) {
							(*scoremap[id]).Tend = timestamp
						}

						//	makePQ = func() {

						log.Printf(" eventid=%s id=%d !isonlive\n", eventid, id)
						if _, ok := scoremap[id]; !ok {
							log.Printf(" eventid=%s scoremap[%d] not found.\n", eventid, id)
							//	return
							continue
						}

						//	RU20G1
						//	(*scoremap[id]).Qtime = (*scoremap[id]).Tstart0.Add(-time.Duration(gschedule.Modmin*60+gschedule.Modsec)*time.Second).Format("01/02 15:04") + "--" + timestamp.Add(-time.Duration(gschedule.Modmin*60+gschedule.Modsec)*time.Second-delay).Format("15:04")
						ststart0 := (*scoremap[id]).Tstart0.Format("01/02 15:04")
						stend := (*scoremap[id]).Tend.Format("15:04")
						log.Printf(" eventid=%s ststart0 = [%s] stend = [%s]\n", eventid, ststart0, stend)
						if ststart0 == "01/01 00:00" {
							ststart0 = ""
						}
						if stend == "00:00" {
							stend = ""
						}
						log.Printf(" eventid=%s ststart0 = [%s] stend = [%s]\n", eventid, ststart0, stend)
						if ststart0 != "" || stend != "" {
							(*scoremap[id]).Qtime = ststart0 + "--" + stend
						} else {
							(*scoremap[id]).Qtime = ""
						}
						//	RU20G1	-----------------------------

						if (*scoremap[id]).Continued > 0 {
							(*scoremap[id]).Qtime += fmt.Sprintf("(C%d)", (*scoremap[id]).Continued)
						} else if (*scoremap[id]).Continued == -1 {
							(*scoremap[id]).Qtime += "(E)"
						} else if (*scoremap[id]).Continued < -1 {
							(*scoremap[id]).Qtime += "(U)"
						}
						//	}

						//	makePQ()

						(*scoremap[id]).Continued = 0

						if cntrblist[i] == "Y" {
							//	イベント配信者設定で貢献ポイントランキングを取得すると設定されている場合
							if (*scoremap[id]).Sum0 != 0 {
								//	配信がされていないときに順位が変わったケースは除く
								//	Ver. RU20G4	配信中にイベントが終了したら貢献ポイントを取得する（ことによって不要になった部分）
								//	InsertIntoTimeTable(eventid, id, timestamp, (*scoremap[id]).Sum0, (*scoremap[id]).Tstart0, (*scoremap[id]).Tend)
								InsertIntoTimeTable(eventid, id, timestamp.Add(5*time.Minute), (*scoremap[id]).Sum0, (*scoremap[id]).Tstart0, (*scoremap[id]).Tend)
								/*
									if time.Until(gschedule.Endtime) > 5 * time.Minute {
									} else {
										//	配信終了直前では獲得ポイントの更新は行われなくなるが貢献ポイントは更新されるはず (RU20G3)
										log.Printf(" time.Until(gschedule.Endtime) < 5 * time.Minute\n")
										InsertIntoTimeTable(eventid, id, timestamp.Add(15 * time.Minute), (*scoremap[id]).Sum0, (*scoremap[id]).Tstart0, (*scoremap[id]).Tend)
									}
								*/
								//	Ver. RU20G4	-----------------------------------------------------

							}
						}

						ptime = ""
						pstatus = "="
						log.Printf(" eventid=%s p = [%s], [%s] q= [%s], [%s]\n", eventid, pstatus, ptime, (*scoremap[id]).Qstatus, (*scoremap[id]).Qtime)

						//	(*scoremap[id]).Dup += 1
						(*scoremap[id]).Sum0 = 0
					}
					if (*scoremap[id]).Dup != 0 {
						//	獲得ポイントが3回（以上）同じなのでまんなかのデータを削除する。
						//	これによってすべての配信者の獲得ポイントは最終取得時刻のものが存在する）
						DeleteFromPoints(tx, eventid, scoremap[id].ts, id)
						ptime = ""
						pstatus = "="
						//	log.Printf(" eventid=%s %s Dup=%d %8d%7d %s deleted.\n", eventid, timestamp.Format("15:04:05"), (*scoremap[id]).Dup, point, id, eventid)
					}
					(*scoremap[id]).Dup += 1
					//	(*scoremap[id]).Sum0 = 0
				} else {
					//	配信中
					ptime = (*scoremap[id]).Tstart0.Format("01/02 15:04:05")
					if startedat != (*scoremap[id]).Tstart1 {
						//	配信が始まった
						(*scoremap[id]).Tstart1 = startedat
						if scoremap[id].Sum0 != 0 {
							//	配信が更新された	Ver. RU20J0
							(*scoremap[id]).Continued++
							ptime = (*scoremap[id]).Tstart0.Format("01/02 15:04:05") + fmt.Sprintf("C%d", (*scoremap[id]).Continued)
						}
					}
					if (*scoremap[id]).Sum0 > 0 {
						pstatus = "+" + humanize.Comma(int64((*scoremap[id]).Sum0))
					} else if (*scoremap[id]).Sum0 < 0 {
						pstatus = "-" + humanize.Comma(int64(-(*scoremap[id]).Sum0))
					}
					(*scoremap[id]).Dup = 0

				}
				(*scoremap[id]).ts = timestamp
			} else {
				//	獲得ポイントか順位が変化した。
				pdelta := point - (*scoremap[id]).Score

				if pdelta != 0 {
					//	獲得ポイントが変化した。
					if isonlive {
						//	配信中のとき
						if (*scoremap[id]).Sum0 == 0 {
							//	最初の変化＝配信の開始であるとき
							(*scoremap[id]).Tstart0 = startedat
							(*scoremap[id]).Tend = startedat.Add(10000 * time.Hour)
							(*scoremap[id]).Tstart1 = startedat
						} else {
							//	獲得ポイントの変化が続いているとき
							if (*scoremap[id]).Tstart1 != startedat {
								//	更新が行われた。
								(*scoremap[id]).Tstart1 = startedat
								(*scoremap[id]).Continued++
							}
						}
					} else {
						if (*scoremap[id]).Sum0 == 0 {
							//	減算が行われたあるいは短時間の配信が行われたと思われるとき（誤操作で配信をはじめ、すぐに配信をやめたようなケース）
							(*scoremap[id]).Tstart0 = timestamp.Add(-time.Duration(gschedule.Modmin*60+gschedule.Modsec) * time.Second)
							(*scoremap[id]).Tend = timestamp
							(*scoremap[id]).Continued = -1
						} else {
							//	配信が終了した。
							(*scoremap[id]).Tend = timestamp
						}
					}

					(*scoremap[id]).Sum0 += pdelta
					ptime = (*scoremap[id]).Tstart0.Format("01/02 15:04:05")
					if (*scoremap[id]).Continued > 0 {
						ptime += fmt.Sprintf("(C%d)", (*scoremap[id]).Continued)
					} else if (*scoremap[id]).Continued == -1 {
						ptime += "(E)"
					} else if (*scoremap[id]).Continued < -1 {
						ptime += "(U)"
					}

					if (*scoremap[id]).Sum0 > 0 {
						pstatus = "+" + humanize.Comma(int64((*scoremap[id]).Sum0))
					} else if (*scoremap[id]).Sum0 < 0 {
						pstatus = "-" + humanize.Comma(int64(-(*scoremap[id]).Sum0))
					}
				} else {
					//	順位だけ変動した
					pstatus = "="
					ptime = ""

					//	RU20G6 順位だけ変化したときはQtimeが変化しないようにする。
					(*scoremap[id]).Dup += 1

				}

				//	log.Printf("different data idx=%d, eventid=%s, user_id=%d point=%d\n", idx, eventid, id, point)
				log.Printf(" eventid=%s %s Diff.%8d%7d %s %s %s\n", eventid, timestamp.Format("15:04:05"), point, id, eventid, ptime, pstatus)
				(*scoremap[id]).Score = point
				(*scoremap[id]).Rank = rank
				(*scoremap[id]).ts = timestamp

				(*scoremap[id]).Dup = 0
			}
		} else {
			//	ユーザの獲得ポイント履歴がない。新しく作ります。
			//	log.Printf("new data idx=%d, user_id=%d point=%d\n", idx, id, point)
			log.Printf(" eventid=%s %s *New*%8d%7d %s\n", eventid, timestamp.Format("15:04:05"), point, id, eventid)
			var score LastScore
			score.Eventid = gschedule.Eventid
			score.Score = point
			score.Rank = rank
			score.ts = timestamp
			score.Dup = 0
			score.Qtime = ""
			score.Qstatus = ""

			if isonlive {
				score.Tstart0 = startedat
				score.Tstart1 = startedat
				score.Continued = -999
				ptime = startedat.Format("01/02 15:04:05")
				pstatus = "n/a"
			} else {
				score.Continued = 0
				ptime = ""
				pstatus = "="
			}

			scoremap[id] = &score

			//	ptime = ""
			//	log.Printf("scoremap[%d]=%v\n", id, scoremap[id])
		}

		InsertIntoPoints(tx, timestamp, id, point, rank, gap, eventid, pstatus, ptime, (*scoremap[id]).Qstatus, (*scoremap[id]).Qtime)

	}

	tx.Commit()

	SaveScoremap()

	//	if runtime.GOOS == "windows" {
	MakeComment()
	//	}

	return
}

func SaveScoremap() (status int) {

	status = 0

	file, err := os.OpenFile("scoremap.txt", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	if err != nil {
		fmt.Println(" Can't open file. [", "scoremap.txt", "]")
		status = 1
		return
	}

	//	fmt.Fprintf(file, "%d\n", -1)
	fmt.Fprintf(file, "%d\n", -2)
	for id, lastscore := range scoremap {

		fmt.Fprintf(file, "%d\n", id)
		fmt.Fprintf(file, "%s\n", (*lastscore).Eventid)
		fmt.Fprintf(file, "%d %d %d %d\n", (*lastscore).Score, (*lastscore).Rank, (*lastscore).Dup, (*lastscore).Sum0)
		fmt.Fprintf(file, "%q\n", (*lastscore).ts.Format("2006/01/02 15:04:05 MST"))
		fmt.Fprintf(file, "%q\n", (*lastscore).Tstart0.Format("2006/01/02 15:04:05 MST"))
		fmt.Fprintf(file, "%q\n", (*lastscore).Tstart1.Format("2006/01/02 15:04:05 MST"))
		fmt.Fprintf(file, "%d\n", (*lastscore).Continued)
		fmt.Fprintf(file, "%q\n", (*lastscore).Qstatus)
		fmt.Fprintf(file, "%q\n", (*lastscore).Qtime)

		fmt.Fprintf(file, "%d\n", (*lastscore).NoOffline)

		//	file.Write([]byte(lastscore))
		//	err = binary.Write(file, binary.LittleEndian, lastscore)
		//	fmt.Printf("%v\n%#v\n", err, lastscore)

	}

	file.Close()

	return
}

func RestoreScoremap() (status int) {

	status = 0

	file, err := os.OpenFile("scoremap.txt", os.O_RDONLY, 0644)
	if err != nil {
		fmt.Println(" Can't open file. [", "scoremap.txt", "]")
		status = 1
		return
	}

	fver := 0

	id := 0
	ts := ""
	tstart0 := ""
	tstart1 := ""
	eventid := ""
	for {
		var lastscore LastScore

		_, err = fmt.Fscanf(file, "%d\n", &id)
		if err != nil {
			break
		}

		if id < 0 {
			fver = -id
			fmt.Fscanf(file, "%d\n", &id)
		}

		fmt.Fscanf(file, "%s\n", &eventid)
		log.Printf("RestoreScoremap() eventid=%s, id=%d\n", eventid, id)

		if _, ok := eventmap[eventid]; !ok {
			eventinf, _ := GSE5Mlib.SelectEventInf(eventid)
			eventmap[eventid] = &eventinf
		}

		lastscore.Eventid = eventid
		fmt.Fscanf(file, "%d %d %d %d\n", &lastscore.Score, &lastscore.Rank, &lastscore.Dup, &lastscore.Sum0)
		fmt.Fscanf(file, "%q\n", &ts)
		lastscore.ts, _ = time.Parse("2006/01/02 15:04:05 MST", ts)
		fmt.Fscanf(file, "%q\n", &tstart0)
		lastscore.Tstart0, _ = time.Parse("2006/01/02 15:04:05 MST", tstart0)
		if fver > 0 {
			fmt.Fscanf(file, "%q\n", &tstart1)
			lastscore.Tstart1, _ = time.Parse("2006/01/02 15:04:05 MST", tstart1)
			fmt.Fscanf(file, "%d\n", &lastscore.Continued)
		} else {
			lastscore.Tstart1 = lastscore.Tstart0
			lastscore.Continued = 0
		}
		fmt.Fscanf(file, "%q\n", &lastscore.Qstatus)
		fmt.Fscanf(file, "%q\n", &lastscore.Qtime)
		log.Printf("%v\n%#v %v\n", err, lastscore, lastscore.ts)

		if fver > 1 {
			fmt.Fscanf(file, "%d\n", &lastscore.NoOffline)
			log.Printf("%d\n", lastscore.NoOffline)
		}

		if (*eventmap[eventid]).End_time.Before(time.Now()) {
			log.Printf("ignored eventid=%s, id=%d\n", eventid, id)
			continue
		}

		scoremap[id] = &lastscore

	}

	file.Close()

	return
}

func MakeComment() (status int) {

	status = 0

	var userno [11]int
	var shortname [11]string
	var point [11]int
	var idxtid int

	filet, errt := os.OpenFile("target.txt", os.O_RDONLY, 0644)
	if errt != nil {
		fmt.Println(" Can't open file. [", "target.txt", "]")
		status = 1
		return
	}
	trank := 0
	tid := 0
	fmt.Fscanf(filet, "%d %d", &trank, &tid)
	filet.Close()
	log.Printf("trank=%d, tid=%d\n", trank, tid)

	if tid == 0 {
		//	着目すべき配信者が指定されていない
		return
	}

	for id, lastscore := range scoremap {
		idx := (*lastscore).Rank - 1
		if trank != 0 {
			idx = (*lastscore).Rank - trank + 5
		}
		if idx < 0 || idx > 10 {
			continue
		}
		if id == tid {
			idxtid = idx
		}
		userno[idx] = id
		if sn, ok := snmap[id]; ok {
			shortname[idx] = sn
		} else {
			_, shortname[idx], _, _, _, _, _ = GSE5Mlib.SelectUserName(id)
			snmap[id] = shortname[idx]
		}
		point[idx] = (*lastscore).Score
	}

	log.Printf("idxtid=%d\n", idxtid)

	if idxtid < 0 {
		return
	}

	ib := 0

	if trank == 0 {
		ib := idxtid - 2
		if ib < 0 {
			ib = 0
		}
	} else {
		ib = idxtid
		switch idxtid {
		case 5, 6, 7, 8:
			ib = 4
		case 9:
			ib = 5
		case 10:
			ib = 6
		}
	}

	ie := ib + 4

	if ie > 10 {
		ie = 10
		ib = 6
	}

	log.Printf("ib=%d\n", ib)

	file, err := os.OpenFile("comment.txt", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	if err != nil {
		fmt.Println(" Can't open file. [", "comment.txt", "]")
		status = 1
		return
	}

	for i := ib; i < ie+1; i++ {
		if i != ib {
			fmt.Fprintf(file, "|")
		}
		if point[i] == 0 {
			fmt.Fprint(file, "**")
		} else {
			fmt.Fprintf(file, "%s", shortname[i])
			if i != idxtid {
				fmt.Fprintf(file, " %s", humanize.Comma(int64(point[i]-point[idxtid])))
			} else {
				if trank == 0 {
					fmt.Fprintf(file, "(%d)", idxtid+1)
				} else {
					fmt.Fprintf(file, "(%d)", idxtid+trank-5)
				}
			}
		}
	}
	fmt.Fprint(file, "\n")

	file.Close()

	return
}

/*
	各配信者さんの獲得ポイントのリストを作る（ファイルに追記する）
	ファイルは獲得ポイントを横並びにしたものと、各配信者さんの順位、獲得ポイント、
*/

func ScanActive(gschedule Gschedule) (status int) {

	var stmt *sql.Stmt
	var rows *sql.Rows

	sqlstmt := "select userno, iscntrbpoints from eventuser where eventid = ? and istarget ='Y'"
	stmt, srdblib.Dberr = srdblib.Db.Prepare(sqlstmt)
	if srdblib.Dberr != nil {
		log.Printf("ScanActive() Prepare() err=%s\n", srdblib.Dberr.Error())
		status = -5
		return
	}
	defer stmt.Close()

	rows, srdblib.Dberr = stmt.Query(gschedule.Eventid)
	if srdblib.Dberr != nil {
		log.Printf("ScanActive() Query() (6) err=%s\n", srdblib.Dberr.Error())
		status = -6
		return
	}
	defer rows.Close()

	idlist := make([]string, 0)
	cntrblist := make([]string, 0)
	userno := 0

	iscntrb := "N"
	for rows.Next() {
		Err := rows.Scan(&userno, &iscntrb)

		if Err != nil {
			log.Printf("ScanActive() Scan() err=%s\n", Err.Error())
			status = -7
			return
		}

		idlist = append(idlist, fmt.Sprintf("%d", userno))
		cntrblist = append(cntrblist, iscntrb)

	}

	if srdblib.Dberr = rows.Err(); srdblib.Dberr != nil {
		log.Printf("ScanActive() rows err=%s\n", srdblib.Dberr.Error())
		status = -8
		return
	}

	//	log.Println("ScanActive() idlist=", idlist)
	if len(idlist) != 0 {
		//	status = GetPointsAll(idlist, gschedule, cntrblist)
		go GetPointsAll(idlist, gschedule, cntrblist)
	}

	return

}

func SelectIstargetAndIiscntrbpoint(
	eventid string,
	userno int,
) (
	istarget string,
	iscntrbpoint string,
	status int,
) {

	sqlstmt := "select istarget, iscntrbpoints from eventuser where eventid = ? and userno =?"
	srdblib.Dberr = srdblib.Db.QueryRow(sqlstmt, eventid, userno).Scan(&istarget, &iscntrbpoint)
	if srdblib.Dberr != nil {
		log.Printf("SelectIstargetAndIiscntrbpoint() Prepare() err=%s\n", srdblib.Dberr.Error())
		istarget = "N"
		iscntrbpoint = "N"
		status = -5
	}

	return

}

func CopyScore(gschedule Gschedule) (status int) {

	var stmt *sql.Stmt
	var rows *sql.Rows

	status = 0

	//	svtime := gschedule.Endtime.Add(1 * time.Second)
	svtime := gschedule.Endtime.Add(time.Duration(ConfirmedAt) * time.Second)
	eventid := gschedule.Eventid

	log.Printf("**************** CopyScore() called.\n")

	/*
		sql := "select distinct max(ts) from points where eventid = ?"
		stmt, err := Db.Prepare(sql)
		if err != nil {
			log.Printf("CopyScore() (3) err=%s\n", err.Error())
			status = -3
			return
		}
		defer stmt.Close()

		//	idx := 0
		var gtime time.Time
		stmt.QueryRow(eventid).Scan(&gtime)
		if err != nil {
			log.Printf("CopyScore() (4) err=%s\n", err.Error())
			status = -4
			return
		}
	*/
	//	var gtime time.Time
	var nullgtime sql.NullTime
	var gtime time.Time
	sqlstmt := "select distinct max(ts) from points where eventid = ?"
	srdblib.Dberr = srdblib.Db.QueryRow(sqlstmt, eventid).Scan(&nullgtime)
	if srdblib.Dberr != nil {
		log.Printf("CopyScore() (4) err=%s\n", srdblib.Dberr.Error())
		status = -4
		return
	}
	if nullgtime.Valid {
		gtime = nullgtime.Time
	} else {
		//	データが存在しないイベントの場合
		log.Printf("CopyScore() (4) gtime is null.\no")
		status = -4
		return
	}

	log.Printf("gtime=%s\n", gtime.Format("2006/01/02 15:04:06"))

	if gtime.Before(gschedule.Endtime) {
		//	終了処理が行われていない。

		//	---------------------------------------------------

		//	データ取得プロセスが途中で落ちた場合、イベント終了直前のデータは存在しないので
		//	下記コメントにした部分が生きていると終了処理は行われないことになってしますのだが...
		//
		//	if gtime.Before(gschedule.Endtime.Add(-15 * time.Minute)) {
		//		//	最新データ（＝イベント終了直前のデータ）が存在しない。
		//		return
		//	}

		stmt, srdblib.Dberr = srdblib.Db.Prepare("select user_id, `rank`, point from points where eventid = ? and ts = ?")
		if srdblib.Dberr != nil {
			log.Printf("CopyScore() (5) err=%s\n", srdblib.Dberr.Error())
			status = -5
			return
		}
		defer stmt.Close()

		rows, srdblib.Dberr = stmt.Query(eventid, gtime)
		if srdblib.Dberr != nil {
			log.Printf("CopyScore() (6) err=%s\n", srdblib.Dberr.Error())
			status = -6
			return
		}
		defer rows.Close()

		var score GSE5Mlib.CurrentScore
		var scorelist []GSE5Mlib.CurrentScore

		i := 0

		for rows.Next() {
			srdblib.Dberr = rows.Scan(&score.Userno, &score.Rank, &score.Point)
			if srdblib.Dberr != nil {
				log.Printf("CopyScore() (7) err=%s\n", srdblib.Dberr.Error())
				status = -7
				return
			}
			scorelist = append(scorelist, score)
			i++
		}
		if srdblib.Dberr = rows.Err(); srdblib.Dberr != nil {
			log.Printf("CopyScore() (8) err=%s\n", srdblib.Dberr.Error())
			status = -8
			return
		}

		for _, score = range scorelist {
			var roominf GSE5Mlib.RoomInfo
			roominf.Userno = score.Userno
			roominf.Point = score.Point
			InsertIntoOrUpdatePoints(svtime, roominf, score.Rank, 0, eventid, "Prov.", "", "", "")
			/*
				_, iscntrbpoint, _ := SelectIstargetAndIiscntrbpoint(eventid, score.Userno)
				if iscntrbpoint == "Y" {
					//	イベント配信者設定で貢献ポイントランキングを取得すると設定されている場合
					log.Printf("  InsertIntoTimeTable() called. eventid=%s userno=%d\n", eventid, score.Userno)
					//	最後の2つの引数はダミー 4月13日までに修正のこと
					InsertIntoTimeTable(eventid, score.Userno, svtime, 0, time.Now(), time.Now())
				}
			*/

		}
	}

	//	終了処理が行われていてもこのパスを通るのはデータの整合性が失われた（失わせた）ケース。

	sqlstmt = "update event set rstatus = ? where eventid = ?"
	_, srdblib.Dberr = srdblib.Db.Exec(sqlstmt, "Provisional", eventid)

	if srdblib.Dberr != nil {
		log.Printf("CopyScore() update event err=[%s]\n", srdblib.Dberr.Error())
		status = -1
	}

	return
}

func GetConfirmed(gschedule Gschedule) (status int) {

	var eventinf GSE5Mlib.Event_Inf
	var roominflist GSE5Mlib.RoomInfoList
	//	var roominf RoomInfo

	log.Printf("**************** GetConfirmed() called.\n")

	status = 0

	//	svtime := gschedule.Endtime.Add(1 * time.Second)
	svtime := gschedule.Endtime.Add(time.Duration(ConfirmedAt) * time.Second)
	eventid := gschedule.Eventid
	ieventid := gschedule.Ieventid

	//	イベントに参加しているルームの一覧を取得します。
	//	ルーム名、ID、URLを取得しますが、イベント終了直後の場合の最終獲得ポイントが表示されている場合はそれも取得します。
	breg := 1
	//	確定値（最終獲得ポイント）が発表されるのは30位まで。確定値が発表されないイベントもあるので要注意。
	ereg := 30
	isquest, status := GSE5Mlib.GetEventInfAndRoomList(eventid, ieventid, breg, ereg, &eventinf, &roominflist)

	isconfirm := false
	for i, roominf := range roominflist {

		//	log.Printf(" i+1=%d, userno=%d, point=%d\n", i+1, roominf.Userno, roominf.Point)
		if roominf.Point > 0 {
			//	最終獲得ポイントが発表された場合のみ更新する
			//	InsertIntoOrUpdatePoints(svtime, roominf.Userno, roominf.Point, i+1, 0, eventid, "Conf.", "", "", "")
			InsertIntoOrUpdatePoints(svtime, roominf, i+1, 0, eventid, "Conf.", "", "", "")
			isconfirm = true
		}
	}

	log.Printf("  isconfirm =%t, isquest=%t\n", isconfirm, isquest)
	if isconfirm || isquest {
		sqlstmt := "update event set rstatus = ? where eventid = ?"
		_, srdblib.Dberr = srdblib.Db.Exec(sqlstmt, "Confirmed", eventid)

		if srdblib.Dberr != nil {
			log.Printf("GetConfirmed() update event err=[%s]\n", srdblib.Dberr.Error())
			status = -1
			return
		}

		if isconfirm {
			SRDBlib.MakePointPerSlot(eventid)
		}
	}

	return

}

/*
func GetEventInfo() {

	return
}
*/

func GetSchedule() (
	gschedulelist Gschedulelist,
	status int,
) {

	var stmt *sql.Stmt
	var rows *sql.Rows

	//	eventno := 0
	eventid := ""

	tnow := time.Now()

	sqlstmt := "select eventid, ieventid, starttime, endtime, rstatus from event where endtime > ? "
	stmt, Err := srdblib.Db.Prepare(sqlstmt)
	if Err != nil {
		log.Printf("GetSchedule() Prepare() err=%s\n", Err.Error())
		status = -5
		return
	}
	defer stmt.Close()

	//	48時間マイナスしてあるのは、翌日発表の確定値を取得する必要あるイベントも含めるため
	rows, Err = stmt.Query(tnow.Add(-48 * time.Hour))
	if Err != nil {
		log.Printf("GetSchedule() Query() (6) err=%s\n", Err.Error())
		status = -6
		return
	}
	defer rows.Close()

	var gschedule Gschedule
	var starttime, endtime time.Time
	var rstatus string
	var ieventid int

	i := 0
	for rows.Next() {
		Err = rows.Scan(&eventid, &ieventid, &starttime, &endtime, &rstatus)

		if Err != nil {
			log.Printf("GetSchedule() Scan() err=%s\n", Err.Error())
			status = -7
			return
		}

		//	log.Printf(" eventid=%s rstatus=%s\n", eventid, rstatus)
		if rstatus == "Confirmed" {
			//	確定した最終結果がすでに保存されたイベントは対象ではない。
			continue
		}

		end_date := endtime.Truncate(time.Hour).Add(-time.Duration(endtime.Hour())*time.Hour).AddDate(0, 0, 1)
		//	log.Printf("tnow= %s end_date=%s (%s)\n", tnow.Format("2006-01-02 15:04:05"), end_date.Format("2006-01-02 15:04:05"), eventid)

		//	rstatusを書き換えて、終了処理をやり直すことができるように条件を設定してある。
		if tnow.Before(endtime.Add(time.Duration(gschedule.Intervalmin*2+1)*time.Minute)) || rstatus == "" {
			//	イベント期間中は獲得ポイントデータを取得する。
			gschedule.Method = "GetScore"
		} else if tnow.After(endtime.Add(1*time.Minute)) && rstatus != "Provisional" {
			//	イベント終了後、最終結果を格納するためのレコードを一回だけ追加する。
			gschedule.Method = "CopyScore"
		} else if rstatus == "Provisional" && tnow.After(end_date.Add(660*time.Minute)) {
			//	イベント終了時を含む日の24時00分から11時間経過し、最終結果格納用のレコードが作成済みである。
			gschedule.Method = "GetConfirmed"
		} else {
			//	イベント終了後最終結果格納用レコードが作成されたが終了日から11時間経過していない。
			continue
		}
		//	log.Printf("tnow=%s Method=%s\n", tnow.Format("2006-01-02 15:04"), gschedule.Method)

		//	log.Printf("eventno=%d, eventid=%s\n", eventno, eventid)
		gschedule.Eventid = eventid
		if starttime.Before(time.Now()) {
			gschedule.Beforestart = false
		} else {
			gschedule.Beforestart = true
		}
		gschedule.Eventid = eventid
		gschedule.Ieventid = ieventid
		gschedule.Endtime = endtime
		gschedule.Done = false
		gschedulelist = append(gschedulelist, gschedule)

		i++
	}

	if Err = rows.Err(); Err != nil {
		log.Printf("GetSchedule() rows err=%s\n", Err.Error())
		status = -8
		return
	}

	//	=================================================

	for i := 0; i < len(gschedulelist); i++ {

		sqlstmt := "select intervalmin, modmin, modsec from event where eventid = ?"
		Err = srdblib.Db.QueryRow(sqlstmt, gschedulelist[i].Eventid).Scan(&gschedulelist[i].Intervalmin, &gschedulelist[i].Modmin, &gschedulelist[i].Modsec)
		if Err != nil {
			log.Printf("GetSchedule() select err=[%s]\n", Err.Error())
			status = -1
		}
		if gschedulelist[i].Intervalmin == 0 {
			//	Intervaminが0だと剰余を求めるときゼロ割りが起きる。
			gschedulelist[i].Intervalmin = 5
		}

	}

	//	log.Println(gschedulelist)

	return

}
func main() {

	//	eventmap = make(map[string]int)
	eventmap = make(map[string]*GSE5Mlib.Event_Inf)
	//	parameters.eventmap = &eventmap

	snmap = make(map[int]string)
	scoremap = map[int]*LastScore{}

	logfilename := version + "_" + GSE5Mlib.Version + "_" + SRDBlib.Version + "_" + time.Now().Format("20060102") + ".txt"
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
	srdblib.Dbmap.AddTableWithName(srdblib.Points{}, "points").SetKeys(false, "Eventid", "User_id", "Ts")

	RestoreScoremap()

	var gschedulelist Gschedulelist

	//	hh, _, ss := time.Now().Clock()
	_, _, ss := time.Now().Clock()
	if ss != 0 {
		time.Sleep(time.Duration(61-ss) * time.Second)
	}
	st := time.Now()
	//	t := st
	_, mm, _ := st.Clock()
	log.Printf(" start time=%s\n", st.Format("2006-01-02 15:04:05"))

	status := 0

//	outerloop:
	for {
		gschedulelist, status = GetSchedule()
		if status != 0 {
			log.Printf("GetSchedule() status=%d\n", status)
			return
		}
		//	fmt.Printf("now=%s t=%s status=%d len=%d\n", time.Now().Format("2006/01/02 15:04:05"), t.Format("2006/01/02 15:04:05"), status, len(gschedulelist))
		for {
			nextsec := 99
			idx := -1

			for i := 0; i < len(gschedulelist); i++ {
				if gschedulelist[i].Done {
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
				//	hh, mm, ss = time.Now().Clock()
				_, mm, ss = time.Now().Clock()
				if ss < nextsec {
					time.Sleep(time.Duration(nextsec-ss) * time.Second)
				}
				log.Printf(" eventid=%s method=%s\n", gschedulelist[idx].Eventid, gschedulelist[idx].Method)
				switch gschedulelist[idx].Method {
				case "GetScore":
					ScanActive(gschedulelist[idx])
				case "CopyScore":
					CopyScore(gschedulelist[idx])
				case "GetConfirmed":
					GetConfirmed(gschedulelist[idx])
				}
				gschedulelist[idx].Done = true
			} else {
				break
			}
		}

		/*
		//	毎日偶数時 5分に特定ユーザーのユーザー情報を取得する
		//	レベルやフォロワー数の推移を記録する
		//	if hh%6 == 3 && mm == 1 {
		if hh%2 == 0 && mm == 5 {
			GSE5Mlib.GetUserInfForHistory()
		}
		*/

		//	毎分00秒になるまで待つ
		_, _, ss = time.Now().Clock()
		time.Sleep(time.Duration(61-ss) * time.Second)
		//	t = time.Now()
		//	hh, mm, _ = time.Now().Clock()
		_, mm, _ = time.Now().Clock()
		hh24 := mm
		if hh24 == 0 {
			hh24 = 24
		}
		/*
		if hh24%GSE5Mlib.Dbconfig.TimeLimit == 0 && mm == 0 {
			//	一定時間経ったら処理を終了する
			break outerloop
		}
		*/
	}
	//	log.Printf(" end time=%s\n", t.Format("2006-01-02 15:04:05"))
}
