/*
!
Copyright © 2022 chouette.21.00@gmail.com
Released under the MIT license
https://opensource.org/licenses/mit-license.php
*/
package GSE5Mlib

import (
	"log"
	"fmt"
	"os"
	"time"

	"net/http"
	"reflect"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/go-gorp/gorp"


	"SRGSE5M/SRDBlib"

	//	"github.com/Chouette2100/srapi"
	"github.com/Chouette2100/exsrapi"
	"github.com/Chouette2100/srdblib"
)

func TestGetEventInfAndRoomList(t *testing.T) {
	type args struct {
		eventid      string
		ieventid     int
		breg         int
		ereg         int
		eventinfo    *Event_Inf
		roominfolist *RoomInfoList
	}

	var eventinf Event_Inf
	var roominflist RoomInfoList

	tests := []struct {
		name        string
		args        args
		wantIsquest bool
		wantStatus  int
	}{
		// TODO: Add test cases.
		{
			name: "test1",
			args: args{
				//	eventid:      "azabusmith-mc0608",
				//	ieventid:     36323,
				eventid:      "enjoykaraoke_vol114",
				ieventid:     36142,
				breg:         1,
				ereg:         3,
				eventinfo:    &eventinf,
				roominfolist: &roominflist,
			},
			wantIsquest: false,
			wantStatus:  0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIsquest, gotStatus := GetEventInfAndRoomList(tt.args.eventid, tt.args.ieventid, tt.args.breg, tt.args.ereg, tt.args.eventinfo, tt.args.roominfolist)
			if gotIsquest != tt.wantIsquest {
				t.Errorf("GetEventInfAndRoomList() gotIsquest = %v, want %v", gotIsquest, tt.wantIsquest)
			}
			if gotStatus != tt.wantStatus {
				t.Errorf("GetEventInfAndRoomList() gotStatus = %v, want %v", gotStatus, tt.wantStatus)
			}
		})
	}
}

func TestGetIsOnliveByAPI(t *testing.T) {
	type args struct {
		client  *http.Client
		room_id string
	}

	logfilename := Version + "_" + SRDBlib.Version + "_" + time.Now().Format("20060102") + ".txt"
	logfile, err := os.OpenFile(logfilename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		panic("cannnot open logfile: " + logfilename + err.Error())
	}
	defer logfile.Close()
	log.SetOutput(logfile)
	//	log.SetOutput(io.MultiWriter(logfile, os.Stdout))

	log.Printf(" ****************************\n")
	log.Printf(" GetScoreEvery5Minutes version=%s\n", Version)

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

	//	srdblib.Dbmap.AddTableWithName(srdblib.Wuser{}, "wuser").SetKeys(false, "Userno")
	//	srdblib.Dbmap.AddTableWithName(srdblib.Userhistory{}, "wuserhistory").SetKeys(false, "Userno", "Ts")
	//	srdblib.Dbmap.AddTableWithName(srdblib.Event{}, "wevent").SetKeys(false, "Eventid")
	//	srdblib.Dbmap.AddTableWithName(srdblib.Eventuser{}, "weventuser").SetKeys(false, "Eventid", "Userno")
	srdblib.Dbmap.AddTableWithName(srdblib.Event{}, "event").SetKeys(false, "Eventid")

	//      cookiejarがセットされたHTTPクライアントを作る
	client, jar, err := exsrapi.CreateNewClient("ShowroomCGI")
	if err != nil {
		log.Printf("CreateNewClient: %s\n", err.Error())
		return
	}
	//      すべての処理が終了したらcookiejarを保存する。
	defer jar.Save()

	tests := []struct {
		name          string
		args          args
		wantIsonlive  bool
		wantStartedat time.Time
		wantStatus    int
	}{
		// TODO: Add test cases.
		{
			name: "TestGetIsOnliveByAPI-1",
			args: args {
				client: client,
				room_id: "338333",
			},
			wantIsonlive:  false,
			wantStartedat: time.Time{},
			wantStatus:    0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIsonlive, gotStartedat, gotStatus := GetIsOnliveByAPI(tt.args.client, tt.args.room_id)
			log.Printf("GetIsOnliveByAPI() Isonlive = %v, Starteddat=%+v Status=%v\n", gotIsonlive, gotStartedat, gotStatus)
			if gotIsonlive != tt.wantIsonlive {
				t.Errorf("GetIsOnliveByAPI() gotIsonlive = %v, want %v", gotIsonlive, tt.wantIsonlive)
			}
			if !reflect.DeepEqual(gotStartedat, tt.wantStartedat) {
				t.Errorf("GetIsOnliveByAPI() gotStartedat = %v, want %v", gotStartedat, tt.wantStartedat)
			}
			if gotStatus != tt.wantStatus {
				t.Errorf("GetIsOnliveByAPI() gotStatus = %v, want %v", gotStatus, tt.wantStatus)
			}
		})
	}
}
