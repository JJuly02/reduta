#!/usr/bin/env bash
# End-to-end smoke test for the Reduta M1 core, run against a live stack
# (docker compose up). Exercises auth, events, teams, challenges, submissions,
# scoring/first-blood, scoreboard ranking, rate limiting and authz.
#
#   BASE=http://localhost:8080 bash scripts/smoke.sh
set -u

BASE="${BASE:-http://localhost:8080}"
ADMIN_EMAIL="${REDUTA_BOOTSTRAP_ADMIN_EMAIL:-admin@reduta.local}"
ADMIN_PASS="${REDUTA_BOOTSTRAP_ADMIN_PASSWORD:-admin-dev-password}"
TMP="$(mktemp -d)"; trap 'rm -rf "$TMP"' EXIT
PASS=0; FAIL=0
RID="$(date +%s)-$$-${RANDOM}"  # unique per run (persistent DB safe)
RED="Red-$RID"; BLUE="Blue-$RID"

jget(){ python3 -c 'import sys,json
d=json.load(sys.stdin)
for k in sys.argv[1].split("."):
    if k=="":continue
    d=d[int(k)] if k.isdigit() else d[k]
print(d)' "$1" 2>/dev/null; }

# req METHOD PATH JAR [DATA] -> sets $CODE and $BODY
req(){
  local m="$1" p="$2" jar="$3" data="${4:-}"
  local args=(-s -w $'\n%{http_code}' -X "$m" -b "$jar" -c "$jar" -H 'Content-Type: application/json')
  [ -n "$data" ] && args+=(-d "$data")
  local out; out="$(curl "${args[@]}" "$BASE$p")"
  CODE="$(printf '%s' "$out" | tail -n1)"
  BODY="$(printf '%s' "$out" | sed '$d')"
}

ok(){   PASS=$((PASS+1)); printf '  \033[32mPASS\033[0m %s\n' "$1"; }
bad(){  FAIL=$((FAIL+1)); printf '  \033[31mFAIL\033[0m %s\n' "$1"; [ -n "${2:-}" ] && printf '        %s\n' "$2"; }
expect_code(){ [ "$CODE" = "$2" ] && ok "$1 ($CODE)" || bad "$1" "want $2 got $CODE: $BODY"; }
expect_eq(){ [ "$2" = "$3" ] && ok "$1 ($2)" || bad "$1" "want $3 got $2"; }

echo "== Reduta M1 smoke @ $BASE =="

# 1. health
req GET /healthz "$TMP/anon"; expect_code "health" 200

# 2. admin login
req POST /api/v1/auth/login "$TMP/admin" "{\"email\":\"$ADMIN_EMAIL\",\"password\":\"$ADMIN_PASS\"}"
expect_code "admin login" 200
ROLE="$(printf '%s' "$BODY" | jget role)"; expect_eq "admin role" "$ROLE" "owner"

# 3. create event (first blood bonus 50)
SLUG="smoke-$RANDOM"
req POST /api/v1/events "$TMP/admin" "{\"slug\":\"$SLUG\",\"name\":\"Smoke Cup\",\"first_blood_bonus\":50}"
expect_code "create event" 201
EV="$(printf '%s' "$BODY" | jget id)"
req PATCH "/api/v1/events/$EV/state" "$TMP/admin" '{"state":"running"}'; expect_code "event -> running" 200

# 4. challenges: two published + one draft
req POST "/api/v1/events/$EV/challenges" "$TMP/admin" '{"title":"Alpha","category":"web","scoring":{"type":"static","points":100},"state":"published","flags":[{"value":"flag{smoke_alpha}"}]}'
expect_code "create chA" 201; CHA="$(printf '%s' "$BODY" | jget id)"
req POST "/api/v1/events/$EV/challenges" "$TMP/admin" '{"title":"Bravo","category":"pwn","scoring":{"type":"static","points":200},"state":"published","flags":[{"value":"FLAG{smoke_bravo}","case_sensitive":false}]}'
expect_code "create chB" 201; CHB="$(printf '%s' "$BODY" | jget id)"
req POST "/api/v1/events/$EV/challenges" "$TMP/admin" '{"title":"Secret","category":"misc","state":"draft","flags":[{"value":"flag{hidden}"}]}'
expect_code "create draft" 201; CHD="$(printf '%s' "$BODY" | jget id)"

# 5. authz: anon /me -> 401
req GET /api/v1/auth/me "$TMP/anon"; expect_code "anon /me -> 401" 401

# 6. register two players, each with a team
req POST /api/v1/auth/register "$TMP/p1" "{\"email\":\"p1-$RID@smoke.local\",\"display_name\":\"P1\",\"password\":\"password123\"}"
expect_code "register p1" 201
req POST /api/v1/auth/register "$TMP/p2" "{\"email\":\"p2-$RID@smoke.local\",\"display_name\":\"P2\",\"password\":\"password123\"}"
expect_code "register p2" 201

# 7. player cannot create events (admin only)
req POST /api/v1/events "$TMP/p1" '{"slug":"nope","name":"nope"}'; expect_code "player create event -> 403" 403

req POST /api/v1/teams "$TMP/p1" "{\"name\":\"$RED\"}"; expect_code "p1 create global team" 201
T1="$(printf '%s' "$BODY" | jget id)"; REDCODE="$(printf '%s' "$BODY" | jget invite_code)"
req POST "/api/v1/events/$EV/event-teams" "$TMP/admin" "{\"team_id\":\"$T1\"}"; expect_code "admin assigns team1 to event" 200
req POST /api/v1/teams "$TMP/p2" "{\"name\":\"$BLUE\"}"; expect_code "p2 create global team" 201
T2="$(printf '%s' "$BODY" | jget id)"
req POST "/api/v1/events/$EV/event-teams" "$TMP/admin" "{\"team_id\":\"$T2\"}"; expect_code "admin assigns team2 to event" 200
req POST /api/v1/auth/register "$TMP/p1b" "{\"email\":\"p1b-$RID@smoke.local\",\"display_name\":\"P1b\",\"password\":\"password123\"}"; expect_code "register p1b" 201
req POST /api/v1/teams/join "$TMP/p1b" "{\"invite_code\":\"$REDCODE\"}"; expect_code "p1b joins team by invite code" 200

# 8. player challenge list shows only published (2), never flags
req GET "/api/v1/events/$EV/challenges" "$TMP/p1"; expect_code "list challenges" 200
N="$(printf '%s' "$BODY" | jget challenges | tr ',' '\n' | grep -c id || true)"
CNT="$(printf '%s' "$BODY" | python3 -c 'import sys,json;print(len(json.load(sys.stdin)["challenges"]))')"
expect_eq "player sees 2 published" "$CNT" "2"
if printf '%s' "$BODY" | grep -qi 'flag{'; then bad "no flags in listing" "found flag literal"; else ok "no flags in listing"; fi

# 9. player cannot fetch draft challenge
req GET "/api/v1/events/$EV/challenges/$CHD" "$TMP/p1"; expect_code "draft hidden from player -> 404" 404

# 10. submissions + scoring
req POST "/api/v1/events/$EV/challenges/$CHA/submit" "$TMP/p1" '{"flag":"flag{wrong}"}'
expect_code "p1 wrong submit" 200
expect_eq "p1 wrong -> correct:false" "$(printf '%s' "$BODY" | jget correct)" "False"

req POST "/api/v1/events/$EV/challenges/$CHA/submit" "$TMP/p1" '{"flag":"flag{smoke_alpha}"}'
expect_code "p1 solve chA" 200
expect_eq "p1 chA correct" "$(printf '%s' "$BODY" | jget correct)" "True"
expect_eq "p1 chA first blood" "$(printf '%s' "$BODY" | jget first_blood)" "True"
expect_eq "p1 chA points (100+50 fb)" "$(printf '%s' "$BODY" | jget points)" "150"

req POST "/api/v1/events/$EV/challenges/$CHA/submit" "$TMP/p2" '{"flag":"flag{smoke_alpha}"}'
expect_code "p2 solve chA" 200
expect_eq "p2 chA not first blood" "$(printf '%s' "$BODY" | jget first_blood)" "False"
expect_eq "p2 chA points (100)" "$(printf '%s' "$BODY" | jget points)" "100"

req POST "/api/v1/events/$EV/challenges/$CHA/submit" "$TMP/p1" '{"flag":"flag{smoke_alpha}"}'
expect_eq "p1 re-solve chA -> already_solved" "$(printf '%s' "$BODY" | jget already_solved)" "True"

# case-insensitive flag on chB
req POST "/api/v1/events/$EV/challenges/$CHB/submit" "$TMP/p2" '{"flag":"flag{SMOKE_BRAVO}"}'
expect_eq "p2 chB case-insensitive correct" "$(printf '%s' "$BODY" | jget correct)" "True"
expect_eq "p2 chB first blood (first solver of chB)" "$(printf '%s' "$BODY" | jget first_blood)" "True"
expect_eq "p2 chB points (200+50 fb)" "$(printf '%s' "$BODY" | jget points)" "250"

# 11. scoreboard: Blue (p2) 300 leads Red (p1) 150
req GET "/api/v1/events/$EV/scoreboard" "$TMP/anon"; expect_code "scoreboard" 200
expect_eq "rank1 is team2"    "$(printf '%s' "$BODY" | jget entries.0.team_id)" "$T2"
expect_eq "rank1 points 350"  "$(printf '%s' "$BODY" | jget entries.0.points)" "350"
expect_eq "rank2 is team1"    "$(printf '%s' "$BODY" | jget entries.1.team_id)" "$T1"
expect_eq "rank2 points 150"  "$(printf '%s' "$BODY" | jget entries.1.points)" "150"

# 12. submit without a team -> 403 (register a teamless player)
req POST /api/v1/auth/register "$TMP/p3" "{\"email\":\"p3-$RID@smoke.local\",\"display_name\":\"P3\",\"password\":\"password123\"}"
req POST "/api/v1/events/$EV/challenges/$CHA/submit" "$TMP/p3" '{"flag":"x"}'
expect_code "teamless submit -> 403" 403

# 13. rate limit: burst wrong submits from p1 -> expect a 429
RL=""
for i in $(seq 1 16); do
  req POST "/api/v1/events/$EV/challenges/$CHB/submit" "$TMP/p1" '{"flag":"flag{spam}"}'
  [ "$CODE" = "429" ] && { RL="yes"; break; }
done
[ -n "$RL" ] && ok "rate limit triggers 429" || bad "rate limit triggers 429" "no 429 in 16 submits"

# ---- M2: library / clone / revisions / embed / blocks / bulk / undo ----
echo "-- M2 library & bulk --"
LSLUG="lib-$RANDOM"
req POST /api/v1/challenges "$TMP/admin" "{\"slug\":\"$LSLUG\",\"title\":\"Lib Web\",\"category\":\"web\",\"tags\":[\"web\",\"beginner\"],\"description_md\":\"lib desc\",\"scoring\":{\"type\":\"static\",\"points\":300},\"flags\":[{\"value\":\"flag{lib_one}\"}]}"
expect_code "create library challenge" 201; LC="$(printf '%s' "$BODY" | jget id)"
req GET "/api/v1/challenges/$LC" "$TMP/admin"; expect_code "get library challenge" 200
if printf '%s' "$BODY" | grep -qi 'flag{'; then bad "library get hides flags" "flag literal present"; else ok "library get hides flags"; fi
req POST "/api/v1/challenges/$LC/revisions" "$TMP/admin" '{"description_md":"v2","scoring":{"type":"static","points":350},"flags":[{"value":"flag{lib_v2}"}]}'
expect_code "new revision" 201; expect_eq "revision bumped to 2" "$(printf '%s' "$BODY" | jget rev)" "2"
req POST "/api/v1/challenges/$LC/clone" "$TMP/admin" "{\"slug\":\"$LSLUG-copy\"}"; expect_code "clone challenge" 201

req POST "/api/v1/events/$EV/challenges/from-library" "$TMP/admin" "{\"challenge_id\":\"$LC\"}"
expect_code "embed from library" 201; EMB="$(printf '%s' "$BODY" | jget id)"

req POST "/api/v1/events/$EV/blocks" "$TMP/admin" '{"name":"Day 1","position":1}'
expect_code "create block" 201; BLK="$(printf '%s' "$BODY" | jget id)"
req POST "/api/v1/events/$EV/challenges/bulk" "$TMP/admin" "{\"selector\":{\"mode\":\"ids\",\"ids\":[\"$EMB\"]},\"action\":{\"type\":\"assign_block\",\"params\":{\"block_id\":\"$BLK\"}}}"
expect_code "bulk assign_block" 200; expect_eq "assign affected 1" "$(printf '%s' "$BODY" | jget affected)" "1"

req POST "/api/v1/events/$EV/challenges/bulk" "$TMP/admin" '{"selector":{"mode":"filter","filter":{"state":"draft"}},"action":{"type":"publish"}}'
expect_code "bulk publish by filter" 200; JOB="$(printf '%s' "$BODY" | jget job_id)"
AFF="$(printf '%s' "$BODY" | jget affected)"; { [ -n "$AFF" ] && [ "$AFF" -ge 1 ]; } && ok "bulk publish affected>=1 ($AFF)" || bad "bulk publish affected" "$AFF"

req GET "/api/v1/events/$EV/challenges" "$TMP/p1"
CNT2="$(printf '%s' "$BODY" | python3 -c 'import sys,json;print(len(json.load(sys.stdin)["challenges"]))')"
{ [ -n "$CNT2" ] && [ "$CNT2" -ge 3 ]; } && ok "player sees embedded published ($CNT2)" || bad "player sees embedded" "got $CNT2"

req POST "/api/v1/bulk-jobs/$JOB/undo" "$TMP/admin"; expect_code "bulk undo" 200
req GET "/api/v1/events/$EV/blocks" "$TMP/anon"; expect_code "list blocks (public)" 200

req POST "/api/v1/events/$EV/saved-views" "$TMP/admin" '{"name":"drafts","filter":{"state":"draft"}}'; expect_code "create saved view" 201
req GET "/api/v1/events/$EV/saved-views" "$TMP/admin"; expect_code "list saved views" 200


# ---- M3: schedules (RRULE) + unlock rules ----
echo "-- M3 schedule & unlock --"
# gated challenge: closed by schedule (past window) + unlock requiring solved chA
req POST "/api/v1/events/$EV/challenges" "$TMP/admin" '{"title":"Gated","category":"misc","scoring":{"type":"static","points":500},"state":"published","flags":[{"value":"flag{gated}"}]}'
expect_code "create gated challenge" 201; CHG="$(printf '%s' "$BODY" | jget id)"

# set a schedule window entirely in the past -> closed (locked behavior so it stays visible)
req PATCH "/api/v1/events/$EV/challenges/$CHG/schedule" "$TMP/admin" '{"schedule":{"windows":[{"opens_at":"2020-01-01T00:00:00Z","duration":"PT1H"}],"closed_behavior":"locked"}}'
expect_code "set past schedule" 200
# a fresh teamless-but-on-team player: p1 is on team Red and already solved chA earlier
req POST "/api/v1/events/$EV/challenges/$CHG/submit" "$TMP/p2" '{"flag":"flag{gated}"}'
expect_code "submit to scheduled-closed -> 403" 403

# open the schedule (wide window) but require unlock: must have >= 999 points (p2 has 350) -> locked
req PATCH "/api/v1/events/$EV/challenges/$CHG/schedule" "$TMP/admin" '{"schedule":{"windows":[{"opens_at":"2020-01-01T00:00:00Z","duration":"P3650D"}]}}'
expect_code "set open schedule" 200
req PATCH "/api/v1/events/$EV/challenges/$CHG/unlock" "$TMP/admin" '{"unlock_rule":{"team_points_gte":999}}'
expect_code "set unlock rule" 200
req POST "/api/v1/events/$EV/challenges/$CHG/submit" "$TMP/p2" '{"flag":"flag{gated}"}'
expect_code "submit to unlock-locked -> 403" 403

# listing marks it locked for the player
req GET "/api/v1/events/$EV/challenges" "$TMP/p2"
GST="$(printf '%s' "$BODY" | python3 -c 'import sys,json
d=json.load(sys.stdin)["challenges"]
print(next((c["status"] for c in d if c["id"]==sys.argv[1]), "missing"))' "$CHG")"
expect_eq "gated challenge shows locked" "$GST" "locked"

# relax unlock (>=100, p2 has 350) -> now solvable
req PATCH "/api/v1/events/$EV/challenges/$CHG/unlock" "$TMP/admin" '{"unlock_rule":{"team_points_gte":100}}'
expect_code "relax unlock rule" 200
req POST "/api/v1/events/$EV/challenges/$CHG/submit" "$TMP/p2" '{"flag":"flag{gated}"}'
expect_eq "unlocked submit correct" "$(printf '%s' "$BODY" | jget correct)" "True"


# ---- M4: export / import (native + CTFd) ----
echo "-- M4 import/export --"
req GET "/api/v1/events/$EV/export" "$TMP/admin"; expect_code "export event" 200
EXPN="$(printf '%s' "$BODY" | python3 -c 'import sys,json;print(len(json.load(sys.stdin)["challenges"]))')"
{ [ -n "$EXPN" ] && [ "$EXPN" -ge 2 ]; } && ok "export has challenges ($EXPN)" || bad "export challenges" "$EXPN"

req POST /api/v1/events "$TMP/admin" "{\"slug\":\"imp-$RID\",\"name\":\"Import Target\",\"first_blood_bonus\":0}"
expect_code "create import target" 201; EV2="$(printf '%s' "$BODY" | jget id)"
req PATCH "/api/v1/events/$EV2/state" "$TMP/admin" '{"state":"running"}' >/dev/null 2>&1

IMPBODY='{"challenges":[{"title":"Imported One","category":"web","state":"published","scoring":{"type":"static","points":120},"flags":[{"value":"flag{imp1}"}]},{"title":"Imported Two","category":"pwn","state":"draft","scoring":{"type":"static","points":300},"flags":[{"value":"flag{imp2}"}]}]}'
req POST "/api/v1/events/$EV2/import?dry_run=true" "$TMP/admin" "$IMPBODY"
expect_code "import dry-run" 200
expect_eq "dry-run created=2" "$(printf '%s' "$BODY" | jget plan.created)" "2"
req GET "/api/v1/events/$EV2/challenges" "$TMP/admin"
expect_eq "dry-run created nothing" "$(printf '%s' "$BODY" | python3 -c 'import sys,json;print(len(json.load(sys.stdin)["challenges"]))')" "0"
req POST "/api/v1/events/$EV2/import" "$TMP/admin" "$IMPBODY"
expect_eq "import created 2" "$(printf '%s' "$BODY" | jget plan.created)" "2"
req POST "/api/v1/events/$EV2/import" "$TMP/admin" "$IMPBODY"
expect_eq "re-import updated 2 (idempotent)" "$(printf '%s' "$BODY" | jget plan.updated)" "2"

CTFD='{"challenges":[{"name":"CTFd Chal","category":"misc","value":75,"state":"visible","flags":["flag{ctfd}"]}]}'
req POST "/api/v1/events/$EV2/import?format=ctfd" "$TMP/admin" "$CTFD"
CTC="$(printf '%s' "$BODY" | jget plan.created)"
{ [ -n "$CTC" ] && [ "$CTC" -ge 1 ]; } && ok "ctfd import created>=1 ($CTC)" || bad "ctfd import" "$CTC"

# imported flag works end-to-end
req POST /api/v1/auth/register "$TMP/pi" "{\"email\":\"pi-$RID@smoke.local\",\"display_name\":\"PI\",\"password\":\"password123\"}"; expect_code "register importer player" 201
req POST /api/v1/teams "$TMP/pi" "{\"name\":\"Importers-$RID\"}"; expect_code "pi create team" 201
PIT="$(printf '%s' "$BODY" | jget id)"
req POST "/api/v1/events/$EV2/event-teams" "$TMP/admin" "{\"team_id\":\"$PIT\"}"; expect_code "assign pi team to EV2" 200
req GET "/api/v1/events/$EV2/challenges" "$TMP/pi"
IMPEC="$(printf '%s' "$BODY" | python3 -c 'import sys,json
d=json.load(sys.stdin)["challenges"]
print(next((c["id"] for c in d if c["title"]=="Imported One"),""))')"
req POST "/api/v1/events/$EV2/challenges/$IMPEC/submit" "$TMP/pi" '{"flag":"flag{imp1}"}'
expect_eq "solve imported challenge" "$(printf '%s' "$BODY" | jget correct)" "True"

# ---- M4: file attachments carried in the import JSON ----
echo "-- M4 file attachments --"
FDATA="$(printf 'reduta-file-ok' | base64 | tr -d '\n')"
FBODY="{\"challenges\":[{\"title\":\"WithFile\",\"category\":\"misc\",\"state\":\"published\",\"scoring\":{\"type\":\"static\",\"points\":50},\"flags\":[{\"value\":\"flag{file}\"}],\"files\":[{\"name\":\"note.txt\",\"content_type\":\"text/plain\",\"data\":\"$FDATA\"}]}]}"
req POST "/api/v1/events/$EV2/import" "$TMP/admin" "$FBODY"
expect_code "import challenge with file" 200
req GET "/api/v1/events/$EV2/challenges" "$TMP/admin"
WFEC="$(printf '%s' "$BODY" | python3 -c 'import sys,json
d=json.load(sys.stdin)["challenges"]
print(next((c["id"] for c in d if c["title"]=="WithFile"),""))')"
req GET "/api/v1/events/$EV2/challenges/$WFEC" "$TMP/admin"
expect_eq "challenge exposes 1 file" "$(printf '%s' "$BODY" | python3 -c 'import sys,json;print(len(json.load(sys.stdin).get("files",[])))')" "1"
WFID="$(printf '%s' "$BODY" | python3 -c 'import sys,json;fs=json.load(sys.stdin)["files"];print(fs[0]["id"] if fs else "")')"
DL="$(curl -s "$BASE/api/v1/events/$EV2/challenges/$WFEC/files/$WFID")"
expect_eq "download returns file bytes" "$DL" "reduta-file-ok"
req GET "/api/v1/events/$EV2/export" "$TMP/admin"
expect_eq "export round-trips file bytes" "$(printf '%s' "$BODY" | python3 -c 'import sys,json,base64
d=json.load(sys.stdin)
ch=next((c for c in d["challenges"] if c["title"]=="WithFile"),None)
print(base64.b64decode(ch["files"][0]["data"]).decode() if ch and ch.get("files") else "")')" "reduta-file-ok"

# ---- M5: realtime + cache ----
echo "-- M5 realtime & cache --"
WSC=$(curl -s -m3 -o /dev/null -w '%{http_code}' -H "Connection: Upgrade" -H "Upgrade: websocket" -H "Sec-WebSocket-Version: 13" -H "Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==" "$BASE/ws?event=$EV")
[ "$WSC" = "101" ] && ok "websocket handshake 101" || bad "websocket handshake" "got $WSC"
curl -s -o /dev/null "$BASE/api/v1/events/$EV/scoreboard"
HDR=$(curl -s -D - -o /dev/null "$BASE/api/v1/events/$EV/scoreboard" | tr -d '\r' | grep -i "^x-cache:")
echo "$HDR" | grep -qi hit && ok "scoreboard redis cache hit" || bad "scoreboard cache" "hdr=$HDR"


# ---- M7: plugin API (registration, idempotent awards, teams) ----
echo "-- M7 plugin API --"
req POST /api/v1/plugins "$TMP/admin" "{\"id\":\"koth-$RID\",\"name\":\"KotH\",\"capabilities\":[\"score_awards\"],\"events\":[\"solve.created\",\"tick.minute\"]}"
expect_code "register plugin" 201
PTOK="$(printf '%s' "$BODY" | jget token)"
[ -n "$PTOK" ] && ok "plugin token issued" || bad "plugin token" "empty"

pluginreq(){ local m="$1" p="$2" data="${3:-}"; local a=(-s -w $'\n%{http_code}' -X "$m" -H "Authorization: Bearer $PTOK" -H 'Content-Type: application/json'); [ -n "$data" ] && a+=(-d "$data"); local o; o="$(curl "${a[@]}" "$BASE$p")"; CODE="$(printf '%s' "$o"|tail -n1)"; BODY="$(printf '%s' "$o"|sed '$d')"; }

pluginreq POST "/api/v1/plugin/v1/awards" "{\"event_id\":\"$EV\",\"team_id\":\"$T1\",\"points\":5,\"ref_id\":\"tick:one\",\"reason\":\"koth tick\"}"
expect_code "plugin award" 200
expect_eq "award applied" "$(printf '%s' "$BODY" | jget applied)" "True"
pluginreq POST "/api/v1/plugin/v1/awards" "{\"event_id\":\"$EV\",\"team_id\":\"$T1\",\"points\":5,\"ref_id\":\"tick:one\",\"reason\":\"koth tick\"}"
expect_eq "award idempotent (second no-op)" "$(printf '%s' "$BODY" | jget applied)" "False"

# scoreboard: Red (T1) went 150 -> 155
req GET "/api/v1/events/$EV/scoreboard" "$TMP/anon"
REDPTS="$(printf '%s' "$BODY" | python3 -c 'import sys,json
d=json.load(sys.stdin)["entries"]
print(next((e["points"] for e in d if e["team_id"]==sys.argv[1]),""))' "$T1")"
expect_eq "award reflected on scoreboard (team1 155)" "$REDPTS" "155"

pluginreq GET "/api/v1/plugin/v1/events/$EV/teams"
expect_code "plugin lists teams" 200

pluginreq DELETE "/api/v1/plugin/v1/awards/tick:one?event_id=$EV"
expect_code "plugin delete award" 200
expect_eq "award removed" "$(printf '%s' "$BODY" | jget removed)" "True"
req GET "/api/v1/events/$EV/scoreboard" "$TMP/anon"
REDPTS2="$(printf '%s' "$BODY" | python3 -c 'import sys,json
d=json.load(sys.stdin)["entries"]
print(next((e["points"] for e in d if e["team_id"]==sys.argv[1]),""))' "$T1")"
expect_eq "award reversal (team1 back to 150)" "$REDPTS2" "150"

pluginreq POST "/api/v1/plugin/v1/announcements" "{\"event_id\":\"$EV\",\"text\":\"round starting\"}"
expect_code "plugin announcement" 202


# ---- M8: per-team instances + dynamic flags ----
echo "-- M8 instances --"
req POST "/api/v1/events/$EV/challenges" "$TMP/admin" '{"title":"Instanced","category":"pwn","scoring":{"type":"static","points":400},"state":"published","flags":[{"value":"flag{static-unused}"}]}'
expect_code "create instanced challenge" 201; CHI="$(printf '%s' "$BODY" | jget id)"
req PATCH "/api/v1/events/$EV/challenges/$CHI/instance-spec" "$TMP/admin" '{"instance_spec":{"image":"registry.local/pwn:1","port":1337,"ttl":"PT45M","env":{"FLAG":"{{team_flag}}"}}}'
expect_code "set instance spec" 200
req POST /api/v1/auth/register "$TMP/p8" "{\"email\":\"p8-$RID@smoke.local\",\"display_name\":\"P8\",\"password\":\"password123\"}"; expect_code "register p8" 201
req POST /api/v1/teams "$TMP/p8" "{\"name\":\"Instancers-$RID\"}"; expect_code "p8 team" 201
P8T="$(printf '%s' "$BODY" | jget id)"
req POST "/api/v1/events/$EV/event-teams" "$TMP/admin" "{\"team_id\":\"$P8T\"}"; expect_code "assign p8 team to event" 200

req POST "/api/v1/events/$EV/challenges/$CHI/instance" "$TMP/p8" ""
expect_code "create instance" 201
IHOST="$(printf '%s' "$BODY" | jget instance.host)"; IFLAG="$(printf '%s' "$BODY" | jget flag)"
[ -n "$IHOST" ] && ok "instance has host ($IHOST)" || bad "instance host" "empty"
printf '%s' "$IFLAG" | grep -q '^flag{' && ok "per-team flag issued" || bad "team flag" "$IFLAG"
req GET "/api/v1/events/$EV/challenges/$CHI/instance" "$TMP/p8" ""; expect_code "get instance" 200
req POST "/api/v1/events/$EV/challenges/$CHI/submit" "$TMP/p8" "{\"flag\":\"$IFLAG\"}"
expect_eq "team-flag submit correct" "$(printf '%s' "$BODY" | jget correct)" "True"
req POST "/api/v1/events/$EV/challenges/$CHI/submit" "$TMP/p8" '{"flag":"flag{not-my-team}"}'
expect_eq "foreign flag rejected" "$(printf '%s' "$BODY" | jget correct)" "False"
req POST "/api/v1/events/$EV/challenges/$CHI/instance/extend" "$TMP/p8" ""
expect_code "extend instance" 200
expect_eq "extend count 1" "$(printf '%s' "$BODY" | jget instance.extends)" "1"
req DELETE "/api/v1/events/$EV/challenges/$CHI/instance" "$TMP/p8" ""
expect_code "destroy instance" 200
expect_eq "destroyed true" "$(printf '%s' "$BODY" | jget destroyed)" "True"
req GET "/api/v1/events/$EV/challenges/$CHI/instance" "$TMP/p8" ""; expect_code "instance gone -> 404" 404


echo
echo "== $PASS passed, $FAIL failed =="
[ "$FAIL" -eq 0 ]
