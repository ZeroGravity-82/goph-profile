#!/usr/bin/env bash

# Выполняет короткую smoke-проверку наблюдаемости на локальном стеке: проходит пользовательский сценарий с аватаркой,
# после которого в UI должны появиться логи, метрики и трассы сервера и воркера.

set -Eeuo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
base_url="${GOPH_PROFILE_OBSERVABILITY_BASE_URL:-https://localhost:3202}"
tmp_dir="$(mktemp -d /tmp/goph-profile-observability.XXXXXX)"

cleanup() {
	rm -rf "$tmp_dir"
}
trap cleanup EXIT

require_command() {
	local command_name="$1"

	if ! command -v "$command_name" >/dev/null 2>&1; then
		echo "$command_name is required" >&2
		exit 1
	fi
}

require_status() {
	local got="$1"
	local want="$2"
	local label="$3"
	local body_file="${4:-}"

	if [[ "$got" == "$want" ]]; then
		return
	fi

	echo "$label: expected HTTP $want, got $got" >&2
	if [[ -n "$body_file" && -f "$body_file" ]]; then
		cat "$body_file" >&2
		echo >&2
	fi
	exit 1
}

curl_status() {
	local output_file="$1"
	shift

	curl -ksS --output "$output_file" --write-out "%{http_code}" "$@"
}

json_field() {
	local file="$1"
	local field="$2"

	jq -r ".$field" "$file"
}

wait_for_health() {
	local health_body="$tmp_dir/health.json"
	local status=""

	for _ in $(seq 1 30); do
		status="$(curl_status "$health_body" "$base_url/health" || true)"
		if [[ "$status" == "200" ]]; then
			return
		fi
		sleep 1
	done

	require_status "$status" "200" "health check" "$health_body"
}

wait_for_ready_avatar() {
	local avatar_id="$1"
	local metadata_body="$tmp_dir/avatar-metadata.json"
	local status=""
	local avatar_status=""

	for _ in $(seq 1 30); do
		status="$(curl_status "$metadata_body" "$base_url/api/v1/avatars/$avatar_id/metadata" || true)"
		if [[ "$status" == "200" ]]; then
			avatar_status="$(json_field "$metadata_body" status)"
			case "$avatar_status" in
			ready)
				return
				;;
			failed)
				echo "avatar processing failed" >&2
				cat "$metadata_body" >&2
				echo >&2
				exit 1
				;;
			esac
		fi
		sleep 1
	done

	echo "avatar did not become ready, last HTTP status: $status, last status: $avatar_status" >&2
	cat "$metadata_body" >&2 || true
	echo >&2
	exit 1
}

require_command curl
require_command jq

wait_for_health
echo "OK: health check returned HTTP 200"

email="observability-$(date +%s)@example.com"
resolve_body="$tmp_dir/resolve-user.json"
upload_body="$tmp_dir/upload-avatar.json"
select_body="$tmp_dir/select-current-avatar.txt"
current_avatar="$tmp_dir/current-avatar.png"

status="$(curl_status "$resolve_body" \
	--header "Content-Type: application/json" \
	--data "{\"email\":\"$email\"}" \
	"$base_url/api/v1/users/resolve")"
require_status "$status" "200" "resolve user" "$resolve_body"
user_id="$(json_field "$resolve_body" user_id)"
echo "OK: user resolved or created"

status="$(curl_status "$upload_body" \
	--header "X-User-ID: $user_id" \
	--form "file=@$repo_root/web/static/default-avatar.png;type=image/png;filename=observability.png" \
	"$base_url/api/v1/avatars")"
require_status "$status" "201" "upload avatar" "$upload_body"
avatar_id="$(json_field "$upload_body" id)"
echo "OK: avatar uploaded"

wait_for_ready_avatar "$avatar_id"
echo "OK: worker processed avatar to ready status"

status="$(curl_status "$select_body" \
	--request PATCH \
	--header "Content-Type: application/json" \
	--header "X-User-ID: $user_id" \
	--data "{\"avatar_id\":\"$avatar_id\"}" \
	"$base_url/api/v1/avatar")"
require_status "$status" "204" "select current avatar" "$select_body"
echo "OK: avatar selected as current"

status="$(curl_status "$current_avatar" "$base_url/api/v1/avatar?email=$email")"
require_status "$status" "200" "get current avatar by email" "$current_avatar"
echo "OK: current avatar received by email"

echo "OK: observability check passed"
echo "email: $email"
echo "user_id: $user_id"
echo "avatar_id: $avatar_id"
