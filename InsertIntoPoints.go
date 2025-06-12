// Copyright © 2022-2025 chouette2100@gmail.com
// Released under the MIT license
// https://opensource.org/licenses/mit-license.php

package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"database/sql"

	_ "github.com/go-sql-driver/mysql"

	"SRGSE5M/GSE5Mlib"

	"github.com/dustin/go-humanize"

	"github.com/Chouette2100/exsrapi/v2"
	"github.com/Chouette2100/srdblib/v2"
)

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

	var err error
	status = 0

	//	log.Printf("InsertIntoPoints()　db.Prepare()\n")
	var stmt *sql.Stmt
	stmt, err = tx.Prepare("INSERT INTO points(ts, user_id, eventid, point, `rank`, gap, pstatus, ptime, qstatus, qtime) VALUES(?,?,?,?,?,?,?,?,?,?)")
	if err != nil {
		log.Printf("InsertIntoPoints() select err=[%s]\n", err.Error())
		status = -1
	}
	defer stmt.Close()

	//	log.Printf("InsertIntoPoints()　row.Exec("InsertIntoOrUpdate",...)\n")
	//	log.Printf("timestamp=%v, userno=%v, eventid=%v, point=%v, rank=%v, gap=%v, pstatus=%v, ptime=%v, qstatus=%v, qtime=%v\n", timestamp, userno, eventid, point, rank, gap, pstatus, ptime, qstatus, qtime)
	_, err = stmt.Exec(timestamp, userno, eventid, point, rank, gap, pstatus, ptime, qstatus, qtime)

	if err != nil {
		log.Printf("InsertIntoPoints() insert into points err=[%s]\n", err.Error())
		status = -1
	}

	//	===============================================
	sqlstmt := "update eventuser set point = ? where eventid = ? and userno = ?"
	_, err = tx.Exec(sqlstmt, point, eventid, userno)

	if err != nil {
		log.Printf("InsertIntoPoints() update eventuser err=[%s]\n", err.Error())
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

	var err error
	status = 0

	nrow := 0
	sqlstmt := "select count(*) from points where ts = ? and eventid = ? and user_id= ?"
	err = srdblib.Db.QueryRow(sqlstmt, timestamp, eventid, roominf.Userno).Scan(&nrow)
	if err != nil {
		log.Printf("InsertIntoOrUpdatePoints() select err=[%s]\n", err.Error())
		status = -1
	}
	if nrow == 0 {
		//	log.Printf("InsertIntoOrUpdatePoints()　db.Prepare()\n")
		var stmt *sql.Stmt
		stmt, err = srdblib.Db.Prepare("INSERT INTO points(ts, user_id, eventid, point, `rank`, gap, pstatus, ptime, qstatus, qtime) VALUES(?,?,?,?,?,?,?,?,?,?)")
		if err != nil {
			log.Printf("InsertIntoOrUpdatePoints() select err=[%s]\n", err.Error())
			status = -1
		}
		defer stmt.Close()

		//	log.Printf("InsertIntoOrUpdatePoints()　row.Exec("InsertIntoOrUpdate",...)\n")
		_, err = stmt.Exec(timestamp, roominf.Userno, eventid, roominf.Point, rank, gap, pstatus, ptime, qstatus, qtime)

		if err != nil {
			log.Printf("InsertIntoOrUpdatePoints() select err=[%s]\n", err.Error())
			status = -1
		}
	} else {
		sqlstmt = "update points set point = ?, `rank`=?, gap=?, pstatus=?, ptime =?, qstatus=?, qtime=? where ts=? and eventid=? and user_id=?"
		_, err = srdblib.Db.Exec(sqlstmt, roominf.Point, rank, gap, pstatus, ptime, qstatus, qtime, timestamp, eventid, roominf.Userno)

		if err != nil {
			log.Printf("InsertIntoOrUpdatePoints() update points err=[%s]\n", err.Error())
			status = -1
		}
	}

	//	===============================================
	nrow = 0
	sqlstmt = "select count(*) from eventuser where eventid = ? and userno = ?"
	err = srdblib.Db.QueryRow(sqlstmt, eventid, roominf.Userno).Scan(&nrow)
	if err != nil {
		log.Printf("InsertIntoOrUpdatePoints() select err=[%s]\n", err.Error())
		status = -1
	}

	if nrow != 0 {
		sqlstmt = "update eventuser set point = ? where eventid = ? and userno = ?"
		_, err = srdblib.Db.Exec(sqlstmt, roominf.Point, eventid, roominf.Userno)

		if err != nil {
			log.Printf("InsertIntoOrUpdatePoints() update eventuser err=[%s]\n", err.Error())
			status = -1
		}

	} else {
		sqlstmt = "insert into eventuser (eventid, userno, istarget, iscntrbpoints, graph, color, point) values (?,?,?,?,?,?,?)"
		_, err = srdblib.Db.Exec(sqlstmt, eventid, roominf.Userno, "N", "N", "N", "white", roominf.Point)
		if err != nil {
			log.Printf("InsertIntoOrUpdatePoints() insert into eventuser err=[%s]\n", err.Error())
			status = -1
		}
	}

	nrow = 0
	sqlstmt = "select count(*) from user where userno = ?"
	err = srdblib.Db.QueryRow(sqlstmt, roominf.Userno).Scan(&nrow)
	if err != nil {
		log.Printf("InsertIntoOrUpdatePoints() select err=[%s]\n", err.Error())
		status = -1
	}

	log.Printf("%s userno=%d nrow=%d\n", eventid, roominf.Userno, nrow)

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

	var err error

	sql := "delete from points where eventid = ? and ts= ? and user_id = ?"

	//	log.Printf("Db.Exec(\"delete ...\")\n")
	_, err = tx.Exec(sql, eventid, ts, user_id)

	if err != nil {
		log.Printf("DeleteFromPoints() select err=[%s]\n", err.Error())
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

	var err error
	status = 0

	log.Printf("InsertIntoTimeTable() called. eventid=%s, userno =%d st1=%v\n", eventid, userno, st1)

	var stmt *sql.Stmt
	sql := "INSERT INTO timetable(eventid, userid, sampletm1, stime, etime, target, earnedpoint, status)"
	sql += " VALUES(?,?,?,?,?,?,?,?)"
	stmt, err = srdblib.Db.Prepare(sql)
	if err != nil {
		log.Printf("InsertIntoPoints() prepare() err=[%s]\n", err.Error())
		status = -1
	}
	defer stmt.Close()

	_, err = stmt.Exec(eventid, userno, st1, stime, etime, -1, earnedp, 0)

	if err != nil {
		log.Printf("InsertIntoEventrank() exec() err=[%s]\n", err.Error())
		status = -1
	}

	return
}

func MakeComment() (status int) {

	status = 0

	//	var userno [11]int
	var shortname [11]string
	var point [11]int
	var idxtid int

	filet, errt := os.OpenFile("target.txt", os.O_RDONLY, 0644)
	if errt != nil {
		log.Println(" Can't open file. [", "target.txt", "]")
		status = 1
		return
	}
	trank := 0
	tid := 0
	fmt.Fscanf(filet, "%d %d", &trank, &tid)
	filet.Close()
	//	log.Printf("trank=%d, tid=%d\n", trank, tid)

	if tid == 0 {
		//	着目すべき配信者が指定されていない
		return
	}

	//	for id, lastscore := range scoremap {
	scoremap.Range(func(id, value interface{}) bool {
		//	fmt.Println("key:", id, " value:", value)
		ls := value.(*LastScore)

		idx := ls.Rank - 1
		if trank != 0 {
			idx = ls.Rank - trank + 5
		}
		//	if idx < 0 || idx > 10 {
		if idx >= 0 && idx <= 10 {
			if id == tid {
				idxtid = idx
			}
			//	userno[idx] = id
			if sn, ok := snmap.Load(id.(int)); ok {
				shortname[idx] = sn.(string)
			} else {
				_, shortname[idx], _, _, _, _, _ = GSE5Mlib.SelectUserName(id.(int))
				snmap.Store(id.(int), shortname[idx])
			}
			point[idx] = ls.Score
		}
		return true

	})

	//	log.Printf("idxtid=%d\n", idxtid)

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

	//	log.Printf("ib=%d\n", ib)

	file, err := os.OpenFile("comment.txt", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	if err != nil {
		log.Println(" Can't open file. [", "comment.txt", "]")
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

func SelectIstargetAndIiscntrbpoint(
	eventid string,
	userno int,
) (
	istarget string,
	iscntrbpoint string,
	status int,
) {

	var err error

	sqlstmt := "select istarget, iscntrbpoints from eventuser where eventid = ? and userno =?"
	err = srdblib.Db.QueryRow(sqlstmt, eventid, userno).Scan(&istarget, &iscntrbpoint)
	if err != nil {
		log.Printf("SelectIstargetAndIiscntrbpoint() Prepare() err=%s\n", err.Error())
		istarget = "N"
		iscntrbpoint = "N"
		status = -5
	}

	return

}

func CopyScore(gschedule Gschedule) (status int) {

	var err error
	var stmt *sql.Stmt
	var rows *sql.Rows

	//	cmt0 := "=========="
	fncname := exsrapi.FuncNameOfThisFunction(1) + "()"

	//	fncname := "GetConfirmed()"
	cmt0 := gschedule.Eventid
	log.Println(cmt0, ">>>>>>>>>>>>>>>>>>", fncname, ">>>>>>>>>>>>>>>>>>>")
	defer exsrapi.PrintExf(cmt0, fncname)()

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
	err = srdblib.Db.QueryRow(sqlstmt, eventid).Scan(&nullgtime)
	if err != nil {
		log.Printf("CopyScore() (4) err=%s\n", err.Error())
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

	if gtime.Before(gschedule.Endtime.Add(58 * time.Second)) {
		//	終了処理が行われていない。

		//	---------------------------------------------------

		//	データ取得プロセスが途中で落ちた場合、イベント終了直前のデータは存在しないので
		//	下記コメントにした部分が生きていると終了処理は行われないことになってしますのだが...
		//
		//	if gtime.Before(gschedule.Endtime.Add(-15 * time.Minute)) {
		//		//	最新データ（＝イベント終了直前のデータ）が存在しない。
		//		return
		//	}

		stmt, err = srdblib.Db.Prepare("select user_id, `rank`, point from points where eventid = ? and ts = ?")
		if err != nil {
			log.Printf("CopyScore() (5) err=%s\n", err.Error())
			status = -5
			return
		}
		defer stmt.Close()

		rows, err = stmt.Query(eventid, gtime)
		if err != nil {
			log.Printf("CopyScore() (6) err=%s\n", err.Error())
			status = -6
			return
		}
		defer rows.Close()

		var score GSE5Mlib.CurrentScore
		var scorelist []GSE5Mlib.CurrentScore

		i := 0

		for rows.Next() {
			err = rows.Scan(&score.Userno, &score.Rank, &score.Point)
			if err != nil {
				log.Printf("CopyScore() (7) err=%s\n", err.Error())
				status = -7
				return
			}
			scorelist = append(scorelist, score)
			i++
		}
		if err = rows.Err(); err != nil {
			log.Printf("CopyScore() (8) err=%s\n", err.Error())
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
	} else {
		log.Printf("CopyScore() (9) err=Unexpected data found\n")
		status = -9
	}

	//	終了処理が行われていてもこのパスを通るのはデータの整合性が失われた（失わせた）ケース。

	sqlstmt = "update event set rstatus = ? where eventid = ?"
	_, err = srdblib.Db.Exec(sqlstmt, "Provisional", eventid)

	if err != nil {
		log.Printf("CopyScore() update event err=[%s]\n", err.Error())
		status = -1
	}

	return
}
