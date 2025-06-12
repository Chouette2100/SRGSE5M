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
	"SRGSE5M/GSE5Mlib"
	//	"SRGSE5M/SRDBlib"

	"github.com/dustin/go-humanize"

	"github.com/Chouette2100/exsrapi/v2"
	"github.com/Chouette2100/srapi/v2"
	"github.com/Chouette2100/srdblib/v2"
)

/*
	各配信者さんの獲得ポイントのリストを作る（ファイルに追記する）
	ファイルは獲得ポイントを横並びにしたものと、各配信者さんの順位、獲得ポイント、
*/

func ScanActive(client *http.Client, gschedule Gschedule) (status int) {

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic:", r)
		}
	}()

	var err error

	var stmt *sql.Stmt
	var rows *sql.Rows

	cmt0 := gschedule.Eventid
	fncname := exsrapi.FuncNameOfThisFunction(1) + "()"
	//	fncname := "ScanActive()"
	log.Println(cmt0, ">>>>>>>>>>>>>>>>>>", fncname, ">>>>>>>>>>>>>>>>>>>")
	defer exsrapi.PrintExf(cmt0, fncname)()

	sqlstmt := "select userno, iscntrbpoints from eventuser where eventid = ? and istarget ='Y'"
	stmt, err = srdblib.Db.Prepare(sqlstmt)
	if err != nil {
		log.Printf("ScanActive() Prepare() err=%s\n", err.Error())
		status = -5
		return
	}
	defer stmt.Close()

	rows, err = stmt.Query(gschedule.Eventid)
	if err != nil {
		log.Printf("ScanActive() Query() (6) err=%s\n", err.Error())
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

	if err = rows.Err(); err != nil {
		log.Printf("ScanActive() rows err=%s\n", err.Error())
		status = -8
		return
	}

	//	log.Println("ScanActive() idlist=", idlist)
	//	if len(idlist) != 0 {
	//	status = GetPointsAll(idlist, gschedule, cntrblist)
	GetPointsAll(client, idlist, gschedule, cntrblist)
	//	}

	return

}

