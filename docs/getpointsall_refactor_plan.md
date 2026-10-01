## Plan: GetPointsAll 再設計

GetPointsAll は、取得・状態遷移・文字列整形・保存が 1 つに混在しているため、まず挙動互換を保ったまま責務を分離し、その後に配信状態判定を API から bchistory に置き換える。最初から保存形式を変えるのではなく、段階的に見通しの改善と API 変更耐性の向上を優先する。

### Implementation progress

- Phase 1 の初手として、scoremap キー生成、しきい値計算、DontGetScore 更新、配信状態取得ラッパー、イベント終了処理を GetPointsAll から補助関数へ抽出した。
- 収集処理の前半ブロックを collectPointSnapshots() として分離し、ランキング取得、対象ルーム拡張、個別ポイント補完を GetPointsAll 本体から切り離した。
- 状態遷移の巨大分岐を applyPointTransition() として分離し、GetPointsAll のループ本体から A-1-1、A-1-2、A-2、新規作成の処理塊を切り出した。
- RoomContext と buildRoomContext() を導入し、各ルームごとの point/rank 解決、block_id=0 の順位補正、イベント整合性確認、配信状態取得をループ本体の前処理から切り出した。
- persistTransition() と finalizeEventUserUpdates() を導入し、points への保存と eventuser 補完更新を GetPointsAll の本体末尾から分離した。あわせて tx.Commit() のエラー確認を追加した。
- TransitionResult に削除対象 points 行と timetable 追記内容を持たせ、applyPointTransition() から DeleteFromPoints() / InsertIntoTimeTable() の直接呼び出しを外して persistTransition() 側へ寄せた。
- applyPointTransition() は RoomContext を受け取る形に寄せ、状態遷移の入力を 1 つの構造体で渡せるようにした。
- persistTransition() も TransitionResult を主入力に寄せ、保存側が参照する情報を遷移結果に集約した。
- この時点では状態遷移ロジックと永続化ロジックはまだ GetPointsAll 内に残しており、挙動互換を優先している。
- 現状の Phase 1 は、責務分離の骨格はほぼ完了しており、残りは LastScore の意味整理とテスト観点の補強が中心になっている。
- applyPointTransition() を直接叩く最小のテーブル駆動テストを追加し、外部 API に依存しない状態遷移の確認手順を先に固定した。

### Steps

1. Phase 1: 現状の振る舞いを固定する。GetPointsAll の入力・出力・副作用を棚卸しし、最低限守るべき仕様を文章化する。対象は points への INSERT、eventuser 更新、timetable への書き込み、scoremap 更新、3 連続重複時の中央削除、イベント終了時処理。
2. Phase 1: ScanActive.go の GetPointsAll をオーケストレーション関数へ縮小する。役割は、ランキング/ポイント収集、処理対象ルーム確定、配信枠情報取得、状態遷移計算、永続化、イベント終了処理に分ける。
3. Phase 1: API 分岐を収集層へ隔離する。現在の GetEventsRankingByApi、GetEventRankingByApi、GetEventQuestRoomsByApi、GetEventBlockRanking、GetPointByApi をまとめ、最終的に eventid・userno ごとの point/rank/gap を返す形へ寄せる。
4. Phase 1: ルーム単位の状態遷移を専用関数へ抽出する。現行の A-1-1、A-1-2、A-2、新規作成の分岐を 1 か所に集約し、更新後 LastScore、保存用レコード、重複削除要否、timetable 追記要否を返す。
5. Phase 1: 永続化を 1 か所にまとめる。InsertIntoPoints.go の InsertIntoPoints、DeleteFromPoints、InsertIntoTimeTable を個別分岐から直接叩くのではなく、状態遷移結果を受けて保存する形に寄せる。
6. Phase 1: LastScore の意味を明文化する。SRGSE5M.go の LastScore は、直近値、同値継続回数、区間累積、区間開始/終了、配信再検出、オフライン継続回数を持つ状態キャッシュとして扱う。Dup は同一値連続回数、Sum0 は区間累積増分、Tstart0/Tend は区間境界、Tstart1 は配信開始時刻の再確定用、Continued は配信再開回数、NoOffline はオフライン継続回数として固定する。
7. Phase 2: 配信状態取得を bchistory ベースへ切り替える。GetIsOnliveByAPI の直接呼び出しをやめ、roomid と timestamp から有効な配信枠を返す読み取り関数を用意する。startedat は確定値、endedatlb と endedatub は終了時刻の下限/上限として扱う。
8. Phase 2: 配信枠ポイント算出を bchistory に寄せる。Tstart0 は bchistory.startedat を正とし、終了側は endedatlb/endedatub による区間として扱う。
9. Phase 2: 配信枠の終了とポイント加算の終了を分離する。配信終了後も 5〜10 分ポイントが伸びうるため、配信枠は終了済みだがポイントは未確定という中間状態を設計に入れる。
10. Phase 2: timetable 登録条件を整理する。bchistory の終了記録だけで即時確定せず、ポイント増分が落ち着いたことを条件に集計を閉じる。
11. Phase 3: 文字列整形を隔離する。pstatus、ptime、qstatus、qtime は当面互換維持のため残すが、状態遷移の中心ロジックから切り離し、将来的に Web 側へ移せる形にする。
12. Phase 3: テスト戦略を追加する。新規配信開始、配信中のポイント増加、配信終了後も加点が続くケース、配信再開、オフライン継続、3 連続重複削除、順位だけ変化、イベント終了時処理をテーブル駆動で固定する。

