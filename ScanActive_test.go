//	Copyright © 2022-2024 chouette.21.00@gmail.com
//	Released under the MIT license
//	https://opensource.org/licenses/mit-license.php

package main

import (
	"testing"
	"time"
)

func resetScoremapForTest() {
	scoremap.Range(func(key, value any) bool {
		scoremap.Delete(key)
		return true
	})
}

func TestApplyPointTransition(t *testing.T) {
	t.Cleanup(resetScoremapForTest)

	baseSchedule := Gschedule{
		Eventid: "event-1",
		Modmin:  5,
		Modsec:  0,
	}
	timestamp := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	startedAt := time.Date(2026, 10, 1, 11, 30, 0, 0, time.UTC)

	tests := []struct {
		name          string
		seed          *LastScore
		ctx           RoomContext
		thpoint       int
		wantPersist   bool
		wantDelete    bool
		wantTimeTable bool
		wantScore     *LastScore
	}{
		{
			name: "new live room creates state",
			ctx: RoomContext{
				UserNo:         1001,
				EventID:        "event-1",
				Point:          1200,
				Rank:           25,
				Gap:            10,
				IsOnLive:       true,
				StartedAt:      startedAt,
				IsContribution: true,
				KnownEventUser: false,
			},
			thpoint:     100,
			wantPersist: true,
			wantScore: &LastScore{
				Eventid:   "event-1",
				Score:     1200,
				Rank:      25,
				Tstart0:   startedAt,
				Tstart1:   startedAt,
				Continued: -999,
				Dup:       0,
			},
		},
		{
			name: "offline repeated point closes timetable and marks delete",
			seed: &LastScore{
				Eventid:   "event-1",
				Score:     1500,
				Rank:      11,
				ts:        timestamp.Add(-2 * time.Minute),
				Dup:       1,
				Sum0:      230,
				Tstart0:   startedAt,
				Tend:      timestamp.Add(10 * time.Minute),
				Tstart1:   startedAt,
				Continued: 0,
				Qstatus:   "+230",
				Qtime:     "01/01 00:00--12:10",
				NoOffline: 2,
			},
			ctx: RoomContext{
				UserNo:         1002,
				EventID:        "event-1",
				Point:          1500,
				Rank:           11,
				Gap:            3,
				IsOnLive:       false,
				StartedAt:      startedAt,
				IsContribution: true,
				KnownEventUser: true,
			},
			thpoint:       100,
			wantPersist:   true,
			wantDelete:    true,
			wantTimeTable: true,
			wantScore: &LastScore{
				Eventid:   "event-1",
				Score:     1500,
				Rank:      11,
				Tstart0:   startedAt,
				Tend:      timestamp,
				Tstart1:   startedAt,
				Continued: 0,
				Dup:       2,
				Sum0:      0,
				Qstatus:   "+230",
			},
		},
		{
			name: "unknown room below threshold is skipped",
			ctx: RoomContext{
				UserNo:         1003,
				EventID:        "event-1",
				Point:          40,
				Rank:           80,
				Gap:            0,
				IsOnLive:       false,
				StartedAt:      startedAt,
				IsContribution: false,
				KnownEventUser: false,
			},
			thpoint:     100,
			wantPersist: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetScoremapForTest()
			if tt.seed != nil {
				scoremap.Store(makeRoomKey(tt.ctx.UserNo, tt.ctx.EventID), tt.seed)
			}

			result := applyPointTransition(baseSchedule, timestamp, tt.ctx, tt.thpoint)

			if result.ShouldPersist != tt.wantPersist {
				t.Fatalf("ShouldPersist = %v, want %v", result.ShouldPersist, tt.wantPersist)
			}
			if (result.DeletePointTS != nil) != tt.wantDelete {
				t.Fatalf("DeletePointTS presence = %v, want %v", result.DeletePointTS != nil, tt.wantDelete)
			}
			if (result.TimeTable != nil) != tt.wantTimeTable {
				t.Fatalf("TimeTable presence = %v, want %v", result.TimeTable != nil, tt.wantTimeTable)
			}

			if !tt.wantPersist {
				if result.PStatus != "" || result.PTime != "" {
					t.Fatalf("unexpected status for skipped case: PStatus=%q PTime=%q", result.PStatus, result.PTime)
				}
				return
			}

			loaded, ok := scoremap.Load(makeRoomKey(tt.ctx.UserNo, tt.ctx.EventID))
			if !ok {
				t.Fatalf("scoremap entry not found")
			}
			got := loaded.(*LastScore)
			if tt.wantScore != nil {
				if got.Eventid != tt.wantScore.Eventid || got.Score != tt.wantScore.Score || got.Rank != tt.wantScore.Rank {
					t.Fatalf("score state mismatch: got=%+v want=%+v", got, tt.wantScore)
				}
				if !got.Tstart0.Equal(tt.wantScore.Tstart0) {
					t.Fatalf("Tstart0 = %v, want %v", got.Tstart0, tt.wantScore.Tstart0)
				}
				if !got.Tstart1.Equal(tt.wantScore.Tstart1) {
					t.Fatalf("Tstart1 = %v, want %v", got.Tstart1, tt.wantScore.Tstart1)
				}
				if got.Continued != tt.wantScore.Continued {
					t.Fatalf("Continued = %d, want %d", got.Continued, tt.wantScore.Continued)
				}
			}
		})
	}
}
