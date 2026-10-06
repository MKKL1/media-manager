#!/bin/sh
set -eu
api=${API:-http://localhost:3000}/api/v1

auth="Authorization: Bearer ${TOKEN:?set TOKEN}"
post() { curl -sSf -m 300 -XPOST -H "$auth" -H 'Content-Type: application/json' "$api/$1" -d "$2"; echo; }

for tmdb_and_tvdb in 'tv:127532 series:389597' \
                     'tv:209867 series:424536' \
                     'tv:120089 series:405920'; do
	set -- $tmdb_and_tvdb
	tmdb=$1 tvdb=$2
	post providers/tmdb/fetch "{\"ref\":\"$tmdb\"}"
	post providers/tvdb/fetch "{\"ref\":\"$tvdb\"}"
done
post providers/anime-lists/fetch '{}'

for show in tv:127532 tv:209867 tv:120089; do
	post "entries/tmdb:$show/edits" '{"op":"choice","chosen":true}'
done
curl -sSf -H "$auth" "$api/library"; echo
