#!/usr/bin/env bash

# Запускает сервер и воркер на тестовой инфраструктуре и проверяет сквозной сценарий загрузки, выбора, получения
# и удаления аватарки.

set -Eeuo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
env_file="${1:-.env.test}"
if [[ "$env_file" != /* ]]; then
	env_file="$repo_root/$env_file"
fi

if [[ ! -f "$env_file" ]]; then
	echo "create $env_file from .env.test.example" >&2
	exit 1
fi

set -a
# shellcheck disable=SC1090
. "$env_file"
set +a

: "${GOPH_PROFILE_TEST_POSTGRES_DB:?}"
: "${GOPH_PROFILE_TEST_POSTGRES_USER:?}"
: "${GOPH_PROFILE_TEST_POSTGRES_PASSWORD:?}"
: "${GOPH_PROFILE_TEST_MINIO_ROOT_USER:?}"
: "${GOPH_PROFILE_TEST_MINIO_ROOT_PASSWORD:?}"
: "${GOPH_PROFILE_TEST_RABBITMQ_USER:?}"
: "${GOPH_PROFILE_TEST_RABBITMQ_PASSWORD:?}"

tmp_dir="$(mktemp -d /tmp/goph-profile-e2e.XXXXXX)"
server_log="$tmp_dir/server.log"
worker_log="$tmp_dir/worker.log"
server_bin="$tmp_dir/server"
worker_bin="$tmp_dir/worker"
processes=()

dump_logs() {
	echo
	echo "server log:"
	tail -n 120 "$server_log" 2>/dev/null || true
	echo
	echo "worker log:"
	tail -n 120 "$worker_log" 2>/dev/null || true
}

cleanup() {
	for pid in "${processes[@]}"; do
		kill "$pid" 2>/dev/null || true
	done
	for pid in "${processes[@]}"; do
		wait "$pid" 2>/dev/null || true
	done
	rm -rf "$tmp_dir"
}

on_exit() {
	status=$?
	if [[ $status -ne 0 ]]; then
		dump_logs
	fi
	cleanup
	exit "$status"
}
trap on_exit EXIT

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

json_field() {
	local file="$1"
	local path="$2"

	python3 - "$file" "$path" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as f:
    value = json.load(f)

for part in sys.argv[2].split("."):
    value = value[part]

print(value)
PY
}

curl_status() {
	local output_file="$1"
	shift

	curl -k --silent --show-error --output "$output_file" --write-out "%{http_code}" "$@"
}

curl_status_and_type() {
	local output_file="$1"
	shift

	curl -k --silent --show-error --output "$output_file" --write-out "%{http_code} %{content_type}" "$@"
}

wait_for_health() {
	local health_body="$tmp_dir/health.json"
	local status=""

	for _ in $(seq 1 60); do
		if ! kill -0 "$server_pid" 2>/dev/null; then
			echo "server process stopped before health check" >&2
			exit 1
		fi

		status="$(curl -k --silent --output "$health_body" --write-out "%{http_code}" "$base_url/health" || true)"
		if [[ "$status" == "200" ]]; then
			return
		fi
		sleep 1
	done

	echo "server did not become healthy, last HTTP status: $status" >&2
	cat "$health_body" >&2 || true
	echo >&2
	exit 1
}

wait_for_ready_avatar() {
	local avatar_id="$1"
	local metadata_body="$tmp_dir/avatar-metadata.json"
	local status=""
	local avatar_status=""

	for _ in $(seq 1 60); do
		status="$(curl_status "$metadata_body" "$base_url/api/v1/avatars/$avatar_id/metadata")"
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

verify_avatar_list() {
	local file="$1"
	local avatar_id="$2"

	python3 - "$file" "$avatar_id" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as f:
    data = json.load(f)

avatar_id = sys.argv[2]
for avatar in data["avatars"]:
    if avatar["id"] == avatar_id:
        if avatar["status"] != "ready":
            raise SystemExit(f"avatar status is {avatar['status']}, expected ready")
        if not avatar["is_current"]:
            raise SystemExit("avatar is not current")
        break
else:
    raise SystemExit("uploaded avatar is not present in user avatar list")
PY
}

run_id="$(date +%Y%m%d%H%M%S)-$$"
http_address="${GOPH_PROFILE_E2E_HTTP_ADDRESS:-127.0.0.1:13202}"
base_url="https://$http_address"
database_uri="postgres://${GOPH_PROFILE_TEST_POSTGRES_USER}:${GOPH_PROFILE_TEST_POSTGRES_PASSWORD}"
database_uri="$database_uri@localhost:15432/${GOPH_PROFILE_TEST_POSTGRES_DB}?sslmode=disable"
queue_url="amqp://${GOPH_PROFILE_TEST_RABBITMQ_USER}:${GOPH_PROFILE_TEST_RABBITMQ_PASSWORD}@localhost:15673/"

export GOPH_PROFILE_HTTP_ADDRESS="$http_address"
export GOPH_PROFILE_TLS_CERT="$repo_root/certs/server.crt"
export GOPH_PROFILE_TLS_KEY="$repo_root/certs/server.key"
export GOPH_PROFILE_DATABASE_URI="${TEST_DATABASE_URI:-$database_uri}"
export GOPH_PROFILE_FILE_STORAGE_ENDPOINT="${TEST_FILE_STORAGE_ENDPOINT:-localhost:19000}"
export GOPH_PROFILE_FILE_STORAGE_ACCESS_KEY="$GOPH_PROFILE_TEST_MINIO_ROOT_USER"
export GOPH_PROFILE_FILE_STORAGE_SECRET_KEY="$GOPH_PROFILE_TEST_MINIO_ROOT_PASSWORD"
export GOPH_PROFILE_FILE_STORAGE_BUCKET="${TEST_FILE_STORAGE_BUCKET:-goph-profile-test}"
export GOPH_PROFILE_FILE_STORAGE_USE_SSL="false"
export GOPH_PROFILE_QUEUE_URL="${TEST_QUEUE_URL:-$queue_url}"
export GOPH_PROFILE_QUEUE_EXCHANGE="goph-profile.avatar.e2e.$run_id"
export GOPH_PROFILE_QUEUE_AVATAR_PROCESSING_QUEUE="goph-profile.avatar-processing.e2e.$run_id"
export GOPH_PROFILE_QUEUE_AVATAR_DELETION_QUEUE="goph-profile.avatar-deletion.e2e.$run_id"
export GOPH_PROFILE_QUEUE_AVATAR_PROCESSING_ROUTING_KEY="avatar.process"
export GOPH_PROFILE_QUEUE_AVATAR_DELETION_ROUTING_KEY="avatar.delete"
export GOPH_PROFILE_LOGGING_FORMAT="text"
export GOPH_PROFILE_LOGGING_LEVEL="warn"
export GOPH_PROFILE_LOGGING_ADD_SOURCE="false"

cd "$repo_root"

go build -o "$server_bin" ./cmd/server
go build -o "$worker_bin" ./cmd/worker

"$server_bin" >"$server_log" 2>&1 &
server_pid=$!
processes+=("$server_pid")
wait_for_health

"$worker_bin" >"$worker_log" 2>&1 &
worker_pid=$!
processes+=("$worker_pid")
sleep 1
if ! kill -0 "$worker_pid" 2>/dev/null; then
	echo "worker process stopped after start" >&2
	exit 1
fi

email="e2e-$run_id@example.com"
resolve_body="$tmp_dir/resolve-user.json"
status="$(curl_status "$resolve_body" \
	--header "Content-Type: application/json" \
	--data "{\"email\":\"$email\"}" \
	"$base_url/api/v1/users/resolve")"
require_status "$status" "200" "resolve user" "$resolve_body"
user_id="$(json_field "$resolve_body" user_id)"

upload_body="$tmp_dir/upload-avatar.json"
status="$(curl_status "$upload_body" \
	--header "X-User-ID: $user_id" \
	--form "file=@$repo_root/web/static/default-avatar.png;type=image/png;filename=e2e-avatar.png" \
	"$base_url/api/v1/avatars")"
require_status "$status" "201" "upload avatar" "$upload_body"
avatar_id="$(json_field "$upload_body" id)"

wait_for_ready_avatar "$avatar_id"

select_body="$tmp_dir/select-current-avatar.txt"
status="$(curl_status "$select_body" \
	--request PATCH \
	--header "Content-Type: application/json" \
	--header "X-User-ID: $user_id" \
	--data "{\"avatar_id\":\"$avatar_id\"}" \
	"$base_url/api/v1/avatar")"
require_status "$status" "204" "select current avatar" "$select_body"

list_body="$tmp_dir/list-user-avatars.json"
status="$(curl_status "$list_body" "$base_url/api/v1/users/$user_id/avatars")"
require_status "$status" "200" "list user avatars" "$list_body"
verify_avatar_list "$list_body" "$avatar_id"

current_avatar="$tmp_dir/current-avatar.png"
result="$(curl_status_and_type "$current_avatar" "$base_url/api/v1/avatar?email=$email")"
status="${result%% *}"
content_type="${result#* }"
require_status "$status" "200" "get current avatar by email" "$current_avatar"
if [[ "$content_type" != "image/png" ]]; then
	echo "current avatar: expected image/png, got $content_type" >&2
	exit 1
fi
if ! cmp -s "$repo_root/web/static/default-avatar.png" "$current_avatar"; then
	echo "current avatar content differs from uploaded file" >&2
	exit 1
fi

thumbnail="$tmp_dir/avatar-100.png"
result="$(curl_status_and_type "$thumbnail" "$base_url/api/v1/avatars/$avatar_id?size=100x100&format=png")"
status="${result%% *}"
content_type="${result#* }"
require_status "$status" "200" "get avatar thumbnail" "$thumbnail"
if [[ "$content_type" != "image/png" ]]; then
	echo "avatar thumbnail: expected image/png, got $content_type" >&2
	exit 1
fi
if [[ ! -s "$thumbnail" ]]; then
	echo "avatar thumbnail is empty" >&2
	exit 1
fi

delete_body="$tmp_dir/delete-current-avatar.txt"
status="$(curl_status "$delete_body" \
	--request DELETE \
	--header "X-User-ID: $user_id" \
	"$base_url/api/v1/avatar")"
require_status "$status" "204" "delete current avatar" "$delete_body"

echo "e2e checks passed"
