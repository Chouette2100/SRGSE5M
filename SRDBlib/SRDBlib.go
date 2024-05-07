/*
!
Copyright © 2022 chouette.21.00@gmail.com
Released under the MIT license
https://opensource.org/licenses/mit-license.php
*/
package SRDBlib

import (
	"strings"

	"fmt"
	"log"

	//	"math"
	"sort"
	"time"

	//	"database/sql"
	_ "github.com/go-sql-driver/mysql"

	"github.com/dustin/go-humanize"

	"github.com/Chouette2100/srdblib"
)

/*
01AA00	SRDBlib.goを導入する（最終結果確定時にMakePointPerSlot()を実行する、ことが目的）
01AA01	MakePointPerSlot()のperslotの変数宣言をループの中に入れる（毎回初期化されるように）
01AB00	stmtを使いまわしたとき、2回目の前にstmt.Close()を行う。
021AA00	gorpを導入するとともに srdblib を共通パッケージに変更する（第一ステップ）
*/

const Version = "021AA00"

type Event_Inf struct {
	Event_ID    string
	I_Event_ID  int
	Event_name  string
	Event_no    int
	MaxPoint    int
	Start_time  time.Time
	Sstart_time string
	Start_date  float64
	End_time    time.Time
	Send_time   string
	Period      string
	Dperiod     float64
	Intervalmin int
	Modmin      int
	Modsec      int
	Fromorder   int
	Toorder     int
	Resethh     int
	Resetmm     int
	Nobasis     int
	Maxdsp      int
	NoEntry     int
	NoRoom      int    //	ルーム数
	EventStatus string //	"Over", "BeingHeld", "NotHeldYet"
	Pntbasis    int
	Ordbasis    int
	League_ids  string
	Cmap        int
	Target      int
	Maxpoint    int
	//	Status		string		//	"Confirmed":	イベント終了日翌日に確定した獲得ポイントが反映されている。
}

type Color struct {
	Name  string
	Value string
}

// https://www.fukushihoken.metro.tokyo.lg.jp/kiban/machizukuri/kanren/color.files/colorudguideline.pdf
var Colorlist2 []Color = []Color{
	{"red", "#FF2800"},
	{"yellow", "#FAF500"},
	{"green", "#35A16B"},
	{"blue", "#0041FF"},
	{"skyblue", "#66CCFF"},
	{"lightpink", "#FFD1D1"},
	{"orange", "#FF9900"},
	{"purple", "#9A0079"},
	{"brown", "#663300"},
	{"lightgreen", "#87D7B0"},
	{"white", "#FFFFFF"},
	{"gray", "#77878F"},
}

var Colorlist1 []Color = []Color{
	{"cyan", "cyan"},
	{"magenta", "magenta"},
	{"yellow", "yellow"},
	{"royalblue", "royalblue"},
	{"coral", "coral"},
	{"khaki", "khaki"},
	{"deepskyblue", "deepskyblue"},
	{"crimson", "crimson"},
	{"orange", "orange"},
	{"lightsteelblue", "lightsteelblue"},
	{"pink", "pink"},
	{"sienna", "sienna"},
	{"springgreen", "springgreen"},
	{"blueviolet", "blueviolet"},
	{"salmon", "salmon"},
	{"lime", "lime"},
	{"red", "red"},
	{"darkorange", "darkorange"},
	{"skyblue", "skyblue"},
	{"lightpink", "lightpink"},
}

type ColorInf struct {
	Color      string
	Colorvalue string
	Selected   string
}

type ColorInfList []ColorInf

type RoomInfo struct {
	Name      string //	ルーム名のリスト
	Longname  string
	Shortname string
	Account   string //	アカウントのリスト、アカウントは配信のURLの最後の部分の英数字です。
	ID        string //	IDのリスト、IDはプロフィールのURLの最後の部分で5～6桁の数字です。
	Userno    int
	//	APIで取得できるデータ(1)
	Genre      string
	Rank       string
	Irank      int
	Nrank      string
	Prank      string
	Followers  int
	Sfollowers string
	Fans       int
	Fans_lst   int
	Level      int
	Slevel     string
	//	APIで取得できるデータ(2)
	Order        int
	Point        int //	イベント終了後12時間〜36時間はイベントページから取得できることもある
	Spoint       string
	Istarget     string
	Graph        string
	Iscntrbpoint string
	Color        string
	Colorvalue   string
	Colorinflist ColorInfList
	Formid       string
	Eventid      string
	Status       string
	Statuscolor  string
}

