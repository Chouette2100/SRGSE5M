//	Copyright © 2022-2024 chouette.21.00@gmail.com
//	Released under the MIT license
//	https://opensource.org/licenses/mit-license.php

package main

import (
	//	"crypto/aes"
	//	"fmt"
	"log"
	//	"os"
	//	"strconv"
	//	"strings"
	//	"sync"
	"time"

	//	. "log"
	//	"bufio"
	//	"io"

	//	"runtime"

	//	"net/http"

	"database/sql"

	_ "github.com/go-sql-driver/mysql"

	//	"github.com/go-gorp/gorp"

	//	"encoding/json"
	//	"github.com/360EntSecGroup-Skylar/excelize"

	//	. "MyModule/ShowroomCGIlib"
	//	"SRGSE5M/GSE5Mlib"
	//	"SRGSE5M/SRDBlib"

	//	"github.com/dustin/go-humanize"

	"github.com/Chouette2100/exsrapi"
	//	"github.com/Chouette2100/srapi"
	"github.com/Chouette2100/srdblib"
)

//  現時点で（確定データ取得を含む）獲得ポイントデータ取得が必要なイベントの一覧を作成する
func GetSchedule() (
	gschedulelist Gschedulelist,
	status int,
) {

	cmt0 := "=========="
	fncname := exsrapi.FuncNameOfThisFunction() + "()"
	//	fncname := "GetSchedule()"
	log.Println(cmt0, ">>>>>>>>>>>>>>>>>>", fncname, ">>>>>>>>>>>>>>>>>>>")
	defer exsrapi.PrintExf(cmt0, fncname)()

	var stmt *sql.Stmt
	var rows *sql.Rows

	//	eventno := 0
	eventid := ""

	tnow := time.Now()

	//	開催中のイベントを取得する
	sqlstmt := "select eventid, ieventid, starttime, endtime, rstatus, fromorder, toorder, cmap from event "
	sqlstmt += " where starttime < ? and endtime > ? "
	stmt, Err := srdblib.Db.Prepare(sqlstmt)
	if Err != nil {
		log.Printf("GetSchedule() Prepare() err=%s\n", Err.Error())
		status = -5
		return
	}
	defer stmt.Close()

	//	endtimeの比較対象を現在時から48時間マイナスしてあるのは、翌日発表の確定値を取得する必要あるイベントも含めるため
	rows, Err = stmt.Query(tnow, tnow.Add(-48 * time.Hour))
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
	var fromorder, toorder int
	var cmap int

	i := 0
	for rows.Next() {
		Err = rows.Scan(&eventid, &ieventid, &starttime, &endtime, &rstatus, &fromorder, &toorder, &cmap)

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
		gschedule.Starttime = starttime
		gschedule.Endtime = endtime
		gschedule.Done = false
		gschedule.Fromorder = fromorder
		gschedule.Toorder = toorder
		gschedule.Cmap = cmap
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
