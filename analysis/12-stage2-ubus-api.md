# Документ №12. Этап 2: ubus/rpcd API и ACL

Дата: 22.11.2026
Основание: D-05 (единственный канал — ubus/rpcd, сетевой порт не открывается), D-14 (состояния), документ №11.
Статусы: **[У]** утверждено, **[П]** предлагается, **[Т]** требует теста, **[О]** открытый вопрос.
Кодирование не выполнялось; схемы ниже — контракт, а не реализация.

---

## 1. Общие правила

| Правило | Значение |
|---|---|
| Транспорт | ubus через rpcd; демон не слушает TCP/UDP |
| Формат | JSON; имена полей — `snake_case`; время — Unix ms UTC |
| Версии | `api_version` в `vpnmanager.status`; несовместимое изменение — новая мажорная версия |
| Идентификаторы | строки; серверы — устойчивый ID (документ №13 §3) |
| Идемпотентность | изменяющие методы принимают `client_token`; повтор с тем же токеном не создаёт вторую операцию |
| Длительные операции | возвращают `op_id`; прогресс — `vpnmanager.operation.status` и событие `operation` |
| Ошибки | объект `{code, message, details?, retry_after?}`; `message` — для человека, `code` — для логики |
| Ограничения | тело запроса ≤ 256 КБ; список ≤ 1000 записей; строки ≤ 4 КБ (кроме импорта — отдельный лимит) |
| Секреты | URL подписки, UUID/ключи не возвращаются после сохранения; только маска |
| Права | проверяются rpcd (ACL) **и** демоном (повторно) |

Коды ошибок:

| Код | Смысл | HTTP-аналог (для понимания) |
|---|---|---|
| `VM_OK` | успех | 200 |
| `VM_ERR_INVALID_ARGUMENT` | неверный аргумент/схема | 400 |
| `VM_ERR_FORBIDDEN` | нет права | 403 |
| `VM_ERR_NOT_FOUND` | объект не найден | 404 |
| `VM_ERR_CONFLICT` | конфликт ревизий/состояний | 409 |
| `VM_ERR_BUSY` | идёт другая операция (`retry_after`) | 423 |
| `VM_ERR_VALIDATION` | конфигурация невалидна | 422 |
| `VM_ERR_UNSUPPORTED` | операция недоступна в v1 | 501 |
| `VM_ERR_TIMEOUT` | истекло время фазы | 504 |
| `VM_ERR_STORAGE` | ошибка записи/чтения | 500 |
| `VM_ERR_NO_SPACE` | нет места | 507 |
| `VM_ERR_ROLLED_BACK` | операция отменена откатом | 409 |
| `VM_ERR_INTERNAL` | внутренняя ошибка | 500 |

---

## 2. Объекты

| Объект | Назначение | Изменяет |
|---|---|---|
| `vpnmanager` | состояние, включение/выключение | да |
| `vpnmanager.config` | настройки, сохранение, apply | да |
| `vpnmanager.servers` | серверы, выбор, проверка | да |
| `vpnmanager.subscriptions` | подписки | да |
| `vpnmanager.profiles` | профили и назначения | да |
| `vpnmanager.devices` | устройства и режимы | да |
| `vpnmanager.dns` | DNS-политика | да |
| `vpnmanager.policy` | UDP-режим, состояние правил | да (UDP) |
| ~~`vpnmanager.ipv6`~~ | **reserved (v2)** — вне v1, решение №11; объект не регистрируется | нет |
| `vpnmanager.blocklists` | источники, категории, версии, diff, whitelist, статистика, аварийное отключение | да |
| `vpnmanager.maintenance` | вход/выход из Maintenance Mode, проверка компонентов | да |
| `vpnmanager.audit` | аудит настроек, Undo, история изменений | да (Undo) |
| `vpnmanager.devices` (bulk) | массовые операции, Dry Run, группы | да |
| `vpnmanager.diagnostics` | отчёты, журнал | нет |
| `vpnmanager.backup` | копии | да |
| `vpnmanager.operation` | статус длительных операций | нет (кроме cancel) |

