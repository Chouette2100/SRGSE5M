// Copyright © 2022-2024 chouette.21.00@gmail.com
// Released under the MIT license
// https://opensource.org/licenses/mit-license.php
package main

import (
	//	"crypto/aes"
	"fmt"
	"log"
	//	"os"
	"strconv"
	"strings"
	//	"sync"
	"time"

	//	. "log"
	//	"bufio"
	//	"io"

	//	"runtime"

	"net/http"

	"database/sql"

	_ "github.com/go-sql-driver/mysql"

	//	"github.com/go-gorp/gorp"

	//	"encoding/json"
	//	"github.com/360EntSecGroup-Skylar/excelize"

	//	. "MyModule/ShowroomCGIlib"
	//	"SRGSE5M/GSE5Mlib"
	//	"SRGSE5M/SRDBlib"

	"github.com/dustin/go-humanize"

	"github.com/Chouette2100/exsrapi"
	"github.com/Chouette2100/srapi"
	"github.com/Chouette2100/srdblib"
)

/*
	各配信者さんの獲得ポイントのリストを作る（ファイルに追記する）
	ファイルは獲得ポイントを横並びにしたものと、各配信者さんの順位、獲得ポイント、
*/

func ScanActive(client *http.Client, gschedule Gschedule) (status int) {

	var stmt *sql.Stmt
	var rows *sql.Rows

	cmt0 := gschedule.Eventid
	fncname := exsrapi.FuncNameOfThisFunction() + "()"
	//	fncname := "ScanActive()"
	log.Println(cmt0, ">>>>>>>>>>>>>>>>>>", fncname, ">>>>>>>>>>>>>>>>>>>")
	defer exsrapi.PrintExf(cmt0, fncname)()

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
		GetPointsAll(client, idlist, gschedule, cntrblist)
	}

	return

}