### Proposed function split

1. GetPointsAll(client, idList, gschedule, cntrblist) はオーケストレーターとして残す。責務は、イベント終了判定、収集関数呼び出し、ルームごとの状態遷移適用、保存結果の確定、eventuser 補完までに限定する。
2. collectPointSnapshots(client, gschedule, idList, cntrblist) ([]PointSnapshot, []TrackedRoom, bool, error) を新設する。役割は、ランキング系 API と個別 point API を吸収し、最終的に userno ごとの point/rank/gap/eventid を返すこと。戻り値の bool は eventIsOver を表す。
3. loadRankingCandidates(client, gschedule, knownRooms) (rankingResult, error) を新設する。GetEventsRankingByApi、GetEventRankingByApi、GetEventQuestRoomsByApi、GetEventBlockRanking の分岐を閉じ込める。
4. mergeTrackedRooms(existingIDs, cntrblist, rankingResult) ([]TrackedRoom, error) を新設する。eventuser 起点の対象ルームと、ランキングから増えた新規ルームを 1 つの配列に正規化する。
5. resolveLiveWindow(roomID, timestamp, useContribution, fallbackClient) (LiveWindow, error) を新設する。Phase 1 では GetIsOnliveByAPI をラップし、Phase 2 では bchistory 読み取りへ置き換える。LiveWindow は startedat、endedatlb、endedatub、isLive、isKnown を持つ想定とする。
6. buildRoomContext(gschedule, trackedRoom, snapshot, liveWindow, thresholdPoint, timestamp) (RoomContext, error) を新設する。状態遷移関数へ渡す入力を 1 箇所で組み立てる。
7. applyPointTransition(prev *LastScore, ctx RoomContext) (TransitionResult, error) を新設する。GetPointsAll の中で最も複雑な A-1-1、A-1-2、A-2、新規作成をここへ集約する。TransitionResult は nextScore、pointRecord、deleteMiddlePoint、timeTableRecord、needsNewEventUser、pendingSettlement を持つ想定とする。
8. formatLegacyPointStatus(result *TransitionResult) を新設する。pstatus、ptime、qstatus、qtime の組み立てを状態遷移の本体から分ける。Phase 1 では points 保存互換のため必須、Phase 3 で置き換え候補とする。
9. persistTransition(tx, transitionResult) error を新設する。InsertIntoPoints、DeleteFromPoints、InsertIntoTimeTable の実行順序を 1 か所にまとめる。
10. finalizeEventUserUpdates(client, eventid, gschedule, newUsers, timestamp) error を新設する。現行の nu 処理と UpinsEventuser 呼び出しを隔離する。
11. handleEventEnd(gschedule, trackedRooms, timestamp) error を新設する。イベント終了時の DontGetScore 更新、scoremap 参照、必要に応じた timetable 追記を専用化する。
12. computeThresholdPoint(gschedule, timestamp) int を新設する。現行の thpoint 算出を分離し、収集・遷移ロジックから切り離す。
13. makeRoomKey(userno, eventid) string を新設する。scoremap キー生成の重複を除去する。
14. Phase 2 で pendingSettlement を導入する。これは、配信枠は終わっているがポイント加算はまだ継続しうる状態を表し、bchistory の endedat 区間と最終増分時刻の両方を見て確定判定するためのフラグとする。

### Verification

1. 既存ログと同一イベントで比較し、points 件数、重複削除位置、eventuser の最終 point/rank、timetable 作成件数が一致することを確認する。
2. 状態遷移関数を切り出した後、少なくとも 8 系統のテーブル駆動テストを固定する。
3. bchistory 切り替え後、従来の API ベース結果と比較し、startedat と配信枠切れ目の差異を確認する。
4. 配信終了後 5〜10 分程度ポイントが伸びるケースを再現し、配信終了直後に timetable を閉じないことを確認する。
5. イベント終了直前・直後で実行し、DontGetScore 更新、CopyScore への遷移、timetable 追記が従来どおり成立することを確認する。

### Decisions

- 第 1 段階では pstatus、ptime、qstatus、qtime を維持する。
- 優先順位は、見通しの改善、責務の明確化、API 変更耐性であり、性能改善は副次効果として扱う。
- bchistory は、startedat は確定値、endedatlb と endedatub は終了時刻の区間値として扱う。
- 配信終了の代表値は endedatlb と endedatub の平均、誤差はその差の 1/2 とする。
- 配信終了とポイント確定は別概念として扱い、ポイント確定待ちの中間状態を持たせる。
- Web 側で文字列表現を再構成する最終形はスコープに含むが、第 1 段階の実装スコープには含めない。

### Further considerations

1. 終了時刻を単一値に潰し込みすぎず、設計上は終了区間を残した方が後続の精度改善に耐えやすい。
2. 配信終了後の加点をいつ最終ポイントとみなすかは、固定猶予時間だけでなく増分停止確認も条件にした方が安全である。
3. block_id=0 の rank 補完は既知の不安定領域なので、GetPointsAll 分割とは別タスクに隔離した方がよい。