---

## 3. Методы

### 3.1 `vpnmanager`

| Метод | Запрос | Ответ |
|---|---|---|
| `status` | `{}` | см. §4 |
| `enable` | `{server_id?, profile_id?, client_token?}` | `{op_id}` |
| `disable` | `{reason?, client_token?}` | `{op_id}` |
| `reload` | `{}` | `{op_id}` — перечитать сохранённое и пересчитать состояние |

`disable` — штатная операция (D-14): останавливает движок, снимает правила, ставит `intent=disabled`, событие `disabled_by_user`. Ошибкой не считается.

### 3.2 `vpnmanager.config`

| Метод | Запрос | Ответ |
|---|---|---|
| `get` | `{}` | `{revision, settings, config_state}` |
| `set` | `{patch, expected_revision, client_token?}` | `{revision, config_state:"Saved"}` |
| `validate` | `{settings?}` | `{valid, errors[]}` |
| `apply` | `{expected_revision, canary_mac?, client_token?}` | `{op_id}` |
| `discard` | `{expected_revision}` | `{revision}` |

`apply` при `expected_revision` ≠ текущей → `VM_ERR_CONFLICT`. Успех — только `Applied` после trial (D-06).

### 3.3 `vpnmanager.servers`

| Метод | Запрос | Ответ |
|---|---|---|
| `list` | `{subscription_id?, offset?, limit?}` | `{total, items[]}` (без секретов) |
| `get` | `{id}` | `{server, health}` |
| `select` | `{id, force?, client_token?}` | `{op_id}` — немедленная проверка (D-03) |
| `test` | `{id}` | `{health}` — без смены активного |
| `reorder` | `{ids[]}` | `{ok}` |

### 3.4 `vpnmanager.subscriptions`

| Метод | Запрос | Ответ |
|---|---|---|
| `list` | `{}` | `{items[]}` (URL маскирован) |
| `add` | `{url?, file?, name, restricted?, client_token?}` | `{id, report}` |
| `update` | `{id, client_token?}` | `{op_id}` |
| `remove` | `{id, confirm:true}` | `{ok}` |
| `reorder` | `{ids[]}` | `{ok}` |
| `report` | `{id}` | `{accepted, rejected[], reasons[]}` |

### 3.5 `vpnmanager.profiles`

| Метод | Запрос | Ответ |
|---|---|---|
| `list` | `{}` | `{items[]}` |
| `get` | `{id}` | `{profile}` |
| `save` | `{profile, expected_revision?, client_token?}` | `{id, revision}` |
| `remove` | `{id, confirm:true}` | `{ok}` |
| `assign` | `{server_id, profile_id}` | `{ok}` |

### 3.6 `vpnmanager.devices`

| Метод | Запрос | Ответ |
|---|---|---|
| `list` | `{online_only?}` | `{items[]}` |
| `set_mode` | `{mac, mode:"default"\|"always_direct"}` | `{ok}` |
| `merge` | `{macs[]}` | `{id}` |
| `forget` | `{mac, confirm:true}` | `{ok}` |

### 3.7 `vpnmanager.dns`

| Метод | Запрос | Ответ |
|---|---|---|
| `status` | `{}` | `{chain, upstream, fallback, latency_ms, aaaa_mode}` |
| `set_chain` | `{chain, client_token?}` | `{op_id}` |
| `test` | `{domain?}` | `{resolver, answers, latency_ms}` |
| `set_aaaa` | `{mode:"pass"\|"filter_lan"}` | `{ok, warning?}` |

### 3.8 `vpnmanager.policy`

| Метод | Запрос | Ответ |
|---|---|---|
| `status` | `{}` | `{applied, table, mark, counters, mismatch}` |
| `set_udp` | `{mode:"proxy"\|"block"\|"direct", allowlist?}` | `{op_id}` |

