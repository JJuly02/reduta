#!/usr/bin/env bash
# Seeds a rich demo event ("Demo Cup") so the UI has plenty to look at:
# multiple categories, blocks, tags, a gated challenge, draft/hidden states,
# several teams and time-spread solves for a readable scoreboard chart.
# Intended to run against a fresh database.
#   BASE=http://localhost:8080 bash scripts/seed-demo.sh
set -u
BASE="${BASE:-http://localhost:8080}"
AE="${REDUTA_BOOTSTRAP_ADMIN_EMAIL:-admin@reduta.local}"
AP="${REDUTA_BOOTSTRAP_ADMIN_PASSWORD:-admin-dev-password}"
TMP="$(mktemp -d)"; trap 'rm -rf "$TMP"' EXIT
jget(){ python3 -c 'import sys,json;d=json.load(sys.stdin)
for k in sys.argv[1].split("."):
 d=d[int(k)] if k.isdigit() else d[k]
print(d)' "$1" 2>/dev/null; }
J(){ local m="$1" p="$2" jar="$3" d="${4:-}"; local a=(-s -X "$m" -b "$jar" -c "$jar" -H 'Content-Type: application/json'); [ -n "$d" ] && a+=(-d "$d"); curl "${a[@]}" "$BASE$p"; }

J POST /api/v1/auth/login "$TMP/a" "{\"email\":\"$AE\",\"password\":\"$AP\"}" >/dev/null
EV=$(J POST /api/v1/events "$TMP/a" '{"slug":"reduta-open","name":"Reduta Open 2026","first_blood_bonus":100}' | jget id)
[ -z "$EV" ] && EV=$(J GET /api/v1/events "$TMP/a" | python3 -c 'import sys,json;print(next(e["id"] for e in json.load(sys.stdin)["events"] if e["slug"]=="reduta-open"))')
J PATCH "/api/v1/events/$EV/state" "$TMP/a" '{"state":"running"}' >/dev/null

BWARM=$(J POST "/api/v1/events/$EV/blocks" "$TMP/a" '{"name":"Warm-up","position":1}' | jget id)
BWEB=$(J POST "/api/v1/events/$EV/blocks" "$TMP/a" '{"name":"Web","position":2}' | jget id)
BBIN=$(J POST "/api/v1/events/$EV/blocks" "$TMP/a" '{"name":"Binary","position":3}' | jget id)

mkch(){ J POST "/api/v1/events/$EV/challenges" "$TMP/a" "$1" | jget id; }
tagids(){ J POST "/api/v1/events/$EV/challenges/bulk" "$TMP/a" "{\"selector\":{\"mode\":\"ids\",\"ids\":[\"$1\"]},\"action\":{\"type\":\"add_tags\",\"params\":{\"tags\":$2}}}" >/dev/null; }
blkids(){ J POST "/api/v1/events/$EV/challenges/bulk" "$TMP/a" "{\"selector\":{\"mode\":\"ids\",\"ids\":[\"$1\"]},\"action\":{\"type\":\"assign_block\",\"params\":{\"block_id\":\"$2\"}}}" >/dev/null; }

C_WELCOME=$(mkch '{"title":"Welcome","category":"misc","description_md":"Read the rules and submit the flag below to get on the board.","scoring":{"type":"static","points":50},"state":"published","flags":[{"value":"flag{welcome_to_reduta}"}]}')
C_SANITY=$(mkch  '{"title":"Sanity Check","category":"misc","description_md":"The flag is in the page source.","scoring":{"type":"static","points":50},"state":"published","flags":[{"value":"flag{just_getting_started}"}]}')
C_HDR=$(mkch     '{"title":"Header Games","category":"web","description_md":"Inspect the HTTP response headers carefully.","scoring":{"type":"static","points":150},"state":"published","flags":[{"value":"flag{x_powered_by_you}"}]}')
C_SQLI=$(mkch    '{"title":"Login Bypass","category":"web","description_md":"The login form trusts its input a little too much.","scoring":{"type":"static","points":200},"state":"published","flags":[{"value":"flag{or_1_equals_1}"}]}')
C_JWT=$(mkch     '{"title":"None Shall Pass","category":"web","description_md":"The token algorithm is negotiable.","scoring":{"type":"static","points":250},"state":"published","flags":[{"value":"flag{alg_none_ftw}"}]}')
C_XOR=$(mkch     '{"title":"Repeating Key","category":"crypto","description_md":"A short key, reused forever.","scoring":{"type":"static","points":150},"state":"published","flags":[{"value":"flag{xor_is_not_encryption}"}]}')
C_RSA=$(mkch     '{"title":"Small e","category":"crypto","description_md":"e is very small and there is no padding.","scoring":{"type":"static","points":250},"state":"published","flags":[{"value":"flag{cube_root_attack}"}]}')
C_VAULT=$(mkch   '{"title":"Locked Vault","category":"crypto","description_md":"Unlocks once your team passes 100 points.","scoring":{"type":"static","points":300},"state":"published","flags":[{"value":"flag{after_the_gate}"}]}')
C_HEAP=$(mkch    '{"title":"Heap Feng Shui","category":"pwn","description_md":"Classic tcache poisoning.","scoring":{"type":"static","points":350},"state":"published","flags":[{"value":"flag{tcache_poisoning}"}]}')
C_ROP=$(mkch     '{"title":"Return Trip","category":"pwn","description_md":"No shellcode allowed. Build a chain.","scoring":{"type":"static","points":300},"state":"published","flags":[{"value":"flag{rop_and_roll}"}]}')
C_REV=$(mkch     '{"title":"Crack Me","category":"rev","description_md":"Find the serial the binary accepts.","scoring":{"type":"static","points":200},"state":"published","flags":[{"value":"flag{keygen_time}"}]}')
C_PCAP=$(mkch    '{"title":"On The Wire","category":"forensics","description_md":"Someone leaked the flag over the network.","scoring":{"type":"static","points":150},"state":"published","flags":[{"value":"flag{follow_the_stream}"}]}')
C_DRAFT=$(mkch   '{"title":"Steganography II","category":"forensics","description_md":"Draft, not released yet.","scoring":{"type":"static","points":250},"state":"draft","flags":[{"value":"flag{hidden_in_plain_sight}"}]}')
C_HIDDEN=$(mkch  '{"title":"Prize Round","category":"misc","description_md":"Hidden bonus, revealed later.","scoring":{"type":"static","points":500},"state":"hidden","flags":[{"value":"flag{surprise}"}]}')

blkids "$C_WELCOME" "$BWARM"; blkids "$C_SANITY" "$BWARM"
blkids "$C_HDR" "$BWEB"; blkids "$C_SQLI" "$BWEB"; blkids "$C_JWT" "$BWEB"
blkids "$C_HEAP" "$BBIN"; blkids "$C_ROP" "$BBIN"; blkids "$C_REV" "$BBIN"
tagids "$C_WELCOME" '["beginner"]'
tagids "$C_SQLI" '["sqli","auth"]'
tagids "$C_JWT" '["jwt","auth"]'
tagids "$C_XOR" '["classic"]'
tagids "$C_RSA" '["rsa"]'
tagids "$C_HEAP" '["heap","glibc"]'
tagids "$C_PCAP" '["network"]'
J PATCH "/api/v1/events/$EV/challenges/$C_VAULT/unlock" "$TMP/a" '{"unlock_rule":{"team_points_gte":100}}' >/dev/null

# players: first two keep known passwords so the player view can be explored
newteam(){ # email display -> prints "jar" (session cookie jar), assigns team to event
  local jar="$TMP/$2"
  J POST /api/v1/auth/register "$jar" "{\"email\":\"$1\",\"display_name\":\"$2\",\"password\":\"demo-pass-123\"}" >/dev/null
  J POST /api/v1/auth/login "$jar" "{\"email\":\"$1\",\"password\":\"demo-pass-123\"}" >/dev/null
  local tid; tid=$(J POST /api/v1/teams "$jar" "{\"name\":\"$2\"}" | jget id)
  [ -z "$tid" ] && tid=$(J GET /api/v1/me/team "$jar" | jget team.id)
  J POST "/api/v1/events/$EV/event-teams" "$TMP/a" "{\"team_id\":\"$tid\"}" >/dev/null
  echo "$jar"
}
J1=$(newteam demo1@demo.local Ferretwork)
J2=$(newteam demo2@demo.local Bitlords)
J3=$(newteam demo3@demo.local 0xCafe)
J4=$(newteam demo4@demo.local NullBytes)
J5=$(newteam demo5@demo.local RootRiot)
J6=$(newteam demo6@demo.local SegFaulters)

sub(){ J POST "/api/v1/events/$EV/challenges/$2/submit" "$1" "{\"flag\":\"$3\"}" >/dev/null; sleep "${4:-1}"; }

# Time-spread solves: leaders pull ahead, everyone gets on the board.
sub "$J1" "$C_WELCOME" 'flag{welcome_to_reduta}'
sub "$J2" "$C_WELCOME" 'flag{welcome_to_reduta}'
sub "$J3" "$C_WELCOME" 'flag{welcome_to_reduta}'
sub "$J1" "$C_SANITY"  'flag{just_getting_started}'
sub "$J2" "$C_HDR"     'flag{x_powered_by_you}'
sub "$J4" "$C_WELCOME" 'flag{welcome_to_reduta}'
sub "$J1" "$C_HDR"     'flag{x_powered_by_you}'
sub "$J3" "$C_SANITY"  'flag{just_getting_started}'
sub "$J2" "$C_SQLI"    'flag{or_1_equals_1}'
sub "$J5" "$C_WELCOME" 'flag{welcome_to_reduta}'
sub "$J1" "$C_XOR"     'flag{xor_is_not_encryption}'
sub "$J2" "$C_VAULT"   'flag{after_the_gate}'
sub "$J1" "$C_SQLI"    'flag{or_1_equals_1}'
sub "$J3" "$C_HDR"     'flag{x_powered_by_you}'
sub "$J6" "$C_WELCOME" 'flag{welcome_to_reduta}'
sub "$J1" "$C_REV"     'flag{keygen_time}'
sub "$J2" "$C_RSA"     'flag{cube_root_attack}'
sub "$J1" "$C_HEAP"    'flag{tcache_poisoning}'
sub "$J4" "$C_SANITY"  'flag{just_getting_started}'
sub "$J3" "$C_PCAP"    'flag{follow_the_stream}'
sub "$J1" "$C_JWT"     'flag{alg_none_ftw}' 0

echo "Seeded Reduta Open 2026 (event $EV)."
echo "Players: demo1@demo.local .. demo6@demo.local, password demo-pass-123"
