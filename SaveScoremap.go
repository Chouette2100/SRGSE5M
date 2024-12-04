// Copyright © 2022-2024 chouette2100@gmail.com
// Released under the MIT license
// https://opensource.org/licenses/mit-license.php
package main

import (
	//	"crypto/aes"
	"fmt"
	"log"
	"os"
	"strconv"
	//	"sync"
	"strings"
	"time"
	//	. "log"
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
	//	"SRGSE5M/SRDBlib"

	//	"github.com/dustin/go-humanize"

	"github.com/Chouette2100/exsrapi"
	//	"github.com/Chouette2100/srapi"
	//	"github.com/Chouette2100/srdblib"
)

func SaveScoremap() (err error) {

	cmt0 := "=========="
	fncname := exsrapi.FuncNameOfThisFunction() + "()"
	log.Println(cmt0, ">>>>>>>>>>>>>>>>>>", fncname, ">>>>>>>>>>>>>>>>>>>")
	defer exsrapi.PrintExf(cmt0, fncname)()

	var file *os.File
	file, err = os.OpenFile("scoremap.txt", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	if err != nil {
		log.Println(" Can't open file. [", "scoremap.txt", "]")
		err = fmt.Errorf("OpenFile() err=%w", err)
		return
	}
	defer file.Close()

	//	fmt.Fprintf(file, "%d\n", -1)
	//	fmt.Fprintf(file, "%d\n", -2)
	fmt.Fprintf(file, "%d\n", -3)
	//	for id, lastscore := range scoremap {
	no := 0
	scoremap.Range(func(id, value interface{}) bool {
		no++
		//	log.Println("key:", id, " value:", value)
		ls := value.(*LastScore)
		//	fmt.Fprintf(file, "%d\n", id)
		//	fmt.Fprintf(file, "%s\n", ls.Eventid)
		fmt.Fprintf(file, "%s\n", id)
		fmt.Fprintf(file, "%d %d %d %d\n", ls.Score, ls.Rank, ls.Dup, ls.Sum0)
		fmt.Fprintf(file, "%q\n", ls.ts.Format("2006/01/02 15:04:05 MST"))
		fmt.Fprintf(file, "%q\n", ls.Tstart0.Format("2006/01/02 15:04:05 MST"))
		fmt.Fprintf(file, "%q\n", ls.Tstart1.Format("2006/01/02 15:04:05 MST"))
		fmt.Fprintf(file, "%d\n", ls.Continued)
		fmt.Fprintf(file, "%q\n", ls.Qstatus)
		fmt.Fprintf(file, "%q\n", ls.Qtime)

		fmt.Fprintf(file, "%d\n", ls.NoOffline)

		//	file.Write([]byte(lastscore))
		//	err = binary.Write(file, binary.LittleEndian, lastscore)
		//	fmt.Printf("%v\n%#v\n", err, lastscore)

		return true

	})
	//	}

	//	file.Close()
	log.Printf("saved=%d\n", no)

	return
}

// デーモンをrestartしたときデータの継続性を確保するためのデータを読み込む
func RestoreScoremap() (err error) {

	cmt0 := "=========="
	fncname := exsrapi.FuncNameOfThisFunction() + "()"
	log.Println(cmt0, ">>>>>>>>>>>>>>>>>>", fncname, ">>>>>>>>>>>>>>>>>>>")
	defer exsrapi.PrintExf(cmt0, fncname)()

	var file *os.File
	file, err = os.OpenFile("scoremap.txt", os.O_RDONLY, 0644)
	if err != nil {
		log.Println(" Can't open file. [", "scoremap.txt", "]")
		err = fmt.Errorf("OpenFile() err=%w", err)
		return
	}
	defer file.Close()

	fver := 0
	id := 0
	ts := ""
	tstart0 := ""
	tstart1 := ""
	eventid := ""
	first := true
	firstrec := ""

	_, err = fmt.Fscanf(file, "%d\n", &id)
	if err != nil {
		err = fmt.Errorf("OpenFile() err=%w", err)
		return
	}

	if id < 0 {
		fver = -id
	}

	no := 0
	noiv := 0
	for {
		var lastscore LastScore

		switch fver {
		case 0:
			if !first {
				_, err = fmt.Fscanf(file, "%d\n", &id)
			}
			if err == nil {
				_, err = fmt.Fscanf(file, "%s\n", &eventid)
			}
		case 1, 2:
			_, err = fmt.Fscanf(file, "%d\n", &id)
			if err == nil {
				_, err = fmt.Fscanf(file, "%s\n", &eventid)
			}
		case 3:
			_, err = fmt.Fscanf(file, "%s\n", &firstrec)
			if err == nil {
				firstd := strings.Split(firstrec, "#")
				if len(firstd) == 1 {
					id, _ = strconv.Atoi(firstd[0])
				} else {
					id, _ = strconv.Atoi(firstd[0])
					eventid = firstd[1]
				}
			}
		}
		if err != nil {
			if err.Error() == "EOF" {
				err = nil
				break
			}
			err = fmt.Errorf("Fscanf() err=%w", err)
			return
		}

		//	log.Printf("RestoreScoremap() eventid=%s, id=%d\n", eventid, id)

		if _, ok := eventmap[eventid]; !ok {
			eventinf, _ := GSE5Mlib.SelectEventInf(eventid)
			eventmap[eventid] = &eventinf
		}
		if eventmap[eventid].End_time.Add(48 * time.Hour).Before(time.Now()) {
			log.Printf("ignored eventid=%s, id=%d\n", eventid, id)
			continue
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
		//	log.Printf("%v\n%#v %v\n", err, lastscore, lastscore.ts)

		if fver > 1 {
			fmt.Fscanf(file, "%d\n", &lastscore.NoOffline)
			//	log.Printf("%d\n", lastscore.NoOffline)
		}

		no++
		if (*eventmap[eventid]).End_time.Before(time.Now()) {
			log.Printf("ignored eventid=%s, id=%d\n", eventid, id)
			noiv++
			continue
		}

		key := strconv.Itoa(id) + "#" + eventid
		scoremap.Store(key, &lastscore)

	}
	log.Printf("restored=%d, ignored=%d\n", no-noiv, noiv)

	return
}