### 3.9 `vpnmanager.ipv6` — **RESERVED, ВНЕ v1**

Объект не регистрируется в v1 (ТЗ §5, решение №11). Имена методов сохранены только как задел v2, чтобы не переиспользовать их с другой семантикой. Любой вызов → `VM_ERR_UNSUPPORTED`. Состояние IPv6 не вычисляется и не публикуется.

### 3.9a `vpnmanager.blocklists` (новое)

| Метод | Запрос | Ответ |
|---|---|---|
| `status` | `{}` | состояние подсистемы, активные версии, аномалии |
| `categories.list` / `categories.set` | `{}` / `{id, enabled}` | категории |
| `sources.list` / `add` / `update` / `remove` | `{url|path, category, name}` | источник + метаданные |
| `versions.list` | `{source_id}` | версии со статусом |
| `diff` | `{from, to}` | компактный diff (add/remove/category) |
| `dry_run` | `{version_id}` | вердикт без побочных эффектов |
| `apply` | `{version_id, client_token}` | `op_id` |
| `rollback` | `{version_id, client_token}` | `op_id` |
| `whitelist.add` / `remove` | `{domain, permanent, ttl?}` | запись |
| `system_exceptions.list` | `{}` | список (только чтение) |
| `sets.get` / `sets.set` | `{device|group}` | набор категорий |
| `stats` / `export` | `{period}` / `{format, period}` | агрегаты / файл |
| `emergency_disable` | `{confirm: true}` | состояние `disabled` |

### 3.9b `vpnmanager.maintenance`, `vpnmanager.audit` (новые)

| Объект.метод | Запрос | Ответ |
|---|---|---|
| `maintenance.status` / `enter` / `exit` | `{}` / `{reason}` / `{force?}` | состояние + результат проверок компонентов |
| `audit.list` | `{filter, period, limit}` | записи аудита (без секретов) |
| `audit.undo` | `{entry_id, client_token}` | `op_id` или отказ при зависимостях |

| Метод | Запрос | Ответ |
|---|---|---|
| `status` | `{}` | см. §4.2 |
| `set_guard` | `{enabled, client_token?}` | `{op_id}` |

`set_guard` меняет только собственную цепочку; `network`/`firewall`/RA не трогает.

### 3.10 `vpnmanager.diagnostics`

| Метод | Запрос | Ответ |
|---|---|---|
| `report` | `{kind:"summary"\|"network"\|"dns"\|"ipv6"\|"conflicts"\|"storage"}` | `{report}` (read-only) |
| `tail` | `{lines:1..500, since_seq?}` | `{events[]}` |
| `conntrack` | `{}` | `{count, max, percent}` |
| `listeners` | `{}` | `{items[]}` |

### 3.11 `vpnmanager.backup`

| Метод | Запрос | Ответ |
|---|---|---|
| `create` | `{include_secrets:true, confirm:true}` | `{path, size, sha256}` |
| `list` | `{}` | `{items[]}` |
| `restore` | `{path, confirm:true, client_token?}` | `{op_id}` |

### 3.12 `vpnmanager.operation`

| Метод | Запрос | Ответ |
|---|---|---|
| `status` | `{op_id}` | `{op_id, type, phase, state, progress, error?}` |
| `cancel` | `{op_id}` | `{ok}` — только до фазы применения |

---

## 4. Примеры ответов

### 4.1 `vpnmanager.status`