type RoomInfoList []RoomInfo

// sort.Sort()のための関数三つ
func (r RoomInfoList) Len() int {
	return len(r)
}

func (r RoomInfoList) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func (r RoomInfoList) Choose(from, to int) (s RoomInfoList) {
	s = r[from:to]
	return
}

var SortByFollowers bool

// 降順に並べる
func (r RoomInfoList) Less(i, j int) bool {
	//	return e[i].point < e[j].point
	if SortByFollowers {
		return r[i].Followers > r[j].Followers
	} else {
		return r[i].Point > r[j].Point
	}
}

type PerSlot struct {
	Timestart time.Time
	Dstart    string
	Tstart    string
	Tend      string
	Point     string
	Ipoint    int
	Tpoint    string
}

type PerSlotInf struct {
	Eventname   string
	Eventid     string
	Period      string
	Roomname    string
	Roomid      int
	Perslotlist []PerSlot
}

var Event_inf Event_Inf

//	var Db *sql.DB
//	var Err error

func SelectEventNoAndName(eventid string) (
	eventname string,
	period string,
	status int,
) {

	status = 0

	err := srdblib.Db.QueryRow("select event_name, period from event where eventid ='"+eventid+"'").Scan(&eventname, &period)

	if err == nil {
		return
	} else {
		log.Printf("err=[%s]\n", err.Error())
		if err.Error() != "sql: no rows in result set" {
			status = -2
			return
		}
	}

	status = -1
	return
}

func SelectEventInf(eventid string) (eventinf Event_Inf, status int) {

	status = 0

	sql := "select eventid,ieventid,event_name, period, starttime, endtime, noentry, intervalmin, modmin, modsec, "
	sql += " Fromorder, Toorder, Resethh, Resetmm, Nobasis, Maxdsp, cmap, target, maxpoint "
	sql += " from event where eventid = ?"
	err := srdblib.Db.QueryRow(sql, eventid).Scan(
		&eventinf.Event_ID,
		&eventinf.I_Event_ID,
		&eventinf.Event_name,
		&eventinf.Period,
		&eventinf.Start_time,
		&eventinf.End_time,
		&eventinf.NoEntry,
		&eventinf.Intervalmin,
		&eventinf.Modmin,
		&eventinf.Modsec,
		&eventinf.Fromorder,
		&eventinf.Toorder,
		&eventinf.Resethh,
		&eventinf.Resetmm,
		&eventinf.Nobasis,
		&eventinf.Maxdsp,
		&eventinf.Cmap,
		&eventinf.Target,
		&eventinf.Maxpoint,
	)

	if err != nil {
		log.Printf("%s\n", sql)
		log.Printf("err=[%s]\n", err.Error())
		//	if err.Error() != "sql: no rows in result set" {
		status = -1
		return
		//	}
	}

	//	log.Printf("eventno=%d\n", Event_inf.Event_no)

	start_date := eventinf.Start_time.Truncate(time.Hour).Add(-time.Duration(eventinf.Start_time.Hour()) * time.Hour)
	end_date := eventinf.End_time.Truncate(time.Hour).Add(-time.Duration(eventinf.End_time.Hour())*time.Hour).AddDate(0, 0, 1)

	//	log.Printf("start_t=%v\nstart_d=%v\nend_t=%v\nend_t=%v\n", Event_inf.Start_time, start_date, Event_inf.End_time, end_date)

	eventinf.Start_date = float64(start_date.Unix()) / 60.0 / 60.0 / 24.0
	eventinf.Dperiod = float64(end_date.Unix())/60.0/60.0/24.0 - Event_inf.Start_date

	//	log.Printf("Start_data=%f Dperiod=%f\n", eventinf.Start_date, eventinf.Dperiod)

	return
}