/*
配信者のリストからそれぞれの獲得ポイントなどを取得する。
*/
func GetPointsAll(client *http.Client, IdList []string, gschedule Gschedule, cntrblist []string) (status int) {

	//	cmt0 := gschedule.Eventid
	//	fncname := exsrapi.FuncNameOfThisFunction() + "()"
	//	log.Println(cmt0, ">>>>>>>>>>>>>>>>>>", fncname, ">>>>>>>>>>>>>>>>>>>")
	//	defer exsrapi.PrintExf(cmt0, fncname)()

	status = 0

	//	if gschedule.Eventid != "greatdetective?block_id=0" {
	//		return
	//	}

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

	//	以下の二つの理由でIdListのmapを作っておく
	//		指定した順位の範囲のルームがIdListに存在するかチェックする
	//		データ保存済みのルームをマークする
	umap := make(map[int]bool)
	for _, uno := range IdList {
		userno, _ := strconv.Atoi(uno)
		umap[userno] = false
	}

	//	eventid := gschedule.Eventid
	//	eida := strings.Split(eventid, "?")
	var pranking *srapi.Eventranking
	//	pmap := make(map[int]int)
	//	var qranking *srapi.EventBlockRanking
	//	plist := make([]srdblib.Points, 100)
	var err error
	//	50位までのルームの順位、ポイントを取得する。
	//	イベント開催中でない場合はエラーとなる
	//  TODO: イベント終了後も取得可能なのでは？
	pranking, err = srdblib.GetEventsRankingByApi(client, gschedule.Eventid, 1)
	if err != nil {
		log.Printf("GetPointsAll() GetEventsRankingByApi() err=[%s]\n", err.Error())
		return -1
	}

	log.Printf("GetPointsAll() GetEventsRankingByApi() =%d\n", len(pranking.Ranking))

	if len(pranking.Ranking) == 0 {
		roomlistinf, err := srapi.GetRoominfFromEventByApi(
			client,
			gschedule.Ieventid, //	Event_id (int)
			gschedule.Fromorder,
			gschedule.Toorder,
		)
		if err != nil {
			err = fmt.Errorf("srapi.GetRoominfFromEventByApi() returned error. %w", err)
			log.Printf("GetPointsAll() srapi.GetRoominfFromEventByApi() err=[%s]\n", err.Error())
		}

		log.Printf("GetPointsAll() srapi.GetRoominfFromEventByApi() =%d\n", len(roomlistinf.RoomList))

		for _, room := range roomlistinf.RoomList {
			userno := room.Room_id
			if _, ok := umap[userno]; !ok {
				srdblib.UpinsEventuser(client, -1, 0, gschedule.Eventid, gschedule.Starttime, userno, timestamp)
				IdList = append(IdList, strconv.Itoa(userno))
			}
			//	cntrblist := append(cntrblist, "N")
			//	//		GetPointsAll(idlist, gschedule, cntrblist)
			//	GetPointsAll(client, idlist, gschedule, cntrblist)
			//	time.Sleep(time.Duration(gschedule.Intervalmin+1) * time.Minute)
		}
	}

	//	ブロックイベントのときは100位までの順位を取得する
	//	block_id ==0 のときは51位より下位のルームの順位はこの方法でないとわからない
	//	またusernoから順位を取得できるようにmapを作っておく
	eida := strings.Split(gschedule.Eventid, "?block_id=")
	qmap := make(map[int]int)
	blockid := -1
	qlist := new([]srapi.Block_ranking)
	if len(eida) == 2 {
		blockid, _ = strconv.Atoi(eida[1])
		qranking, err := srapi.GetEventBlockRanking(client, gschedule.Ieventid, blockid, 1, 100)
		if err != nil {
			log.Printf("GetPointsAll() GetEventBlockRanking() err=[%s]\n", err.Error())
		} else {
			qlist = &qranking.Block_ranking_list
			for i, ranking := range qranking.Block_ranking_list {
				quno, _ := strconv.Atoi(ranking.Room_id)
				qmap[quno] = i
			}
		}
	}
	//	Fromorder: 1, Toorder: 4

	//	IdListからumapを作る
	//	IdList:   100, 101, 102, 103
	//	umap:  [100]=false, [101]=false, [102]=false, [103]=false
	//
	//  pranking: 100, 200, 101, 102, 300, 103,...

	//  pranking: 100, 200, 101, 102, 300, 103,...
	//	Fromorder、Toorderの範囲のprankingのデータを保存する
	//	umap:	[100]=true, [200]=true, [101]=true, [102]=true
	//	pmap: [100]=0, [200]=1, [101]=2, [102]=3, [300]=4, [103]=5

	//	取得したprankingからIdListに該当するもので未保存のものを保存する
	//	umap:	[100]=true, [200]=true, [101]=true, [102]=true, [103]=true

	//	usernoから結果を取得できるようにmapを作っておく
	plist := make([]srdblib.Points, 0, 50)
	pmap := make(map[int]int)

	//	指定した範囲の順位にあるルームのポイントを保存する。
	for i := gschedule.Fromorder - 1; i < gschedule.Toorder; i++ {
		if i >= len(pranking.Ranking) {
			break
		}
		ranking := pranking.Ranking[i]
		if ranking.Point == 0 {
			break
		}
		userno := ranking.Room.RoomID
		if _, ok := umap[userno]; !ok {
			log.Printf("GetPointsAll() userno=%d is not in IdList\n", userno)
			srdblib.UpinsEventuser(client, ranking.Rank, ranking.Point, gschedule.Eventid, gschedule.Starttime, userno, timestamp)
			IdList = append(IdList, strconv.Itoa(userno))
		}
	}

	log.Printf("GetPointsAll() GetPointsAll() =%+v\n", IdList)

	for i, ranking := range pranking.Ranking {
		pmap[ranking.Room.RoomID] = i
		plist = append(plist, srdblib.Points{
			Eventid: gschedule.Eventid,
			User_id: ranking.Room.RoomID,
			Point:   ranking.Point,
			Rank:    ranking.Rank,
		})
	}

	for _, userid := range IdList {
		userno, _ := strconv.Atoi(userid)
		if _, ok := pmap[userno]; ok {
			continue
		} else {
			log.Printf("GetPointsAll() userno=%d is not in pranking\n", userno)
			//	eventuserには存在するが上位50位のデータには存在しないルーム
			//	point, rank, gap, eventid := GSE5Mlib.GetPointsByAPI(userid)
			point, rank, gap, _, eventid, _, bid, err := srapi.GetPointByApi(client, userno)
			if err != nil {
				log.Printf("GetPointsAll() GetPointByApi() userno=%d err=[%s]\n", userno, err.Error())
				continue
			}
			if eventid != eida[0] || (len(eida) == 2 && blockid != 0 && bid != blockid) {
				log.Printf("GetPointsAll() GetPointByApi() userno=%d eventid=%s blockid=%d\n", userno, eventid, bid)
				//	イベントを変更した等、このイベントにはエントリーしていないルーム
				//	GetPointsByAPI()で取得するeventidにはblock_idは入っていない
				continue
			}
			pmap[userno] = len(plist)
			plist = append(plist, srdblib.Points{
				Eventid: eventid,
				User_id: userno,
				Point:   point,
				Rank:    rank,
				Gap:     gap,
			})

		}

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
	//	log.Printf("%+v\n", IdList)
	for i := 0; i < Length; i++ {
		//	for i, p := range(plist) {

		//	var makePQ func()
		id, _ := strconv.Atoi(IdList[i])
		//	log.Printf("id=%d\n", id)
		//	id := p.User_id

		//	開催されていないイベントに対する設定を兼ねる変数定義
		eventid := gschedule.Eventid

		var isonlive bool
		var startedat time.Time

		point := 0
		rank := 0
		gap := 0
		if !gschedule.Beforestart {
			//	開催されているイベント
			uno, _ := strconv.Atoi(IdList[i])
			if idx, ok := pmap[uno]; ok {
				p := plist[idx]
				point = p.Point
				rank = p.Rank
				gap = p.Gap
				eventid = gschedule.Eventid
				if blockid == 0 {
					//	ブロックIDが0の場合はGetPointByApi()で取得した順位がエントリーしたブロックでの順位であることに注意
					//	OPTIMIZE: 以下の処理はブロックIDがが0で順位が50位より下のケース、工夫が足りない？
					if idxq, ok := qmap[uno]; ok {
						//	log.Printf(" rank=%d, qrank=%d\n", rank, (*qlist)[idxq].Rank)
						rank = (*qlist)[idxq].Rank
					} else {
						rank = 9999
					}
				}
			} else {
				continue
				//		//	ランキングイベントで50位以内にないルームとレベルイベント-のルームの情報は個別に取得する。
				//		point, rank, gap, eventid = GSE5Mlib.GetPointsByAPI(IdList[i])
			}
			/*
				eida := strings.Split(eventid, "?block_id=")
				gida := strings.Split(gschedule.Eventid, "?block_id=")
				if len(eida) == 2 && eida[0] == gida[0] {
					//	この条件は暫定
					//	下記の条件に加え、?block_id=0 がついていないイベントIDもブロックイベント全体を示すことを含んでいる。
					//	（これは一時的な回避方法で発生した）
					//	正しくは
					//	if len(gida) == 2 && len(eida) == 2 && gida[1] == 0 && eida[0] == gida[0] {
					//	bloc_id=0 はブロックイベントに含まれるすべてのイベントを意味している。
					eventid = gschedule.Eventid

					//	len(gida) == 2 のとき
					//		gida[1] != "0" ならば通常のブロックイベント
					//		gida[1] == "0" ならばそのイベントに属するすべてのイベントをまとめたもの
					//			ここで eida[1] != "0" であれば個別のイベントの結果を取得したことになるので rank = 0 とする
					//	len(gida) ==1 && eida[1] != "0" の場合も同様
					if eida[1] != "0" && (len(gida) == 1 || len(gida) == 2 && gida[1] == "0") {
						rank = 0
					}
				}
			*/
			//	if !strings.Contains(eventid, gschedule.Eventid) {
			if !strings.Contains(gschedule.Eventid, eventid) {
				//	イベントがデータ取得対象のイベントではない
				//	Ver. RU20G4	配信中にイベントが終了したら貢献ポイントを取得する。
				log.Printf("%s isn't gschedule.Eventid(%s) .\n", eventid, gschedule.Eventid)
				dup := -9
				//	if _, ok := scoremap[id]; ok {
				//		dup = scoremap[id].Dup
				if _, ok := scoremap.Load(id); ok {
					ls, _ := scoremap.Load(id)
					dup = ls.(*LastScore).Dup
				}
				log.Printf("%s timestamp=%v gschedule.Endtime=%v scoremap[id].Dup=%d\n", eventid, timestamp, gschedule.Endtime, dup)
				if timestamp.After(gschedule.Endtime) {
					//	イベントが終了している。
					//	if scoremap[id].Dup == 0 {	// 該当scoremap[id]が存在しない場合異常終了する。
					if dup == 0 {
						//	配信中のイベント終了であるので貢献ランキングを取得する。
						//	RU20G6 InsertIntoTimeTable(eventid, id, timestamp.Add(15 * time.Minute), (*scoremap[id]).Sum0, (*scoremap[id]).Tstart0, gschedule.Endtime)
						if cntrblist[i] == "Y" {
							//	イベント配信者設定で貢献ポイントランキングを取得すると設定されている場合
							//	InsertIntoTimeTable(gschedule.Eventid, id, timestamp.Add(15*time.Minute), (*scoremap[id]).Sum0, (*scoremap[id]).Tstart0, gschedule.Endtime)
							//	scoremap[id].Dup = -1
							ls, _ := scoremap.Load(id)
							InsertIntoTimeTable(
								gschedule.Eventid, id,
								timestamp.Add(15*time.Minute),
								ls.(*LastScore).Sum0,
								ls.(*LastScore).Tstart0,
								gschedule.Endtime,
							)
							ls.(*LastScore).Dup = -1
						}
						//	makePQ()
						log.Printf("%s id=%d !isonlive\n", eventid, id)
						//	if _, ok := scoremap[id]; !ok {
						if _, ok := scoremap.Load(id); !ok {
							log.Printf("%s scoremap[%d] not found.\n", eventid, id)
							return
						}

						//	RU20G1
						//	(*scoremap[id]).Qtime = (*scoremap[id]).Tstart0.Add(-time.Duration(gschedule.Modmin*60+gschedule.Modsec)*time.Second).Format("01/02 15:04") + "--" + timestamp.Add(-time.Duration(gschedule.Modmin*60+gschedule.Modsec)*time.Second-delay).Format("15:04")
						//	ststart0 := (*scoremap[id]).Tstart0.Format("01/02 15:04")
						//	stend := (*scoremap[id]).Tend.Format("15:04")
						//	ststart0 := (*scoremap[id]).Tstart0.Format("01/02 15:04")
						//	stend := (*scoremap[id]).Tend.Format("15:04")
						sc, _ := scoremap.Load(id)
						ls := sc.(*LastScore)
						ststart0 := ls.Tstart0.Format("01/02 15:04")
						stend := ls.Tend.Format("15:04")
						log.Printf("%s ststart0 = [%s] stend = [%s]\n", eventid, ststart0, stend)
						if ststart0 == "01/01 00:00" {
							ststart0 = ""
						}
						if stend == "00:00" {
							stend = ""
						}
						log.Printf("%s ststart0 = [%s] stend = [%s]\n", eventid, ststart0, stend)
						if ststart0 != "" || stend != "" {
							ls.Qtime = ststart0 + "--" + stend
						} else {
							ls.Qtime = ""
						}
						//	RU20G1	-----------------------------

						if ls.Continued > 0 {
							ls.Qtime += fmt.Sprintf("(C%d)", ls.Continued)
						} else if ls.Continued == -1 {
							ls.Qtime += "(E)"
						} else if ls.Continued < -1 {
							ls.Qtime += "(U)"
						}

					}

				}
				//	Ver. RU20G4	-----------------------------------------------------------------
				continue
			} else {
				if eventid != gschedule.Eventid {
					eventid = gschedule.Eventid
					//	rank = 0
				}

			}
			//	isonlive, startedat, status = GSE5Mlib.GetIsOnliveByAPI(client, IdList[i])
			//	if status != 0 {
			//		log.Printf("GetPointsAll() GetIsOnliveByAPI() err=[%d]\n", status)
			//		//	continue
			isonlive = false
			startedat = time.Now()
			//	}
			if p, ok := scoremap.Load(id); ok {
				if isonlive {
					p.(*LastScore).NoOffline = 0
				} else {
					p.(*LastScore).NoOffline++
				}
			} else {
				log.Printf("%s scoremap[%d] not found.\n", eventid, id)
			}
		}

		//	id, _ := strconv.Atoi(IdList[i])
		pstatus := "n/a"
		ptime := ""
		if p, ok := scoremap.Load(id); ok {
			if p.(*LastScore).Eventid != gschedule.Eventid {
				//	scoremap[]にあるイベントが取得対象のイベントと違う ＝  取得対象イベントでの初めてのデータ取得
				log.Printf("%s %s *Chg*%8d%7d %s\n", eventid, timestamp.Format("15:04:05"), point, id, eventid)
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

				//	scoremap[id] = &score
				scoremap.Store(id, &score)

			} else if p.(*LastScore).Score == point && p.(*LastScore).Rank == rank {
				//	獲得ポイントも順位も変化がないとき
				//	（順位の変化を獲得ポイントの変化と同一視するのは特定順位を目標とする場合があることを考慮しているため）

				if !isonlive {
					if p.(*LastScore).Dup == 1 && p.(*LastScore).NoOffline > 1 {
						//	同一の獲得ポイントが３回、オフラインが２回（以上）連続したとき
						if p.(*LastScore).Sum0 > 0 {
							p.(*LastScore).Qstatus = "+" + humanize.Comma(int64(p.(*LastScore).Sum0))
						} else if p.(*LastScore).Sum0 < 0 {
							p.(*LastScore).Qstatus = "-" + humanize.Comma(int64(-p.(*LastScore).Sum0))
						}
						if p.(*LastScore).Tend.After(timestamp) {
							p.(*LastScore).Tend = timestamp
						}

						//	makePQ = func() {

						log.Printf("%s id=%d !isonlive\n", eventid, id)
						if _, ok := scoremap.Load(id); !ok {
							log.Printf("%s scoremap[%d] not found.\n", eventid, id)
							//	return
							continue
						}

						//	RU20G1
						//	(*scoremap[id]).Qtime = (*scoremap[id]).Tstart0.Add(-time.Duration(gschedule.Modmin*60+gschedule.Modsec)*time.Second).Format("01/02 15:04") + "--" + timestamp.Add(-time.Duration(gschedule.Modmin*60+gschedule.Modsec)*time.Second-delay).Format("15:04")
						ststart0 := p.(*LastScore).Tstart0.Format("01/02 15:04")
						stend := p.(*LastScore).Tend.Format("15:04")
						log.Printf("%s ststart0 = [%s] stend = [%s]\n", eventid, ststart0, stend)
						if ststart0 == "01/01 00:00" {
							ststart0 = ""
						}
						if stend == "00:00" {
							stend = ""
						}
						log.Printf("%s ststart0 = [%s] stend = [%s]\n", eventid, ststart0, stend)
						if ststart0 != "" || stend != "" {
							p.(*LastScore).Qtime = ststart0 + "--" + stend
						} else {
							p.(*LastScore).Qtime = ""
						}
						//	RU20G1	-----------------------------

						if p.(*LastScore).Continued > 0 {
							p.(*LastScore).Qtime += fmt.Sprintf("(C%d)", p.(*LastScore).Continued)
						} else if p.(*LastScore).Continued == -1 {
							p.(*LastScore).Qtime += "(E)"
						} else if p.(*LastScore).Continued < -1 {
							p.(*LastScore).Qtime += "(U)"
						}
						//	}

						//	makePQ()

						p.(*LastScore).Continued = 0

						if cntrblist[i] == "Y" {
							//	イベント配信者設定で貢献ポイントランキングを取得すると設定されている場合
							if p.(*LastScore).Sum0 != 0 {
								//	配信がされていないときに順位が変わったケースは除く
								//	Ver. RU20G4	配信中にイベントが終了したら貢献ポイントを取得する（ことによって不要になった部分）
								//	InsertIntoTimeTable(eventid, id, timestamp, (*scoremap[id]).Sum0, (*scoremap[id]).Tstart0, (*scoremap[id]).Tend)
								InsertIntoTimeTable(eventid, id, timestamp.Add(5*time.Minute), p.(*LastScore).Sum0, p.(*LastScore).Tstart0, p.(*LastScore).Tend)
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
						log.Printf("%s p = [%s], [%s] q= [%s], [%s]\n", eventid, pstatus, ptime, p.(*LastScore).Qstatus, p.(*LastScore).Qtime)

						//	(*scoremap[id]).Dup += 1
						p.(*LastScore).Sum0 = 0
					}
					if p.(*LastScore).Dup != 0 {
						//	獲得ポイントが3回（以上）同じなのでまんなかのデータを削除する。
						//	これによってすべての配信者の獲得ポイントは最終取得時刻のものが存在する）
						DeleteFromPoints(tx, eventid, p.(*LastScore).ts, id)
						ptime = ""
						pstatus = "="
						//	log.Printf(" eventid=%s %s Dup=%d %8d%7d %s deleted.\n", eventid, timestamp.Format("15:04:05"), (*scoremap[id]).Dup, point, id, eventid)
					}
					p.(*LastScore).Dup += 1
					//	(*scoremap[id]).Sum0 = 0
				} else {
					//	配信中
					ptime = p.(*LastScore).Tstart0.Format("01/02 15:04:05")
					if startedat != p.(*LastScore).Tstart1 {
						//	配信が始まった
						p.(*LastScore).Tstart1 = startedat
						if p.(*LastScore).Sum0 != 0 {
							//	配信が更新された	Ver. RU20J0
							p.(*LastScore).Continued++
							ptime = p.(*LastScore).Tstart0.Format("01/02 15:04:05") + fmt.Sprintf("C%d", p.(*LastScore).Continued)
						}
					}
					if p.(*LastScore).Sum0 > 0 {
						pstatus = "+" + humanize.Comma(int64(p.(*LastScore).Sum0))
					} else if p.(*LastScore).Sum0 < 0 {
						pstatus = "-" + humanize.Comma(int64(-p.(*LastScore).Sum0))
					}
					p.(*LastScore).Dup = 0

				}
				p.(*LastScore).ts = timestamp
			} else {
				//	獲得ポイントか順位が変化した。
				pdelta := point - p.(*LastScore).Score

				if pdelta != 0 {
					//	獲得ポイントが変化した。
					if isonlive {
						//	配信中のとき
						if p.(*LastScore).Sum0 == 0 {
							//	最初の変化＝配信の開始であるとき
							p.(*LastScore).Tstart0 = startedat
							p.(*LastScore).Tend = startedat.Add(10000 * time.Hour)
							p.(*LastScore).Tstart1 = startedat
						} else {
							//	獲得ポイントの変化が続いているとき
							if p.(*LastScore).Tstart1 != startedat {
								//	更新が行われた。
								p.(*LastScore).Tstart1 = startedat
								p.(*LastScore).Continued++
							}
						}
					} else {
						if p.(*LastScore).Sum0 == 0 {
							//	減算が行われたあるいは短時間の配信が行われたと思われるとき（誤操作で配信をはじめ、すぐに配信をやめたようなケース）
							p.(*LastScore).Tstart0 = timestamp.Add(-time.Duration(gschedule.Modmin*60+gschedule.Modsec) * time.Second)
							p.(*LastScore).Tend = timestamp
							p.(*LastScore).Continued = -1
						} else {
							//	配信が終了した。
							p.(*LastScore).Tend = timestamp
						}
					}

					p.(*LastScore).Sum0 += pdelta
					ptime = p.(*LastScore).Tstart0.Format("01/02 15:04:05")
					if p.(*LastScore).Continued > 0 {
						ptime += fmt.Sprintf("(C%d)", p.(*LastScore).Continued)
					} else if p.(*LastScore).Continued == -1 {
						ptime += "(E)"
					} else if p.(*LastScore).Continued < -1 {
						ptime += "(U)"
					}

					if p.(*LastScore).Sum0 > 0 {
						pstatus = "+" + humanize.Comma(int64(p.(*LastScore).Sum0))
					} else if p.(*LastScore).Sum0 < 0 {
						pstatus = "-" + humanize.Comma(int64(-p.(*LastScore).Sum0))
					}
				} else {
					//	順位だけ変動した
					pstatus = "="
					ptime = ""

					//	RU20G6 順位だけ変化したときはQtimeが変化しないようにする。
					p.(*LastScore).Dup += 1

				}

				//	log.Printf("different data idx=%d, eventid=%s, user_id=%d point=%d\n", idx, eventid, id, point)
				log.Printf("%s %s Diff.%8d%7d %s %s\n", eventid, timestamp.Format("15:04:05"), point, id, ptime, pstatus)
				p.(*LastScore).Score = point
				p.(*LastScore).Rank = rank
				p.(*LastScore).ts = timestamp

				p.(*LastScore).Dup = 0
			}
		} else {
			//	ユーザの獲得ポイント履歴がない。新しく作ります。
			//	log.Printf("new data idx=%d, user_id=%d point=%d\n", idx, id, point)
			log.Printf("%s %s *New*%8d%7d %s\n", eventid, timestamp.Format("15:04:05"), point, id, eventid)
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

			scoremap.Store(id, &score)

			//	ptime = ""
			//	log.Printf("scoremap[%d]=%v\n", id, scoremap[id])
		}

		log.Printf("%s id=%d point=%d rank=%d\n", eventid, id, point, rank)
		p, _ := scoremap.Load(id)
		ls := p.(*LastScore)
		InsertIntoPoints(tx, timestamp, id, point, rank, gap, eventid, pstatus, ptime, p.(*LastScore).Qstatus, ls.Qtime)

	}

	tx.Commit()

	//	SaveScoremap()

	//	if runtime.GOOS == "windows" {
	MakeComment()
	//	}

	return
}