```json
{
  "api_version": 1,
  "state": "DEGRADED",
  "substate": "SERVER_UNREACHABLE",
  "intent": "enabled",
  "config_state": "Applied",
  "since": 1795344000000,
  "signals": {
    "intent": {"value": "enabled", "ts": 1795343000000, "fresh": true},
    "process": {"value": "running", "pid": 4211, "ts": 1795343995000, "fresh": true},
    "config": {"value": "valid", "hash": "…", "ts": 1795343000000, "fresh": true},
    "policy": {"value": "applied", "ts": 1795343990000, "fresh": true},
    "server": {"value": "unreachable", "fails": 2, "ts": 1795343999000, "fresh": true},
    "traffic": {"value": "unverified", "ts": 1795343800000, "fresh": false}
  },
  "warnings": [
    {"code": "SERVER_UNREACHABLE", "severity": "warning"},
    {"code": "TRAFFIC_UNVERIFIED", "severity": "warning"}
  ],
  "ipv6": {"wan_present": false, "guard_installed": true},
  "active": {"server_id": "srv_…", "profile_id": "prf_…"},
  "revision": 42
}
```

Правило: `ACTIVE` невозможен при `fresh:false` у любого обязательного признака.

### 4.2 `vpnmanager.ipv6.status`

```json
{
  "wan_present": false,
  "sources": [],
  "guard_installed": true,
  "client_egress": "blocked",
  "state": "ABSENT",
  "counter": 0,
  "note": "Отсутствие default route не считается защитой"
}
```

### 4.3 Ошибка

```json
{
  "code": "VM_ERR_BUSY",
  "message": "Выполняется применение конфигурации",
  "details": {"op_id": "op_…", "phase": "TRIAL"},
  "retry_after": 15
}
```

---

## 5. События

Подписка: `ubus subscribe vpnmanager.event`. Конверт:

```json
{"seq": 1042, "ts": 1795344000123, "type": "state", "data": {"from": "ACTIVE", "to": "DEGRADED", "reason": "server_unreachable"}}
```

| Тип | Когда | Данные |
|---|---|---|
| `state` | смена состояния/подсостояния | from, to, reason, signals_hash |
| `intent` | enable/disable | value, initiator |
| `config` | saved/applied/pending/invalid | revision, state |
| `servers` | изменение состава/порядка/здоровья | counts, changed_ids |
| `devices` | online/offline/назначение | mac, mode |
| `dns` | смена цепочки/fallback | chain, upstream |
| `ipv6` | появление/исчезновение, сбой Guard | state, sources |
| `operation` | прогресс длительной операции | op_id, phase, progress |
| `log` | важное событие журнала | severity, code |

Требования: монотонный `seq`; события не содержат секретов; при переполнении буфера UI получает `resync_required` и перечитывает `status`.

---

## 6. ACL

Файл `/usr/share/rpcd/acl.d/luci-app-vpnmanager.json` (имя — [П]). Три группы:

```json
{
  "luci-app-vpnmanager": {
    "description": "VPN Manager: просмотр состояния и диагностика",
    "read": {
      "ubus": {
        "vpnmanager": ["status"],
        "vpnmanager.operation": ["status"],
        "vpnmanager.servers": ["list", "get", "test"],
        "vpnmanager.subscriptions": ["list"],
        "vpnmanager.profiles": ["list", "get"],
        "vpnmanager.devices": ["list"],
        "vpnmanager.dns": ["status"],
        "vpnmanager.policy": ["status"],
        "vpnmanager.blocklists": ["status", "categories.list", "sources.list", "versions.list", "diff", "stats", "system_exceptions.list", "sets.get"],
        "vpnmanager.maintenance": ["status"],
        "vpnmanager.audit": ["list"],
        "vpnmanager.diagnostics": ["report", "tail", "conntrack", "listeners"]
      },
      "uci": ["dhcp", "network", "firewall"]
    }
  },
  "luci-app-vpnmanager-control": {
    "description": "VPN Manager: управление VPN и данными",
    "write": {
      "ubus": {
        "vpnmanager": ["enable", "disable", "reload"],
        "vpnmanager.config": ["set", "apply", "discard", "validate"],
        "vpnmanager.servers": ["select", "reorder"],
        "vpnmanager.subscriptions": ["add", "update", "remove", "reorder"],
        "vpnmanager.profiles": ["save", "remove", "assign"],
        "vpnmanager.devices": ["set_mode", "merge"],
        "vpnmanager.dns": ["set_chain", "set_aaaa"],
        "vpnmanager.policy": ["set_udp"],
        "vpnmanager.blocklists": ["categories.set", "sources.add", "sources.update", "sources.remove", "dry_run", "apply", "whitelist.add", "whitelist.remove", "sets.set"],
        "vpnmanager.devices": ["set_mode", "merge", "assign", "unassign", "temp_direct", "bulk_preview", "bulk_apply"],
        "vpnmanager.maintenance": ["enter", "exit"],
        "vpnmanager.audit": ["undo"],
        "vpnmanager.operation": ["cancel"]
      }
    }
  },
  "luci-app-vpnmanager-admin": {
    "description": "VPN Manager: backup, restore, очистка",
    "write": {
      "ubus": {
        "vpnmanager.backup": ["create", "restore"],
        "vpnmanager.devices": ["forget"],
        "vpnmanager.blocklists": ["rollback", "emergency_disable"]
      }
    }
  }
}
```

