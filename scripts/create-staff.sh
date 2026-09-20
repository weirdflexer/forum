#!/usr/bin/env bash
source "$(dirname "$0")/common.sh"
read -r -p 'Логин: ' STAFF_LOGIN
read -r -s -p 'Пароль (не менее 12 символов): ' STAFF_PASSWORD
printf '\n'
read -r -p 'Роль [admin/moderator, по умолчанию admin]: ' STAFF_ROLE
export STAFF_LOGIN STAFF_PASSWORD STAFF_ROLE="${STAFF_ROLE:-admin}"
cd "$FORUM_ROOT/backend"
go run ./cmd/api create-staff
echo 'Служебная учётная запись создана.'