func SelectEventRoomInfList(
	eventid string,
	roominfolist *RoomInfoList,
) (
	eventname string,
	status int,
) {

	status = 0

	//	eventno := 0
	//	eventno, eventname, _ = SelectEventNoAndName(eventid)
	Event_inf, _ = SelectEventInf(eventid)

	//	eventno := Event_inf.Event_no
	eventname = Event_inf.Event_name

	sql := "select distinct u.userno, userid, user_name, longname, shortname, genre, `rank`, nrank, prank, level, followers, fans, fans_lst, e.istarget, e.graph, e.color, e.iscntrbpoints, e.point "
	sql += " from user u join eventuser e "
	sql += " where u.userno = e.userno and e.eventid= ?"
	if Event_inf.Start_time.After(time.Now()) {
		sql += " order by followers desc"
	} else {
		sql += " order by e.point desc"
	}

	stmt, err := srdblib.Db.Prepare(sql)
	if err != nil {
		log.Printf("SelectEventRoomInfList() Prepare() err=%s\n", err.Error())
		status = -5
		return
	}
	defer stmt.Close()

	rows, err := stmt.Query(eventid)
	if err != nil {
		log.Printf("SelectRoomIn() Query() (6) err=%s\n", err.Error())
		status = -6
		return
	}
	defer rows.Close()

	ColorlistA := Colorlist2
	ColorlistB := Colorlist1
	if Event_inf.Cmap == 1 {
		ColorlistA = Colorlist1
		ColorlistB = Colorlist2
	}

	colormap := make(map[string]int)

	for i := 0; i < len(ColorlistA); i++ {
		colormap[ColorlistA[i].Name] = i
	}

	var roominf RoomInfo

	i := 0
	for rows.Next() {
		err := rows.Scan(&roominf.Userno,
			&roominf.Account,
			&roominf.Name,
			&roominf.Longname,
			&roominf.Shortname,
			&roominf.Genre,
			&roominf.Rank,
			&roominf.Nrank,
			&roominf.Prank,
			&roominf.Level,
			&roominf.Followers,
			&roominf.Fans,
			&roominf.Fans_lst,
			&roominf.Istarget,
			&roominf.Graph,
			&roominf.Color,
			&roominf.Iscntrbpoint,
			&roominf.Point,
		)

		ci := 0
		for ; ci < len(ColorlistA); ci++ {
			if ColorlistA[ci].Name == roominf.Color {
				roominf.Colorvalue = ColorlistA[ci].Value
				break
			}
		}
		if ci == len(ColorlistA) {
			ci := 0
			for ; ci < len(ColorlistB); ci++ {
				if ColorlistB[ci].Name == roominf.Color {
					roominf.Colorvalue = ColorlistB[ci].Value
					break
				}
			}
			if ci == len(ColorlistB) {
				roominf.Colorvalue = roominf.Color
			}
		}

		if roominf.Istarget == "Y" {
			roominf.Istarget = "Checked"
		} else {
			roominf.Istarget = ""
		}
		if roominf.Graph == "Y" {
			roominf.Graph = "Checked"
		} else {
			roominf.Graph = ""
		}
		if roominf.Iscntrbpoint == "Y" {
			roominf.Iscntrbpoint = "Checked"
		} else {
			roominf.Iscntrbpoint = ""
		}
		roominf.Slevel = humanize.Comma(int64(roominf.Level))
		roominf.Sfollowers = humanize.Comma(int64(roominf.Followers))
		if roominf.Point < 0 {
			roominf.Spoint = ""
		} else {
			roominf.Spoint = humanize.Comma(int64(roominf.Point))
		}
		roominf.Formid = "Form" + fmt.Sprintf("%d", i)
		roominf.Eventid = eventid
		roominf.Name = strings.ReplaceAll(roominf.Name, "'", "’")
		if err != nil {
			log.Printf("SelectEventRoomInfList() Scan() err=%s\n", err.Error())
			status = -7
			return
		}
		//	var colorinf ColorInf
		colorinflist := make([]ColorInf, len(ColorlistA))

		for i := 0; i < len(ColorlistA); i++ {
			colorinflist[i].Color = ColorlistA[i].Name
			colorinflist[i].Colorvalue = ColorlistA[i].Value
		}

		roominf.Colorinflist = colorinflist
		if cidx, ok := colormap[roominf.Color]; ok {
			roominf.Colorinflist[cidx].Selected = "Selected"
		}
		*roominfolist = append(*roominfolist, roominf)

		i++
	}

	if err = rows.Err(); err != nil {
		log.Printf("SelectEventRoomInfList() rows err=%s\n", err.Error())
		status = -8
		return
	}

	if Event_inf.Start_time.After(time.Now()) {
		SortByFollowers = true
	} else {
		SortByFollowers = false
	}
	sort.Sort(*roominfolist)

	/*
		for i := 0; i < len(*roominfolist); i++ {

			sql = "select max(point) from points where "
			sql += " user_id = " + fmt.Sprintf("%d", (*roominfolist)[i].Userno)
			//	sql += " and event_id = " + fmt.Sprintf("%d", eventno)
			sql += " and event_id = " + eventid

			err = Db.QueryRow(sql).Scan(&(*roominfolist)[i].Point)
			(*roominfolist)[i].Spoint = humanize.Comma(int64((*roominfolist)[i].Point))

			if err == nil {
				continue
			} else {
				log.Printf("err=[%s]\n", err.Error())
				if err.Error() != "sql: no rows in result set" {
					eventno = -2
					continue
				} else {
					(*roominfolist)[i].Point = -1
					(*roominfolist)[i].Spoint = ""
				}
			}
		}
	*/

	return
}

