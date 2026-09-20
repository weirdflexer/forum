#!/usr/bin/env bash
source "$(dirname "$0")/common.sh"
if [[ "$(podman exec "$POSTGRES_CONTAINER" psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -tAc "SELECT 1 FROM staff_accounts WHERE login='admin'")" == 1 ]]; then
 echo 'Учётная запись admin уже существует. Пароль не изменён.'; exit 0
fi
umask 077
mkdir -p "$FORUM_ROOT/.run"
export STAFF_LOGIN=admin STAFF_PASSWORD="$(openssl rand -hex 12)" STAFF_ROLE=admin
(cd "$FORUM_ROOT/backend" && go run ./cmd/api create-staff)
cat > "$FORUM_ROOT/.run/local-access.txt" <<ACCESS
Локальный форум «Без имени»
Вход: http://localhost:8080/login
Логин: $STAFF_LOGIN
Пароль: $STAFF_PASSWORD

Это случайный пароль локальной служебной учётной записи.
Не публикуйте этот файл и .env. Новые аккаунты: make admin или панель управления.
ACCESS
echo 'Локальный администратор создан. Данные входа сохранены в .run/local-access.txt (доступ только владельцу).'
