/*
!
Copyright © 2022 chouette.21.00@gmail.com
Released under the MIT license
https://opensource.org/licenses/mit-license.php
*/
package GSE5Mlib

import (
	"testing"

	_ "github.com/go-sql-driver/mysql"
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