func SelectPointList(userno int, eventid string) (norow int, tp *[]time.Time, pp *[]int) {

	norow = 0

	//	log.Printf("SelectPointList() userno=%d eventid=%s\n", userno, eventid)
	stmt1, err := srdblib.Db.Prepare("SELECT count(*) FROM points where user_id = ? and eventid = ?")
	if err != nil {
		//	log.Fatal(err)
		log.Printf("err=[%s]\n", err.Error())
		//	status = -1
		return
	}
	defer stmt1.Close()

	//	var norow int
	err = stmt1.QueryRow(userno, eventid).Scan(&norow)
	if err != nil {
		//	log.Fatal(err)
		log.Printf("err=[%s]\n", err.Error())
		//	status = -1
		return
	}
	//	fmt.Println(norow)

	//	----------------------------------------------------

	stmt1.Close()

	//	stmt1, err = Db.Prepare("SELECT max(t.t) FROM timeacq t join points p where t.idx=p.idx and user_id = ? and event_id = ?")
	stmt1, err = srdblib.Db.Prepare("SELECT max(ts) FROM points where user_id = ? and eventid = ?")
	if err != nil {
		//	log.Fatal(err)
		log.Printf("err=[%s]\n", err.Error())
		//	status = -1
		return
	}
	//	defer stmt1.Close()

	var tfinal time.Time
	err = stmt1.QueryRow(userno, eventid).Scan(&tfinal)
	if err != nil {
		//	log.Fatal(err)
		log.Printf("err=[%s]\n", err.Error())
		//	status = -1
		return
	}
	islastdata := false
	if tfinal.After(Event_inf.End_time.Add(time.Duration(-Event_inf.Intervalmin) * time.Minute)) {
		islastdata = true
	}
	//	fmt.Println(norow)

	//	----------------------------------------------------

	t := make([]time.Time, norow)
	point := make([]int, norow)
	if islastdata {
		t = make([]time.Time, norow+1)
		point = make([]int, norow+1)
	}

	tp = &t
	pp = &point

	if norow == 0 {
		return
	}

	//	----------------------------------------------------

	//	stmt2, err := Db.Prepare("select t.t, p.point from points p join timeacq t on t.idx = p.idx where user_id = ? and event_id = ? order by t.t")
	stmt2, err := srdblib.Db.Prepare("select ts, point from points where user_id = ? and eventid = ? order by ts")
	if err != nil {
		//	log.Fatal(err)
		log.Printf("err=[%s]\n", err.Error())
		//	status = -1
		return
	}
	defer stmt2.Close()

	rows, err := stmt2.Query(userno, eventid)
	if err != nil {
		//	log.Fatal(err)
		log.Printf("err=[%s]\n", err.Error())
		//	status = -1
		return
	}
	defer rows.Close()
	i := 0
	for rows.Next() {
		err := rows.Scan(&t[i], &point[i])
		if err != nil {
			//	log.Fatal(err)
			log.Printf("err=[%s]\n", err.Error())
			//	status = -1
			return
		}
		i++

	}
	if err = rows.Err(); err != nil {
		//	log.Fatal(err)
		log.Printf("err=[%s]\n", err.Error())
		//	status = -1
		return
	}

	if islastdata {
		t[norow] = t[norow-1].Add(15 * time.Minute)
		point[norow] = point[norow-1]
	}

	tp = &t
	pp = &point

	return
}

