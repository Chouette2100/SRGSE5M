// Copyright © 2022-2024 chouette.21.00@gmail.com
// Released under the MIT license
// https://opensource.org/licenses/mit-license.php
package main

import (
	//	"crypto/aes"
	//	"fmt"
	//	"log"
	//	"os"
	//	"strconv"
	//	"strings"
	//	"sync"
	"time"

	"log"
	//	"bufio"
	//	"io"

	//	"runtime"

	//	"net/http"

	//	"database/sql"

	_ "github.com/go-sql-driver/mysql"

	//	"github.com/go-gorp/gorp"

	//	"encoding/json"
	//	"github.com/360EntSecGroup-Skylar/excelize"

	//	. "MyModule/ShowroomCGIlib"
	"SRGSE5M/GSE5Mlib"
	"SRGSE5M/SRDBlib"

	//	"github.com/dustin/go-humanize"

	"github.com/Chouette2100/exsrapi/v2"
	//	"github.com/Chouette2100/srapi/v2"
	"github.com/Chouette2100/srdblib/v2"
)

func GetConfirmed(gschedule Gschedule) (status int) {

	var eventinf GSE5Mlib.Event_Inf
	var roominflist GSE5Mlib.RoomInfoList
	//	var roominf RoomInfo

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

	log.Printf("%s isconfirm =%t, isquest=%t\n", eventid, isconfirm, isquest)
	if isconfirm || isquest {
		sqlstmt := "update event set rstatus = ? where eventid = ?"
		_, srdblib.Dberr = srdblib.Db.Exec(sqlstmt, "Confirmed", eventid)

		if srdblib.Dberr != nil {
			log.Printf("%s GetConfirmed() update event err=[%s]\n", eventid, srdblib.Dberr.Error())
			status = -1
			return
		}

		if isconfirm {
			SRDBlib.MakePointPerSlot(eventid)
		}
	}

	return

}
