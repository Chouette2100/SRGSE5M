// Copyright © 2022-2024 chouette.21.00@gmail.com
// Released under the MIT license
// https://opensource.org/licenses/mit-license.php
package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"net/http"

	"database/sql"

	_ "github.com/go-sql-driver/mysql"

	"SRGSE5M/GSE5Mlib"

	"github.com/dustin/go-humanize"

	"github.com/Chouette2100/exsrapi/v2"
	"github.com/Chouette2100/srapi/v2"
	"github.com/Chouette2100/srdblib/v3"
)

type TrackedRoom struct {
	ID             string
	IsContribution bool
}

type LiveWindow struct {
	IsLive    bool
	StartedAt time.Time
	IsKnown   bool
}

type PointCollection struct {
	IDList      []string
	CntrbList   []string
	UmapEU      map[int]bool
	PList       []srdblib.Points
	Pmap        map[int]int
	QList       []srapi.Block_ranking
	Qmap        map[int]int
	BlockID     int
	EventIsOver bool
}

type TransitionResult struct {
	EventID        string
	UserNo         int
	ScoreKey       string
	Point          int
	Rank           int
	Gap            int
	PStatus        string
	PTime          string
	DeletePointTS  *time.Time
	TimeTable      *TimeTableCandidate
	ShouldPersist  bool
	NeedsEventUser bool
}

type TimeTableCandidate struct {
	EventID     string
	UserNo      int
	SampleTime  time.Time
	EarnedPoint int
	StartTime   time.Time
	EndTime     time.Time
}

type RoomContext struct {
	ID             string
	UserNo         int
	EventID        string
	Point          int
	Rank           int
	Gap            int
	IsOnLive       bool
	StartedAt      time.Time
	IsContribution bool
	KnownEventUser bool
}

type EventUserCandidate struct {
	UserNo int
	Rank   int
	Point  int
}

func trackedRoomsFromLists(idList []string, cntrblist []string) []TrackedRoom {
	rooms := make([]TrackedRoom, 0, len(idList))
	for i, id := range idList {
		isContribution := i < len(cntrblist) && cntrblist[i] == "Y"
		rooms = append(rooms, TrackedRoom{ID: id, IsContribution: isContribution})
	}
	return rooms
}

func makeRoomKey(userno int, eventid string) string {
	return fmt.Sprintf("%d#%s", userno, eventid)
}

func computeThresholdPoint(gschedule Gschedule) int {
	hh := time.Since(gschedule.Starttime).Hours()
	thpoint := gschedule.Thdelta * int(hh)
	if thpoint < gschedule.Thinit {
		thpoint = gschedule.Thinit
	}
	log.Printf("%s Starttime=%s Hours=%7.2f\n", gschedule.Eventid, gschedule.Starttime.Format("2006-01-02 15:04:05"), hh)
	log.Printf("%s hh=%d Thinit=%d Thdelta=%d thpoint=%d\n",
		gschedule.Eventid, int(hh), gschedule.Thinit, gschedule.Thdelta, thpoint)
	return thpoint
}

func setEventDontGetScore(eventid string) error {
	sqlstmt := "update event set rstatus = ? where eventid = ?"
	_, err := GSE5Mlib.Db.Exec(sqlstmt, "DontGetScore", eventid)
	return err
}

func resolveLiveWindow(client *http.Client, roomID string, useContribution bool) (LiveWindow, int) {
	if !useContribution {
		return LiveWindow{}, 0
	}

	isonlive, startedat, status := GSE5Mlib.GetIsOnliveByAPI(client, roomID)
	return LiveWindow{
		IsLive:    isonlive,
		StartedAt: startedat,
		IsKnown:   true,
	}, status
}