func UpdatePointsSetQstatus(
	eventid string,
	userno int,
	tstart string,
	tend string,
	point string,
) (status int) {
	status = 0

	log.Printf("  *** UpdatePointsSetQstatus() *** eventid=%s userno=%d\n", eventid, userno)

	nrow := 0
	//	err := Db.QueryRow("select count(*) from points where eventid = ? and user_id = ? and pstatus = 'Conf.'", eventid, userno).Scan(&nrow)
	sql := "select count(*) from points where eventid = ? and user_id = ? and ( pstatus = 'Conf.' or pstatus = 'Prov.' )"
	err := srdblib.Db.QueryRow(sql, eventid, userno).Scan(&nrow)

	if err != nil {
		log.Printf("select count(*) from user ... err=[%s]\n", err.Error())
		status = -1
		return
	}

	if nrow != 1 {
		return
	}

	log.Printf("  *** UpdatePointsSetQstatus() Update!\n")

	sql = "update points set qstatus =?,"
	sql += "qtime=? "
	//	sql += "where user_id=? and eventid = ? and pstatus = 'Conf.'"
	sql += "where user_id=? and eventid = ? and ( pstatus = 'Conf.' or pstatus = 'Prov.' )"
	stmt, err := srdblib.Db.Prepare(sql)
	if err != nil {
		log.Printf("UpdatePointsSetQstatus() Update/Prepare err=%s\n", err.Error())
		status = -1
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(point, tstart+"--"+tend, userno, eventid)

	if err != nil {
		log.Printf("error(UpdatePointsSetQstatus() Update/Exec) err=%s\n", err.Error())
		status = -2
	}

	return
}

func MakePointPerSlot(eventid string) (perslotinflist []PerSlotInf, status int) {

	var perslotinf PerSlotInf
	var event_inf Event_Inf

	status = 0

	event_inf.Event_ID = eventid
	//	eventno, eventname, period := SelectEventNoAndName(eventid)
	eventname, period, _ := SelectEventNoAndName(eventid)

	var roominfolist RoomInfoList

	_, sts := SelectEventRoomInfList(eventid, &roominfolist)

	if sts != 0 {
		log.Printf("status of SelectEventRoomInfList() =%d\n", sts)
		status = sts
		return
	}

	for i := 0; i < len(roominfolist); i++ {

		if roominfolist[i].Graph != "Checked" {
			continue
		}

		var perslot PerSlot

		userid := roominfolist[i].Userno

		perslotinf.Eventname = eventname
		perslotinf.Eventid = eventid
		perslotinf.Period = period

		perslotinf.Roomname = roominfolist[i].Name
		perslotinf.Roomid = userid
		perslotinf.Perslotlist = make([]PerSlot, 0)

		norow, tp, pp := SelectPointList(userid, eventid)

		if norow == 0 {
			continue
		}

		sameaslast := true
		plast := (*pp)[0]
		pprv := (*pp)[0]
		tdstart := ""
		tstart := time.Now().Truncate(time.Second)

		for i, t := range *tp {
			//	if (*pp)[i] != plast && sameaslast {
			if (*pp)[i] != plast {
				tstart = t
				/*
					if i != 0 {
						log.Printf("(1) (*pp)[i]=%d, plast=%d, sameaslast=%v, (*tp)[i]=%s, (*tp)[i-1]=%s\n", (*pp)[i], plast, sameaslast, (*tp)[i].Format("01/02 15:04"), (*tp)[i-1].Format("01/02 15:04"))
					} else {
						log.Printf("(1) (*pp)[i]=%d, plast=%d, sameaslast=%v, (*tp)[i]=%s, \n", (*pp)[i], plast, sameaslast, (*tp)[i].Format("01/02 15:04"))
					}
				*/
				if sameaslast {
					//	これまで変化しなかった獲得ポイントが変化し始めた
					pdstart := t.Add(time.Duration(-Event_inf.Modmin) * time.Minute).Format("2006/01/02")
					if pdstart != tdstart {
						perslot.Dstart = pdstart
						tdstart = pdstart
					} else {
						perslot.Dstart = ""
					}
					perslot.Timestart = t.Add(time.Duration(-Event_inf.Modmin) * time.Minute)
					//	perslot.Tstart = t.Add(time.Duration(-Event_inf.Modmin) * time.Minute).Format("15:04")
					if t.Sub((*tp)[i-1]) < 31*time.Minute {
						perslot.Tstart = perslot.Timestart.Format("15:04")
					} else {
						perslot.Tstart = "n/a"
					}
					//	perslot.Tstart = perslot.Timestart.Format("15:04")

					sameaslast = false
					//	} else if (*pp)[i] == plast && !sameaslast && (*tp)[i].Sub((*tp)[i-1]) > 11*time.Minute {
				}
			} else if (*pp)[i] == plast {
				//	if !sameaslast && (*tp)[i].Sub((*tp)[i-1]) > 16*time.Minute {
				if !sameaslast && t.Sub(tstart) > 11*time.Minute {
					//	if !sameaslast {
					/*
						if i != 0 {
							log.Printf("(2) (*pp)[i]=%d, plast=%d, sameaslast=%v, (*tp)[i]=%s, (*tp)[i-1]=%s\n", (*pp)[i], plast, sameaslast, (*tp)[i].Format("01/02 15:04"), (*tp)[i-1].Format("01/02 15:04"))
						} else {
							log.Printf("(2) (*pp)[i]=%d, plast=%d, sameaslast=%v, (*tp)[i]=%s, \n", (*pp)[i], plast, sameaslast, (*tp)[i].Format("01/02 15:04"))
						}
					*/
					if perslot.Tstart != "n/a" {
						perslot.Tend = (*tp)[i-1].Add(time.Duration(-Event_inf.Modmin) * time.Minute).Format("15:04")
					} else {
						perslot.Tend = "n/a"
					}
					perslot.Ipoint = plast - pprv
					perslot.Point = humanize.Comma(int64(plast - pprv))
					perslot.Tpoint = humanize.Comma(int64(plast))
					sameaslast = true
					perslotinf.Perslotlist = append(perslotinf.Perslotlist, perslot)
					pprv = plast
				}
				//	sameaslast = true
			}
			/* else
			{
					if i != 0 {
						log.Printf("(3) (*pp)[i]=%d, plast=%d, sameaslast=%v, (*tp)[i]=%s, (*tp)[i-1]=%s\n", (*pp)[i], plast, sameaslast, (*tp)[i].Format("01/02 15:04"), (*tp)[i-1].Format("01/02 15:04"))
					} else {
						log.Printf("(3) (*pp)[i]=%d, plast=%d, sameaslast=%v, (*tp)[i]=%s, \n", (*pp)[i], plast, sameaslast, (*tp)[i].Format("01/02 15:04"))
					}
			}
			*/
			plast = (*pp)[i]
		}

		if len(perslotinf.Perslotlist) != 0 {
			perslotinflist = append(perslotinflist, perslotinf)
		}

		UpdatePointsSetQstatus(eventid, userid, perslot.Tstart, perslot.Tend, perslot.Point)

	}

	return
}