Правила:
1. Чтение статуса не даёт права менять.
2. Backup/restore — отдельная группа (секреты в архиве).
3. Демон повторно проверяет право на метод по `ubus`-сессии; при отсутствии — `VM_ERR_FORBIDDEN`.
4. UI не получает `uci`-права на запись: конфигурацию меняет только демон.
5. `diagnostics` доступна read-группе, но не содержит секретов.

---

## 7. Идемпотентность и операции

- `client_token` (UUID) хранится 10 минут; повтор возвращает тот же `op_id`.
- Одна изменяющая операция одновременно; вторая — `VM_ERR_BUSY` с `retry_after`.
- Фазы операции видны в `operation.status`; UI не блокируется.
- `cancel` разрешён до `APPLY_POLICY`; после — только rollback по завершении.
- После перезапуска демона `op_id` прошлой операции → `state:"interrupted"`, UI перечитывает статус.

---

## 8. Что требует проверки

| Пункт | Статус |
|---|---|
| Точный синтаксис ACL для вложенных объектов ubus | [Т] T-05 |
| Поведение rpcd при недоступном демоне | [Т] |
| Лимит 256 КБ и размер ответов `list` | [Т] |
| Подписка на события из LuCI JS | [Т] |
| Разделение прав в реальной сессии LuCI | [Т] T-05 |
| Синтаксис ACL для новых объектов `blocklists`/`maintenance`/`audit` | [Т] W-11 |

---

## 9. Идентичность вызывающего субъекта (ТЗ §4, решение №9)

**Требование:** нельзя доверять имени пользователя, переданному браузером. Механизм должен быть документирован, а не подразумеваться.

| Уровень | Что проверяется | Источник доверия |
|---|---|---|
| rpcd | право на метод по ACL и сессии ubus | rpcd/ubus (не браузер) |
| ubus | существование и владелец сессии; `ubus`-соединение от rpcd | ядро/ubus |
| демон | peer credentials соединения (PID/uid вызывающего) + наличие сессии | `SO_PEERCRED`/ubus-сессия [Т] |
| демон | право на конкретный объект и операцию | повторная проверка внутри демона |

**Правила:**
1. Любое поле вида `user`, `username`, `actor` в теле запроса **игнорируется** для авторизации; оно может использоваться только как подсказка для аудита и всегда заменяется фактическим субъектом.
2. Аудит пишется с `actor` из проверенной сессии, а не из запроса.
3. Отсутствие сессии/прав → `VM_ERR_FORBIDDEN` без раскрытия деталей.
4. Ролевой модели в v1 нет (ТЗ §14): доступ только admin/root, разделение — по группам ACL.
5. Механизм считается **неподтверждённым** до W-11: до его прохождения нельзя утверждать, что API защищён. Это ограничение, а не «поддерживается».
6. `VM_ERR_UNSUPPORTED` возвращается для `vpnmanager.ipv6.*` и любых методов вне v1.