/*
配信者のリストからそれぞれの獲得ポイントなどを取得する。
*/
func GetPointsAll(client *http.Client, idList []string, gschedule Gschedule, cntrblist []string) (status int) {

	var err error
	status = 0

	eventid := gschedule.Eventid

	timestamp := time.Now().Truncate(time.Second)
	//	if timestamp.After(gschedule.Endtime.Add(time.Duration(gschedule.Intervalmin+1) * time.Minute)) {
	if timestamp.After(gschedule.Endtime.Add(1 * time.Minute)) {
		//	イベントが終了した
		log.Printf("%s set rstatus = DontGetScore.\n", eventid)
		sqlstmte := "update event set rstatus = ? where eventid = ?"
		_, err = srdblib.Db.Exec(sqlstmte, "DontGetScore", gschedule.Eventid)

		if err != nil {
			log.Printf("%s update event err=[%s]\n", eventid, err.Error())
		}
	}

	hh := time.Since(gschedule.Starttime).Hours()
	//	thpoint := gschedule.Thdelta*(int(hh)) + gschedule.Thinit
	thpoint := gschedule.Thdelta * (int(hh))
	if thpoint < gschedule.Thinit {
		thpoint = gschedule.Thinit
	}
	log.Printf("%s Starttime=%s Hours=%7.2f\n", eventid, gschedule.Starttime.Format("2006-01-02 15:04:05"), hh)
	log.Printf("%s hh=%d Thinit=%d Thdelta=%d thpoint=%d\n",
		eventid, int(hh), gschedule.Thinit, gschedule.Thdelta, thpoint)

	//	指定した順位の範囲のルームがidListに存在するかチェックするためidListのmapを作っておく
	//		idListはこの時点でeventuserに存在するルームのuserno（をstringで表現したもの）
	umap := make(map[int]bool)
	//	eventuserにあるルームのみで作ったmap
	umap_eu := make(map[int]bool)
	for _, id := range idList {
		userno, _ := strconv.Atoi(id)
		umap[userno] = false
		umap_eu[userno] = false
	}

	var pranking *srapi.Eventranking
	// var err error

	//	50位までのルームの順位、ポイントを取得する(ランキングイベントに限る)
	pranking, err = srdblib.GetEventsRankingByApi(client, gschedule.Eventid, 1)
	if err != nil {
		log.Printf("%s GetEventsRankingByApi() err=[%s]\n", eventid, err.Error())
		// 	return -1
	}
	lpr := 0
	if err == nil {
		lpr = len(pranking.Ranking)
	}
	log.Printf("%s GetEventsRankingByApi() =%d\n", eventid, lpr)
	//	if lpr > 0 &&  lpr < gschedule.Toorder - 5 {
	//		thpoint =  0
	//	} else if lpr > gschedule.Toorder - 6 {
	//			 hh := time.Since(gschedule.Starttime).Hours()
	//			 thpoint =  400 * ( int(hh) + 4)
	//			 log.Printf("%s Starttime=%s Hours=%7.2f\n", eventid, gschedule.Starttime.Format("2006-01-02 15:04:05"), hh)
	//			 log.Printf("%s lpr=%d hh=%d thpoint=%d\n", eventid, lpr, int(hh), thpoint)
	//	}

	//	noranking := false
	// if len(pranking.Ranking) == 0 {
	if lpr == 0 {
		//	1. レベルイベント（GetEventsRankingByApi()で獲得ポイントを取得できない）
		//	2. イベント開始前
		//	3. イベントエントリーなし
		//	noranking = true
		/*
			roomlistinf, err := srapi.GetRoominfFromEventByApi(
				client,
				gschedule.Ieventid, //	Event_id (int) event_url_key ではないことに注意
				gschedule.Fromorder,
				gschedule.Toorder,
			)
			if err != nil {
				err = fmt.Errorf("srapi.GetRoominfFromEventByApi() returned error. %w", err)
				log.Printf("%s srapi.GetRoominfFromEventByApi() err=[%s]\n", eventid, err.Error())
			}

			//	lrr := len(roomlistinf.RoomList)
			//	log.Printf("%s srapi.GetRoominfFromEventByApi() =%d\n", eventid, lrr)
			//	if lrr >  gschedule.Toorder -6 {
			//		 hh := time.Since(gschedule.Starttime).Hours()
			//		 thpoint =  400 * ( int(hh) + 4)
			//		 log.Printf("%s Starttime=%s Hours=%7.2f\n", eventid, gschedule.Starttime.Format("2006-01-02 15:04:05"), hh)
			//		 log.Printf("%s lrr=%d hh=%d thpoint=%d\n", eventid, lrr, int(hh), thpoint)
			//	}

			for _, room := range roomlistinf.RoomList {
				//	if room.Rank < 2 {
				//		//TODO: 除外の条件が厳しすぎる？
				//		break
				//	}
				userno := room.Room_id
				if _, ok := umap[userno]; !ok {
					//	srdblib.UpinsEventuser(client, -1, 0, gschedule.Eventid, gschedule.Starttime, userno, timestamp)
					idList = append(idList, strconv.Itoa(userno))
					cntrblist = append(cntrblist, "N")
				}
				//	cntrblist := append(cntrblist, "N")
				//	//		GetPointsAll(idlist, gschedule, cntrblist)
				//	GetPointsAll(client, idlist, gschedule, cntrblist)
				//	time.Sleep(time.Duration(gschedule.Intervalmin+1) * time.Minute)
			}
		*/
		var erl *srapi.EventRanking
		erl, err = srapi.GetEventRankingByApi(client, gschedule.Eventid, gschedule.Fromorder, gschedule.Toorder)
		if err != nil {
			err = fmt.Errorf("srapi.GetEventRankingByApi() returned error. %w", err)
			log.Printf("%s err=[%s]\n", eventid, err.Error())
		}
		if len(erl.Ranking) != 0 {
			for _, room := range erl.Ranking {
				userno := room.RoomID
				if _, ok := umap[userno]; !ok {
					//	srdblib.UpinsEventuser(client, -1, 0, gschedule.Eventid, gschedule.Starttime, userno, timestamp)
					idList = append(idList, strconv.Itoa(userno))
					cntrblist = append(cntrblist, "N")
				}
			}
		} else {
			// レベルイベントのときは、GetEventQuestRooms()を使う
			var eqr *srapi.EventQuestRooms
			eqr, err = srapi.GetEventQuestRoomsByApi(client, gschedule.Eventid, gschedule.Fromorder, gschedule.Toorder)
			if err != nil {
				err = fmt.Errorf("srapi.GetEventQuestRooms() returned error. %w", err)
				log.Printf("%s err=[%s]\n", eventid, err.Error())
				return
			}
			for _, room := range eqr.EventQuestLevelRanges[0].Rooms {
				userno := room.RoomID
				if _, ok := umap[userno]; !ok {
					//	srdblib.UpinsEventuser(client, -1, 0, gschedule.Eventid, gschedule.Starttime, userno, timestamp)
					idList = append(idList, strconv.Itoa(userno))
					cntrblist = append(cntrblist, "N")
				}
			}

		}
	}

	//	block_id ==0 のブロックイベントのときは100位までの順位を取得する
	//	block_id ==0 のときは51位より下位のルームの順位はこの方法でないとわからない
	//	またusernoから順位を取得できるようにmapを作っておく
	//	OPTIMIZE: ここの処理はblock_id=0のときしか必要ないはず？
	eida := strings.Split(gschedule.Eventid, "?block_id=")
	qmap := make(map[int]int)
	blockid := -1
	qlist := new([]srapi.Block_ranking)
	if len(eida) == 2 {
		blockid, _ = strconv.Atoi(eida[1])
		qranking, err := srapi.GetEventBlockRanking(client, gschedule.Ieventid, blockid, 1, 100)
		if err != nil {
			log.Printf("%s GetEventBlockRanking() err=[%s]\n", eventid, err.Error())
		} else {
			qlist = &qranking.Block_ranking_list
			for i, ranking := range qranking.Block_ranking_list {
				quno, _ := strconv.Atoi(ranking.Room_id)
				qmap[quno] = i
			}
		}
	}

	plist := make([]srdblib.Points, 0, 50)
	pmap := make(map[int]int)

	//	if noranking {
	//		//	prankingが作られていない
	//	指定した範囲の順位にあるルームのid(userno)を保存する。
	for i := gschedule.Fromorder - 1; i < gschedule.Toorder; i++ {
		// if i >= len(pranking.Ranking) {
		if i >= lpr {
			break
		}
		ranking := pranking.Ranking[i]
		//	if ranking.Point == 0 {
		//		//	pranking.Rankingはポイント順に逆ソートされているので獲得ポイントが0のデータがあれば以下は処理の対象としない
		//		break
		//	}
		userno := ranking.Room.RoomID
		if _, ok := umap[userno]; !ok {
			//	eventuserには存在しないルーム
			//	log.Printf("%s id=%6d is not in idList\n", eventid, userno)
			//	srdblib.UpinsEventuser(client, ranking.Rank, ranking.Point, gschedule.Eventid, gschedule.Starttime, userno, timestamp)
			idList = append(idList, strconv.Itoa(userno))
			cntrblist = append(cntrblist, "N")

		}
	}
	//	}

	//	log.Printf("%s GetPointsAll() GetPointsAll() =%+v\n", eventid, idList)

	if len(idList) == 0 {
		//	取得対象が存在しない
		return
	}

	if lpr != 0 {
		for i, ranking := range pranking.Ranking {
			pmap[ranking.Room.RoomID] = i
			plist = append(plist, srdblib.Points{
				Eventid: gschedule.Eventid,
				User_id: ranking.Room.RoomID,
				Point:   ranking.Point,
				Rank:    ranking.Rank,
			})
		}
	}

	for _, userid := range idList {
		userno, _ := strconv.Atoi(userid)
		if _, ok := pmap[userno]; ok {
			continue
		} else {
			//	log.Printf("%s id=%6d is not in pranking\n", eventid, userno)
			//	eventuserには存在するが上位50位のデータには存在しないルーム
			//	point, rank, gap, eventid := GSE5Mlib.GetPointsByAPI(userid)
			point, rank, gap, _, teventid, _, _, err := srapi.GetPointByApi(client, userno)
			if err != nil {
				log.Printf("%s id=%6d GetPointByApi() err=[%s]\n", eventid, userno, err.Error())
				continue
			}
			if teventid == "" {
				log.Printf("%s id=%6d GetPointByApi() not found in event.", eventid, userno)
				continue
			}
			/* =====================================
			if teventid != eida[0] || (len(eida) == 2 && blockid != 0 && bid != blockid) {
				log.Printf("%s id=%6d GetPointByApi() teventid=%s bid=%d\n", eventid, userno, teventid, bid)
				//	イベントを変更した等、このイベントにはエントリーしていないルーム
				//	GetPointsByAPI()で取得するeventidにはblock_idは入っていない
				continue
			}
			==================================== */
			//	if point == 0 {
			//		//	獲得ポイントが0だから除外する（ランキングイベントはすでにチェックずみ、レベルイベントのためにある）
			//		continue
			//	}
			pmap[userno] = len(plist)
			plist = append(plist, srdblib.Points{
				Eventid: teventid,
				User_id: userno,
				Point:   point,
				Rank:    rank,
				Gap:     gap,
			})
			//	if noranking {
			//		//	レベルイベントの場合はeventuserにも追加する
			//		srdblib.UpinsEventuser(client, rank, point, eventid, gschedule.Starttime, userno, timestamp)
			//	}

		}

	}

	// =============================================================================================================

	//	pstatus := "n/a"
	//	ptime := ""
	//	log.Printf("%s %+v\n", eventid, idList)

	//	[]idList: eventuserに存在するルームに取得対象（の候補）となるルームを加えたもののルームIDのリスト
	//	lenth: len(idList)
	//	[]plist: 獲得ポイントデータ
	//	pmap: ルームid/ユーザーNoから獲得ポイント配列（plist）のインデックスを求めるためのmap
	//	[]qlist: block_id == 0 のブロックデータの（100位までの）ルームのイベント順位
	//	qmap: ルームID/ユーザーNoから順位配列（qlist）のインデックスを求めるためのmap

	if timestamp.After(gschedule.Endtime.Add(1 * time.Minute)) {
		//	イベントが終了した
		log.Printf("%s event ended.\n", eventid)
		for i, id := range idList {
			uno, _ := strconv.Atoi(id)
			eventid := gschedule.Eventid
			log.Printf("%s event ended.\n", eventid)

			dup := -9
			unoeid := fmt.Sprintf("%d#%s", uno, eventid)
			if _, ok := scoremap.Load(unoeid); ok {
				ls, _ := scoremap.Load(unoeid)
				dup = ls.(*LastScore).Dup
			}
			//	log.Printf("%s timestamp=%v gschedule.Endtime=%v scoremap[unoeid].Dup=%d\n", eventid, timestamp, gschedule.Endtime, dup)
			//	if scoremap[id].Dup == 0 {	// 該当scoremap[id]が存在しない場合異常終了する。
			if dup == 0 {
				//	配信中のイベント終了
				if cntrblist[i] == "Y" {
					//	イベント配信者設定で貢献ポイントランキングを取得すると設定されている場合
					ls, _ := scoremap.Load(unoeid)
					InsertIntoTimeTable(
						gschedule.Eventid, uno,
						timestamp.Add(15*time.Minute),
						ls.(*LastScore).Sum0,
						ls.(*LastScore).Tstart0,
						gschedule.Endtime,
					)
				}
			}
		}
		return
	}

	type newuser struct {
		userno int
		rank   int
		point  int
	}
	nu := make([]newuser, 0, 50)

	var tx *sql.Tx
	tx, err = srdblib.Db.Begin()
	if err != nil {
		log.Printf("%s srdblib.Db.Begin() err=[%s]\n", eventid, err.Error())
		return -1
	}
	defer tx.Rollback()

	for i, id := range idList {
		//	for i, p := range(plist) {

		//	var makePQ func()
		uno, _ := strconv.Atoi(id)
		//	log.Printf("%s id=%6d\n", eventid, uno)
		//	id := p.User_id

		//	開催されていないイベントに対する設定を兼ねる変数定義
		eventid := gschedule.Eventid

		var isonlive bool
		var startedat time.Time

		point := 0
		rank := 0
		gap := 0

		if idx, ok := pmap[uno]; ok {
			p := plist[idx]
			point = p.Point
			rank = p.Rank
			gap = p.Gap
			eventid = p.Eventid
			if blockid == 0 {
				//	ブロックIDが0の場合はGetPointByApi()で取得した順位がエントリーしたブロックでの順位であることに注意
				//	OPTIMIZE: 以下の処理はブロックIDがが0で順位が50位より下のケース、工夫が足りない？
				if idxq, ok := qmap[uno]; ok {
					//	ルームがbloc_id=0の100位以内リストにある
					//	log.Printf("%s rank=%d, qrank=%d\n", eventid, rank, (*qlist)[idxq].Rank)
					rank = (*qlist)[idxq].Rank
				} else {
					//	block_id=0の100位以内リストにルームがない
					rank = 9999
				}
			}
		} else {
			log.Printf("%s id=%6d id not found in pmap\n", eventid, uno)
			continue
		}
		//	if !strings.Contains(eventid, gschedule.Eventid) {
		if !strings.Contains(gschedule.Eventid, eventid) {
			//	イベントがデータ取得対象のイベントではない
			log.Printf("%s id=%6d isn't gschedule.Eventid(%s)\n", eventid, uno, gschedule.Eventid)
			continue
		} else {
			if eventid != gschedule.Eventid {
				//	ブロックイベントの場合 eventid には "?block_id=...."" の部分を含まない
				eventid = gschedule.Eventid
				//	rank = 0
			}

		}
		/*   =============================================
		//	isonlive, startedat, status = GSE5Mlib.GetIsOnliveByAPI(client, idList[i])
		//	if status != 0 {
		//		log.Printf("%s GetPointsAll() GetIsOnliveByAPI() err=[%d]\n", eventid, status)
		//		//	continue
		isonlive = false
		startedat = time.Now()
		//	}
		/* ==================================================== */
		if cntrblist[i] == "Y" {
			isonlive, startedat, status = GSE5Mlib.GetIsOnliveByAPI(client, id)
			if status != 0 {
				log.Printf("%s GetPointsAll() GetIsOnliveByAPI() err=[%d]\n", eventid, status)
			}
		}
		/* ==================================================== */
		unoeid := strconv.Itoa(uno) + "#" + eventid
		if p, ok := scoremap.Load(unoeid); ok {
			if isonlive {
				p.(*LastScore).NoOffline = 0
			} else {
				p.(*LastScore).NoOffline++
			}
		}
		//	} else {
		//		log.Printf("%s scoremap[%s] not found.\n", eventid, unoeid)
		//	}

		pstatus := "n/a"
		ptime := ""
		//	unoeid := strconv.Itoa(uno) + "#" + eventid
		if p, ok := scoremap.Load(unoeid); ok && p.(*LastScore).Eventid == gschedule.Eventid {
			// A 履歴が存在する
			if p.(*LastScore).Score == point && p.(*LastScore).Rank == rank {
				//	A-1 獲得ポイントも順位も変化がないとき
				//	（順位の変化を獲得ポイントの変化と同一視するのは特定順位を目標とする場合があることを考慮しているため）

				if !isonlive {
					// A-1-1
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

						log.Printf("%s id=%6d !isonlive\n", eventid, uno)
						unoeid := strconv.Itoa(uno) + "#" + eventid
						if _, ok := scoremap.Load(unoeid); !ok {
							log.Printf("%s scoremap[%s] not found.\n", eventid, unoeid)
							//	return
							continue
						}

						//	RU20G1
						//	(*scoremap[id]).Qtime = (*scoremap[id]).Tstart0.Add(-time.Duration(gschedule.Modmin*60+gschedule.Modsec)*time.Second).Format("01/02 15:04") + "--" + timestamp.Add(-time.Duration(gschedule.Modmin*60+gschedule.Modsec)*time.Second-delay).Format("15:04")
						ststart0 := p.(*LastScore).Tstart0.Format("01/02 15:04")
						stend := p.(*LastScore).Tend.Format("15:04")
						log.Printf("%s id=%6d ststart0 = [%s] stend = [%s]\n", eventid, uno, ststart0, stend)
						if ststart0 == "01/01 00:00" {
							ststart0 = ""
						} else {
							if stend == "00:00" {
								p.(*LastScore).Tend = timestamp.Add(-10 * time.Minute)
								stend = p.(*LastScore).Tend.Format("15:04")
							}
						}
						if stend == "00:00" {
							stend = ""
						}
						log.Printf("%s id=%6d ststart0 = [%s] stend = [%s]\n", eventid, uno, ststart0, stend)
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
								//	InsertIntoTimeTable(eventid, uno, timestamp, (*scoremap[id]).Sum0, (*scoremap[id]).Tstart0, (*scoremap[id]).Tend)
								InsertIntoTimeTable(eventid, uno, timestamp.Add(5*time.Minute), p.(*LastScore).Sum0,
									p.(*LastScore).Tstart0, p.(*LastScore).Tend)
								/*
									if time.Until(gschedule.Endtime) > 5 * time.Minute {
									} else {
										//	配信終了直前では獲得ポイントの更新は行われなくなるが貢献ポイントは更新されるはず (RU20G3)
										log.Printf("%s time.Until(gschedule.Endtime) < 5 * time.Minute\n", %s)
										InsertIntoTimeTable(eventid, uno, timestamp.Add(15 * time.Minute), (*scoremap[uno]).Sum0, (*scoremap[uno]).Tstart0, (*scoremap[uno]).Tend)
									}
								*/
								//	Ver. RU20G4	-----------------------------------------------------

							}
						}

						ptime = ""
						pstatus = "="
						log.Printf("%s id=%6d p = [%s], [%s] q= [%s], [%s]\n", eventid, uno, pstatus, ptime, p.(*LastScore).Qstatus, p.(*LastScore).Qtime)

						//	(*scoremap[uno]).Dup += 1
						p.(*LastScore).Sum0 = 0
					}
					if p.(*LastScore).Dup != 0 {
						//	獲得ポイントが3回（以上）同じなのでまんなかのデータを削除する。
						//	これによってすべての配信者の獲得ポイントは最終取得時刻のものが存在する）
						DeleteFromPoints(tx, eventid, p.(*LastScore).ts, uno)
						ptime = ""
						pstatus = "="
						//	log.Printf("%s %s Dup=%d %8d%7d %s deleted.\n", eventid, timestamp.Format("15:04:05"), (*scoremap[uno]).Dup, point, uno, eventid)
					}
					p.(*LastScore).Dup += 1
					//	(*scoremap[uno]).Sum0 = 0
				} else {
					//	A-1-2 配信中
					ptime = p.(*LastScore).Tstart0.Format("01/02 15:04:05")
					if startedat != p.(*LastScore).Tstart1 {
						//	配信が始まった
						p.(*LastScore).Tstart1 = startedat
						p.(*LastScore).Tend = startedat.Add(10000 * time.Hour)
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
				//	A-2 獲得ポイントか順位が変化した。
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

				//	log.Printf("%s different data idx=%d, eventid=%s, user_id=%6d point=%d\n", eventid, idx, eventid, uno, point)
				log.Printf("%s id=%6d Diff. %s %s\n", eventid, uno, ptime, pstatus)
				p.(*LastScore).Score = point
				p.(*LastScore).Rank = rank
				p.(*LastScore).ts = timestamp

				p.(*LastScore).Dup = 0
			}
		} else {
			//	B ユーザの獲得ポイント履歴がない。新しく作ります。
			_, ok := umap_eu[uno]
			if !ok && point < thpoint {
				//	eventuserにないルームのpointが0のときはpointを保存しない
				continue
			}

			//	log.Printf("%s new data idx=%d, user_id=%6d point=%d\n", eventid, idx, uno, point)
			log.Printf("%s id=%6d %s *New*%8d\n", eventid, uno, timestamp.Format("15:04:05"), point)
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

			unoeid := strconv.Itoa(uno) + "#" + eventid
			scoremap.Store(unoeid, &score)

			//	ptime = ""
			//	log.Printf("%s scoremap[%d]=%v\n", eventid, uno, scoremap[uno])
		}

		log.Printf("%s id=%6d point=%d rank=%d\n", eventid, uno, point, rank)
		p, _ := scoremap.Load(unoeid)
		ls := p.(*LastScore)
		InsertIntoPoints(tx, timestamp, uno, point, rank, gap, eventid, pstatus, ptime, p.(*LastScore).Qstatus, ls.Qtime)
		if _, ok := umap_eu[uno]; !ok {
			nu = append(nu, newuser{userno: uno, rank: rank, point: point})
		}

	}

	tx.Commit()

	//	順位と獲得ポイントをeventuserに格納する（eventuserはその時点での最終結果を保存している）
	for _, v := range nu {
		id := v.userno
		point := v.point
		rank := v.rank
		itfc, err := srdblib.Dbmap.Get(srdblib.Eventuser{}, eventid, id)
		if err != nil {
			log.Printf("%s id=%6d Dbmap.Get(Eventuser{},...) err=[%v]\n", eventid, id, err)
			continue
		}
		if itfc == nil {
			err := srdblib.UpinsEventuser(client, rank, point, eventid, gschedule.Starttime, gschedule.Cmap, id, timestamp)
			if err != nil {
				log.Printf("%s id=%6d UpinsEventuser() err=[%v]\n", eventid, id, err)
			} else {
				log.Printf("%s id=%6d UpinsEventuser() ok\n", eventid, id)
			}
		}
	}
	//	SaveScoremap()

	//	MakeComment()	MakeComment()はscoremapのキーがeventid+usernoとなったバージョンに対応していないので現時点では使えない。

	return
}