func handleEventEnd(gschedule Gschedule, trackedRooms []TrackedRoom, timestamp time.Time) {
	eventid := gschedule.Eventid
	log.Printf("%s event ended.\n", eventid)
	for _, room := range trackedRooms {
		uno, _ := strconv.Atoi(room.ID)
		log.Printf("%s event ended.\n", eventid)

		dup := -9
		unoeid := makeRoomKey(uno, eventid)
		if ls, ok := scoremap.Load(unoeid); ok {
			dup = ls.(*LastScore).Dup
		}
		if dup == 0 && room.IsContribution {
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

func collectPointSnapshots(client *http.Client, gschedule Gschedule, timestamp time.Time, idList []string, cntrblist []string) (PointCollection, error) {
	eventid := gschedule.Eventid
	collection := PointCollection{
		IDList:    append([]string(nil), idList...),
		CntrbList: append([]string(nil), cntrblist...),
		UmapEU:    make(map[int]bool),
		PList:     make([]srdblib.Points, 0, 50),
		Pmap:      make(map[int]int),
		QList:     make([]srapi.Block_ranking, 0),
		Qmap:      make(map[int]int),
		BlockID:   -1,
	}

	umap := make(map[int]bool)
	for _, id := range collection.IDList {
		userno, _ := strconv.Atoi(id)
		umap[userno] = false
		collection.UmapEU[userno] = false
	}

	pranking, err := srdblib.GetEventsRankingByApi(GSE5Mlib.Dbmap, client, gschedule.Eventid, 1)
	if err != nil {
		log.Printf("%s GetEventsRankingByApi() err=[%s]\n", eventid, err.Error())
	}
	lpr := 0
	if err == nil {
		lpr = len(pranking.Ranking)
	}
	log.Printf("%s GetEventsRankingByApi() =%d\n", eventid, lpr)

	if lpr == 0 {
		var erl *srapi.EventRanking
		erl, err = srapi.GetEventRankingByApi(client, gschedule.Eventid, gschedule.Fromorder, gschedule.Toorder)
		if err != nil {
			err = fmt.Errorf("srapi.GetEventRankingByApi() returned error. %w", err)
			log.Printf("%s err=[%s]\n", eventid, err.Error())
		}
		if erl != nil && len(erl.Ranking) != 0 {
			for _, room := range erl.Ranking {
				userno := room.RoomID
				if _, ok := umap[userno]; !ok {
					collection.IDList = append(collection.IDList, strconv.Itoa(userno))
					collection.CntrbList = append(collection.CntrbList, "N")
				}
			}
		} else {
			var eqr *srapi.EventQuestRooms
			eqr, err = srapi.GetEventQuestRoomsByApi(client, gschedule.Eventid, gschedule.Fromorder, gschedule.Toorder)
			if err != nil {
				if timestamp.Before(gschedule.Endtime.Add(1 * time.Minute)) {
					err = fmt.Errorf("srapi.GetEventQuestRooms() returned error. %w", err)
					log.Printf("%s err=[%s]\n", eventid, err.Error())
					return collection, err
				}
				collection.EventIsOver = true
			} else {
				for _, room := range eqr.EventQuestLevelRanges[0].Rooms {
					userno := room.RoomID
					if _, ok := umap[userno]; !ok {
						collection.IDList = append(collection.IDList, strconv.Itoa(userno))
						collection.CntrbList = append(collection.CntrbList, "N")
					}
				}
			}
		}
	}

	eida := strings.Split(gschedule.Eventid, "?block_id=")
	if !collection.EventIsOver {
		if len(eida) == 2 {
			collection.BlockID, _ = strconv.Atoi(eida[1])
			qranking, qerr := srapi.GetEventBlockRanking(client, gschedule.Ieventid, collection.BlockID, 1, 100)
			if qerr != nil {
				log.Printf("%s GetEventBlockRanking() err=[%s]\n", eventid, qerr.Error())
			} else {
				collection.QList = qranking.Block_ranking_list
				for i, ranking := range qranking.Block_ranking_list {
					quno, _ := strconv.Atoi(ranking.Room_id)
					collection.Qmap[quno] = i
				}
			}
		}

		for i := gschedule.Fromorder - 1; i < gschedule.Toorder; i++ {
			if i >= lpr {
				break
			}
			ranking := pranking.Ranking[i]
			userno := ranking.Room.RoomID
			if _, ok := umap[userno]; !ok {
				collection.IDList = append(collection.IDList, strconv.Itoa(userno))
				collection.CntrbList = append(collection.CntrbList, "N")
			}
		}

		if len(collection.IDList) == 0 {
			return collection, nil
		}

		if lpr != 0 {
			for i, ranking := range pranking.Ranking {
				collection.Pmap[ranking.Room.RoomID] = i
				collection.PList = append(collection.PList, srdblib.Points{
					Eventid: gschedule.Eventid,
					User_id: ranking.Room.RoomID,
					Point:   ranking.Point,
					Rank:    ranking.Rank,
				})
			}
		}

		for _, userid := range collection.IDList {
			userno, _ := strconv.Atoi(userid)
			if _, ok := collection.Pmap[userno]; ok {
				continue
			}

			point, rank, gap, _, teventid, _, _, gerr := srapi.GetPointByApi(client, userno)
			if gerr != nil {
				log.Printf("%s id=%6d GetPointByApi() err=[%s]\n", eventid, userno, gerr.Error())
				continue
			}
			if teventid == "" {
				log.Printf("%s id=%6d GetPointByApi() not found in event.", eventid, userno)
				continue
			}

			collection.Pmap[userno] = len(collection.PList)
			collection.PList = append(collection.PList, srdblib.Points{
				Eventid: teventid,
				User_id: userno,
				Point:   point,
				Rank:    rank,
				Gap:     gap,
			})
		}
	}

	return collection, nil
}

func buildRoomContext(
	client *http.Client,
	gschedule Gschedule,
	collection PointCollection,
	id string,
	isContribution bool,
) (RoomContext, bool) {
	ctx := RoomContext{
		ID:             id,
		EventID:        gschedule.Eventid,
		IsContribution: isContribution,
	}

	uno, err := strconv.Atoi(id)
	if err != nil {
		log.Printf("%s id=%s invalid user id err=[%s]\n", gschedule.Eventid, id, err.Error())
		return ctx, false
	}
	ctx.UserNo = uno
	ctx.KnownEventUser = collection.UmapEU[uno]

	idx, ok := collection.Pmap[uno]
	if !ok {
		log.Printf("%s id=%6d id not found in pmap\n", gschedule.Eventid, uno)
		return ctx, false
	}

	p := collection.PList[idx]
	ctx.Point = p.Point
	ctx.Rank = p.Rank
	ctx.Gap = p.Gap
	ctx.EventID = p.Eventid

	if collection.BlockID == 0 {
		if idxq, ok := collection.Qmap[uno]; ok {
			if ctx.Rank == 0 {
				ctx.Rank = collection.QList[idxq].Rank
			}
		} else {
			ctx.Rank = 9999
		}
	}

	if !strings.Contains(gschedule.Eventid, ctx.EventID) {
		log.Printf("%s id=%6d isn't gschedule.Eventid(%s)\n", ctx.EventID, uno, gschedule.Eventid)
		return ctx, false
	}
	if ctx.EventID != gschedule.Eventid {
		ctx.EventID = gschedule.Eventid
	}

	liveWindow, liveStatus := resolveLiveWindow(client, id, isContribution)
	if liveStatus != 0 {
		log.Printf("%s GetPointsAll() GetIsOnliveByAPI() err=[%d]\n", ctx.EventID, liveStatus)
	}
	ctx.IsOnLive = liveWindow.IsLive
	ctx.StartedAt = liveWindow.StartedAt

	return ctx, true
}

func applyPointTransition(
	gschedule Gschedule,
	timestamp time.Time,
	ctx RoomContext,
	thpoint int,
) TransitionResult {
	uno := ctx.UserNo
	eventid := ctx.EventID
	point := ctx.Point
	rank := ctx.Rank
	gap := ctx.Gap
	isonlive := ctx.IsOnLive
	startedat := ctx.StartedAt
	isContribution := ctx.IsContribution
	knownEventUser := ctx.KnownEventUser

	result := TransitionResult{
		EventID:  eventid,
		UserNo:   uno,
		ScoreKey: makeRoomKey(uno, eventid),
		Point:    point,
		Rank:     rank,
		Gap:      gap,
	}

	if p, ok := scoremap.Load(result.ScoreKey); ok {
		if isonlive {
			// 配信中は同一配信枠として扱うので、オフライン継続回数をリセットする。
			p.(*LastScore).NoOffline = 0
		} else {
			// 非配信中は連続回数を数え、終了判定や区間確定のきっかけに使う。
			p.(*LastScore).NoOffline++
		}
	}

	pstatus := "n/a"
	ptime := ""
	if p, ok := scoremap.Load(result.ScoreKey); ok && p.(*LastScore).Eventid == gschedule.Eventid {
		if p.(*LastScore).Score == point && p.(*LastScore).Rank == rank {
			if !isonlive {
				// 同一値が続く非配信中のケースでは、Dup と NoOffline を使って
				// 3連続重複の削除や終了区間の確定を行う。
				if p.(*LastScore).Dup == 1 && p.(*LastScore).NoOffline > 1 {
					if p.(*LastScore).Sum0 > 0 {
						p.(*LastScore).Qstatus = "+" + humanize.Comma(int64(p.(*LastScore).Sum0))
					} else if p.(*LastScore).Sum0 < 0 {
						p.(*LastScore).Qstatus = "-" + humanize.Comma(int64(-p.(*LastScore).Sum0))
					}
					if p.(*LastScore).Tend.After(timestamp) {
						p.(*LastScore).Tend = timestamp
					}

					log.Printf("%s id=%6d !isonlive\n", eventid, uno)
					if _, ok := scoremap.Load(result.ScoreKey); !ok {
						log.Printf("%s scoremap[%s] not found.\n", eventid, result.ScoreKey)
						return result
					}

					ststart0 := p.(*LastScore).Tstart0.Format("01/02 15:04")
					stend := p.(*LastScore).Tend.Format("15:04")
					log.Printf("%s id=%6d ststart0 = [%s] stend = [%s]\n", eventid, uno, ststart0, stend)
					if ststart0 == "01/01 00:00" {
						ststart0 = ""
					} else if stend == "00:00" {
						p.(*LastScore).Tend = timestamp.Add(-10 * time.Minute)
						stend = p.(*LastScore).Tend.Format("15:04")
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

					if p.(*LastScore).Continued > 0 {
						p.(*LastScore).Qtime += fmt.Sprintf("(C%d)", p.(*LastScore).Continued)
					} else if p.(*LastScore).Continued == -1 {
						p.(*LastScore).Qtime += "(E)"
					} else if p.(*LastScore).Continued < -1 {
						p.(*LastScore).Qtime += "(U)"
					}

					p.(*LastScore).Continued = 0

					if isContribution && p.(*LastScore).Sum0 != 0 {
						result.TimeTable = &TimeTableCandidate{
							EventID:     eventid,
							UserNo:      uno,
							SampleTime:  timestamp.Add(5 * time.Minute),
							EarnedPoint: p.(*LastScore).Sum0,
							StartTime:   p.(*LastScore).Tstart0,
							EndTime:     p.(*LastScore).Tend,
						}
					}

					ptime = ""
					pstatus = "="
					log.Printf("%s id=%6d p = [%s], [%s] q= [%s], [%s]\n", eventid, uno, pstatus, ptime, p.(*LastScore).Qstatus, p.(*LastScore).Qtime)
					p.(*LastScore).Sum0 = 0
				}
				if p.(*LastScore).Dup != 0 {
					// 中央の重複データを削除するため、直前の points 行を記録しておく。
					deleteTS := p.(*LastScore).ts
					result.DeletePointTS = &deleteTS
					ptime = ""
					pstatus = "="
				}
				p.(*LastScore).Dup += 1
			} else {
				// 配信中は開始時刻の再検出が起こりうるため、Tstart1 を基準に
				// 同一配信枠かどうかを判定し、必要なら Continued を増やす。
				ptime = p.(*LastScore).Tstart0.Format("01/02 15:04:05")
				if startedat != p.(*LastScore).Tstart1 {
					p.(*LastScore).Tstart1 = startedat
					p.(*LastScore).Tend = startedat.Add(10000 * time.Hour)
					if p.(*LastScore).Sum0 != 0 {
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
			pdelta := point - p.(*LastScore).Score

			if pdelta != 0 {
				if isonlive {
					// 配信中の増分は、区間の開始時刻を startedat で揃えて積算する。
					if p.(*LastScore).Sum0 == 0 {
						p.(*LastScore).Tstart0 = startedat
						p.(*LastScore).Tend = startedat.Add(10000 * time.Hour)
						p.(*LastScore).Tstart1 = startedat
					} else if p.(*LastScore).Tstart1 != startedat {
						p.(*LastScore).Tstart1 = startedat
						p.(*LastScore).Continued++
					}
				} else {
					// 非配信中の増分は、イベント終了後の取り込みも含むので、
					// 区間開始を取得時刻基準に寄せて E/U 表示を付ける。
					if p.(*LastScore).Sum0 == 0 {
						p.(*LastScore).Tstart0 = timestamp.Add(-time.Duration(gschedule.Modmin*60+gschedule.Modsec) * time.Second)
						p.(*LastScore).Tend = timestamp
						p.(*LastScore).Continued = -1
					} else {
						p.(*LastScore).Tend = timestamp
					}
				}

				p.(*LastScore).Sum0 += pdelta
				// Sum0 は保存用ポイントの差分累積で、Qstatus/Qtime の表示元にもなる。
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
				// Score と Rank が変わらない場合は単なる連続観測なので Dup を増やす。
				pstatus = "="
				ptime = ""
				p.(*LastScore).Dup += 1
			}

			log.Printf("%s id=%6d Diff. %s %s\n", eventid, uno, ptime, pstatus)
			p.(*LastScore).Score = point
			p.(*LastScore).Rank = rank
			p.(*LastScore).ts = timestamp
			p.(*LastScore).Dup = 0
		}
	} else {
		if !knownEventUser && point < thpoint {
			return result
		}

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

		scoremap.Store(result.ScoreKey, &score)
	}

	result.PStatus = pstatus
	result.PTime = ptime
	result.ShouldPersist = true
	result.NeedsEventUser = !knownEventUser
	return result
}

func persistTransition(tx *sql.Tx, timestamp time.Time, result TransitionResult) *EventUserCandidate {
	if result.DeletePointTS != nil {
		DeleteFromPoints(tx, result.EventID, *result.DeletePointTS, result.UserNo)
	}
	if result.TimeTable != nil {
		InsertIntoTimeTable(
			result.TimeTable.EventID,
			result.TimeTable.UserNo,
			result.TimeTable.SampleTime,
			result.TimeTable.EarnedPoint,
			result.TimeTable.StartTime,
			result.TimeTable.EndTime,
		)
	}
	p, _ := scoremap.Load(result.ScoreKey)
	ls := p.(*LastScore)
	InsertIntoPoints(tx, timestamp, result.UserNo, result.Point, result.Rank, result.Gap, result.EventID, result.PStatus, result.PTime, p.(*LastScore).Qstatus, ls.Qtime)
	if !result.NeedsEventUser {
		return nil
	}
	return &EventUserCandidate{
		UserNo: result.UserNo,
		Rank:   result.Rank,
		Point:  result.Point,
	}
}

func finalizeEventUserUpdates(client *http.Client, eventid string, gschedule Gschedule, candidates []EventUserCandidate, timestamp time.Time) {
	for _, candidate := range candidates {
		id := candidate.UserNo
		point := candidate.Point
		rank := candidate.Rank
		itfc, err := GSE5Mlib.Dbmap.Get(srdblib.Eventuser{}, eventid, id)
		if err != nil {
			log.Printf("%s id=%6d GSE5Mlib.Dbmap.Get(Eventuser{},...) err=[%v]\n", eventid, id, err)
			continue
		}
		if itfc == nil {
			err := srdblib.UpinsEventuser(GSE5Mlib.Db, GSE5Mlib.Dbmap, client, rank, point, eventid, gschedule.Starttime, gschedule.Cmap, id, timestamp)
			if err != nil {
				log.Printf("%s id=%6d UpinsEventuser() err=[%v]\n", eventid, id, err)
			} else {
				log.Printf("%s id=%6d UpinsEventuser() ok\n", eventid, id)
			}
		}
	}
}

// gscheduleで指定したイベントについて配信者さんの獲得ポイントを取得し、保存条件に合致するものをDBに保存する。
func ScanActive(client *http.Client, gschedule Gschedule) (status int) {

	// 異常終了による処理の中断を防止する。
	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovered from panic:", r)
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
	stmt, err = GSE5Mlib.Db.Prepare(sqlstmt)
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
		err = setEventDontGetScore(gschedule.Eventid)
		if err != nil {
			log.Printf("%s update event err=[%s]\n", eventid, err.Error())
		}
	}

	thpoint := computeThresholdPoint(gschedule)

	collection, err := collectPointSnapshots(client, gschedule, timestamp, idList, cntrblist)
	if err != nil {
		return
	}
	idList = collection.IDList
	cntrblist = collection.CntrbList
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

	if collection.EventIsOver || timestamp.After(gschedule.Endtime.Add(1*time.Minute)) {
		//	イベントが終了した
		handleEventEnd(gschedule, trackedRoomsFromLists(idList, cntrblist), timestamp)
		return
	}

	nu := make([]EventUserCandidate, 0, 50)

	var tx *sql.Tx
	tx, err = GSE5Mlib.Db.Begin()
	if err != nil {
		log.Printf("%s GSE5Mlib.Db.Begin() err=[%s]\n", eventid, err.Error())
		return -1
	}
	defer tx.Rollback()

	for i, id := range idList {
		ctx, ok := buildRoomContext(client, gschedule, collection, id, cntrblist[i] == "Y")
		if !ok {
			continue
		}
		transition := applyPointTransition(
			gschedule,
			timestamp,
			ctx,
			thpoint,
		)
		if !transition.ShouldPersist {
			continue
		}

		log.Printf("%s id=%6d point=%d rank=%d\n", transition.EventID, ctx.UserNo, transition.Point, transition.Rank)
		candidate := persistTransition(tx, timestamp, transition)
		if candidate != nil {
			nu = append(nu, *candidate)
		}

	}

	if err = tx.Commit(); err != nil {
		log.Printf("%s tx.Commit() err=[%s]\n", eventid, err.Error())
		return -1
	}

	finalizeEventUserUpdates(client, eventid, gschedule, nu, timestamp)
	//	SaveScoremap()

	//	MakeComment()	MakeComment()はscoremapのキーがeventid+usernoとなったバージョンに対応していないので現時点では使えない。

	return
}
