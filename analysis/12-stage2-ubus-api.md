# Документ №12. Этап 2: ubus/rpcd API и ACL

**Актуализация U9 (26.11.2026, ред. 2 — по итогам независимого ревью).** Из API выведены `vpnmanager.dns.*` и `vpnmanager.blocklists.*`; удалены `servers.select`/`servers.test`/`servers.reorder`, поле `health`, три режима UDP, `diagnostics.tail` и `diagnostics.conntrack`. Добавлены `vpnmanager.logs`, стратегия, ручной override, автообновление подписки. Разделы §2, §3, §5, §6 и §10 сведены к **одному реестру** (§6): методы, права чтения/записи и события описаны один раз, ACL не содержит методов вне реестра. Идентичность вызывающего (§9) сохраняется.

Дата: 22.11.2026.
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
| Идентификаторы | строки; `srv_id` — идентификатор сервера **в текущем наборе** (документ №13 §3), не межверсийная идентичность |
| Идемпотентность | изменяющие методы принимают `client_token`; повтор с тем же токеном не создаёт вторую операцию |
| Длительные операции | возвращают `op_id`; прогресс — `vpnmanager.operation.status` |
| Ошибки | объект `{code, message, details?, retry_after?}`; `message` — для человека, `code` — для логики |
| Ограничения | тело запроса ≤ 256 КБ; список ≤ 1000 записей; строки ≤ 4 КБ (кроме импорта — отдельный лимит) |
| Секреты | URL подписки, UUID/ключи не возвращаются после сохранения; только маска |
| Права | проверяются rpcd (ACL) **и** демоном (повторно) по реестру §6 |

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
| `vpnmanager.servers` | серверы, стратегия, ручной override | да |
| `vpnmanager.subscriptions` | подписки и автообновление | да |
| `vpnmanager.profiles` | профили и назначения | да |
| `vpnmanager.devices` | устройства и режимы | да |
| `vpnmanager.policy` | состояние правил и UDP allowlist | да (allowlist) |
| `vpnmanager.maintenance` | вход/выход из Maintenance Mode, проверка компонентов | да |
| `vpnmanager.audit` | аудит изменений настроек, Undo | да (Undo) |
| `vpnmanager.logs` | два Xray-лога и параметры их ротации | да (порог) |
| `vpnmanager.diagnostics` | read-only отчёты, слушатели | нет |
| `vpnmanager.backup` | копии конфигурации | да |
| `vpnmanager.operation` | статус длительных операций | нет (кроме cancel) |

**Не регистрируются в v1:** `vpnmanager.dns` (DNS — штатный DNS роутера), `vpnmanager.blocklists` (подсистема исключена). Имена не переиспользуются с другой семантикой; вызов любого прежнего метода → `VM_ERR_UNSUPPORTED`.

**Зарезервировано (v2):** `vpnmanager.ipv6` — объект не регистрируется, состояние IPv6 не вычисляется и не публикуется.

---

## 3. Методы

### 3.1 `vpnmanager`

| Метод | Запрос | Ответ |
|---|---|---|
| `status` | `{}` | см. §4 |
| `enable` | `{server_id?, profile_id?, client_token?}` | `{op_id}` |
| `disable` | `{reason?, client_token?}` | `{op_id}` |
| `reload` | `{}` | `{op_id}` — перечитать сохранённое и пересчитать состояние |

`disable` — штатная операция (D-14): останавливает движок, снимает правила, ставит `intent=disabled`. Ошибкой не считается.

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
| `get` | `{id}` | `{server}`; доступность показывает Observatory, приложение её не хранит |
| `get_strategy` | `{}` | `{strategy, mode, override_server_id?}` |
| `set_strategy` | `{strategy, client_token?}` | `{op_id}` |
| `set_override` | `{server_id?, client_token?}` | `{op_id}`; без `server_id` — возврат в `auto` |

`strategy`: `random` (по умолчанию) | `roundRobin` | `leastPing` | `leastLoad`. Неподдерживаемая версия Xray → `VM_ERR_UNSUPPORTED`.
`mode`: `auto` | `manual`. Ручной override — один сервер из текущего набора; при недоступности действует `fallbackTag`. Сохранение override при обновлении подписки — §10 и №13 §3.

### 3.4 `vpnmanager.subscriptions`

| Метод | Запрос | Ответ |
|---|---|---|
| `list` | `{}` | `{items[]}` (URL маскирован) |
| `add` | `{url?, file?, name, restricted?, client_token?}` | `{id, report}` |
| `update` | `{id, client_token?}` | `{op_id}` |
| `remove` | `{id, confirm:true}` | `{ok}` |
| `reorder` | `{ids[]}` | `{ok}` |
| `report` | `{id}` | `{accepted, rejected[], reasons[]}` |
| `get_autoupdate` | `{}` | `{enabled, interval_hours, last_run?, next_run?}` |
| `set_autoupdate` | `{enabled, interval_hours: 6\|12\|24, client_token?}` | `{ok}` |

Автообновление выключено по умолчанию; планировщик — штатный `crond`. Первый запуск — по выбранному интервалу; немедленный первый запрос не утверждён. Ручное `update` доступно всегда.

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

### 3.7 `vpnmanager.dns` — **выведен (superseded U9)**

Методов нет. DNS в v1 — штатный DNS роутера; приложение не предоставляет API управления DNS, не переключает upstream, не отдаёт DNS-состояний и не выполняет DNS-фильтрацию. Вызов любого прежнего метода возвращает `VM_ERR_UNSUPPORTED`.

### 3.8 `vpnmanager.policy`

| Метод | Запрос | Ответ |
|---|---|---|
| `status` | `{}` | `{applied, table, mark, mismatch}` |
| `set_udp` | `{allowlist: [ {port \| from, to} ]}` | `{op_id}` |

Единственный параметр UDP — `allowlist`. Режимов `proxy`/`block`/`direct` нет: UDP вне allowlist идёт DIRECT и не блокируется; Always DIRECT выше allowlist. Пустой список — весь UDP идёт DIRECT.

### 3.9 `vpnmanager.ipv6` — **RESERVED, ВНЕ v1**

Объект не регистрируется в v1 (ТЗ §5, решение №11). Имена методов сохранены только как задел v2, чтобы не переиспользовать их с другой семантикой. Любой вызов → `VM_ERR_UNSUPPORTED`. Состояние IPv6 не вычисляется и не публикуется.

### 3.9a `vpnmanager.blocklists` — **выведен (superseded U9)**

Методов нет: подсистема Blocklists исключена из проекта. Прежние `status`, `categories.*`, `sources.*`, `versions.*`, `diff`, `dry_run`, `apply`, `rollback`, `whitelist.*`, `sets.*`, `stats`, `emergency_disable` не реализуются; вызов возвращает `VM_ERR_UNSUPPORTED`. События `blocklists.*` не публикуются.

### 3.9b `vpnmanager.maintenance`, `vpnmanager.audit`

| Объект.метод | Запрос | Ответ |
|---|---|---|
| `maintenance.status` / `enter` / `exit` | `{}` / `{reason}` / `{force?}` | состояние + результат проверок компонентов |
| `audit.list` | `{filter, period, limit}` | записи аудита настроек (без секретов) |
| `audit.undo` | `{entry_id, client_token}` | `op_id` или отказ при зависимостях |

Аудит — только изменения настроек. Общей ленты событий приложения в v1 нет.

### 3.10 `vpnmanager.logs`

| Метод | Запрос | Ответ |
|---|---|---|
| `list` | `{}` | `{files: [{name:"access.log"\|"error.log", size, mtime}]}` |
| `tail` | `{file, lines: 20\|50\|100}` | `{lines[]}` |
| `get_rotation` | `{}` | `{threshold_mb, last_run?, result?}` |
| `set_rotation` | `{threshold_mb, client_token?}` | `{ok}` |

Только два файла Xray. Ручной очистки, удаления и перезапуска Xray ради ротации нет. `threshold_mb` — целое 1…50, по умолчанию 5; диапазон [П] до W-15.

### 3.11 `vpnmanager.diagnostics`

| Метод | Запрос | Ответ |
|---|---|---|
| `report` | `{kind:"summary"\|"network"\|"conflicts"\|"storage"}` | `{report}` (read-only) |
| `listeners` | `{}` | `{items[]}` |

Журнала событий, `tail` и `conntrack` в API v1 нет. Вид `ipv6` не формируется.

### 3.12 `vpnmanager.backup`

| Метод | Запрос | Ответ |
|---|---|---|
| `create` | `{include_secrets:true, confirm:true}` | `{path, size, sha256}` |
| `list` | `{}` | `{items[]}` |
| `restore` | `{path, confirm:true, client_token?}` | `{op_id}` |

### 3.13 `vpnmanager.operation`

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
  "substate": "TRAFFIC_UNVERIFIED",
  "intent": "enabled",
  "config_state": "Applied",
  "since": 1795344000000,
  "signals": {
    "intent": {"value": "enabled", "ts": 1795343000000, "fresh": true},
    "process": {"value": "running", "pid": 4211, "ts": 1795343995000, "fresh": true},
    "config": {"value": "valid", "hash": "…", "ts": 1795343000000, "fresh": true},
    "policy": {"value": "applied", "ts": 1795343990000, "fresh": true},
    "traffic": {"value": "unverified", "ts": 1795343800000, "fresh": false}
  },
  "warnings": [
    {"code": "TRAFFIC_UNVERIFIED", "severity": "warning"}
  ],
  "active": {"server_id": "srv_…", "profile_id": "prf_…", "selection": "auto"},
  "revision": 42
}
```

Правило: `ACTIVE` невозможен при `fresh:false` у любого обязательного признака. Отдельного признака «здоровье сервера» нет: доступность сообщает Observatory.

**Наблюдаемость (OQ-14, вариант B — утверждён 27.11.2026).** Источник данных об outbound — локальный gRPC `ObservatoryService.GetOutboundStatus` (listener только на `127.0.0.1`, в `api.services` только `ObservatoryService`; `metrics`/`/debug/vars` не используются). `status` получает два разных по природе блока:

| Поле | Природа | Содержимое |
|---|---|---|
| `observability` | служебное | `{state: ok|no_data|stale|error, last_ok_ts, error_code}` |
| `servers[]` | **наблюдение** | `{server_id, outbound_tag, alive, delay_ms, last_seen_ts, last_try_ts, fresh}` |
| `fallback` | **производное** | `{active, mode: direct|block, source: "derived", basis: "no_alive_outbound"}` |

`servers[]` публикуется только из ответа API. `fallback` вычисляется по отсутствию живых outbound подписки и всегда помечается `source: "derived"`: фактический выбор балансировщика и факт применения `fallbackTag` Xray не сообщает. События `service.fallback_entered`/`service.fallback_left` допускаются только как производные переходы с той же пометкой.

При `observability.state != ok` поля `servers`/`fallback` не заполняются или помечаются несвежими; подстановка `healthy`/`unavailable` запрещена. `error_code` различает `unimplemented` (сервис не зарегистрирован), `deadline_exceeded`/`transport` (ошибка вызова) и не используется для отказа запуска движка. Ошибка создания listener — отказ запуска Xray (`ENGINE_DOWN`), а не `api_error`; при работающем Xray ошибка вызова API публикуется как `observability=error` и не выдаётся за отказ VPN (№20 §3.1, §7.2, REQ-168).

**Что именно не видно.** `status` не содержит и не может содержать две разные вещи: (1) выбранный для потока outbound и состояние балансировщика и (2) факт применения `fallbackTag` — это внутреннее состояние Xray, а не поле сервиса (№20 §3.1, уровни 2–4). Доступность отдельных outbound, наоборот, видна через API и публикуется как наблюдение. Ошибка процесса Xray (S2) и состояние сервиса по подтверждённым признакам (S1–S4, S6) от этого не зависят и публикуются раздельно.

### 4.2 `vpnmanager.ipv6.status` — зарезервировано: пример задела v2 (в v1 не возвращается)

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
| `servers` | изменение состава серверов, стратегии или override | strategy, mode, changed_ids |
| `devices` | online/offline/назначение | mac, mode |
| `operation` | прогресс длительной операции | op_id, phase, progress |
| `service.fallback_entered` / `service.fallback_left` | вход/выход из `FALLBACK_*` | `FALLBACK_DIRECT`/`FALLBACK_BLOCK`; публикуются только как производные переходы (`source=derived`) при `observability.state=ok` (OQ-14 B) |
| `subscription.updated` / `subscription.update_failed` | результат обновления | accepted, rejected |
| `strategy.changed` | смена стратегии | from, to |
| `override.set` / `override.preserved` / `override.reset` | ручной выбор | server_id, reason |
| `logrotate.completed` / `logrotate.failed` | результат ротации | file, size |

**Не публикуются:** `dns.*`, `blocklists.*`, `server.down`/`server.up`, событие `log`. Общей пользовательской ленты событий нет; события `state`/`intent`/`config`/`operation` — транзиентные сигналы текущего статуса, а не история.

Требования: монотонный `seq`; события не содержат секретов; при переполнении буфера UI получает `resync_required` и перечитывает `status`.

---

## 6. Единый реестр объектов, методов и прав

**Это единственный источник истины для API, ACL, контрактов и UI.** Метод, которого нет в таблице, не существует; ACL не может ссылаться на метод вне таблицы.

Группы ACL: **read** — просмотр и диагностика; **control** — управление VPN и данными; **admin** — backup/restore и удаление записей.

| Объект | Метод | read | control | admin |
|---|---|---|---|---|
| `vpnmanager` | `status` | да | да | да |
| `vpnmanager` | `enable`, `disable`, `reload` | — | да | — |
| `vpnmanager.config` | `get` | да | да | да |
| `vpnmanager.config` | `set`, `apply`, `discard`, `validate` | — | да | — |
| `vpnmanager.servers` | `list`, `get`, `get_strategy` | да | да | да |
| `vpnmanager.servers` | `set_strategy`, `set_override` | — | да | — |
| `vpnmanager.subscriptions` | `list`, `report`, `get_autoupdate` | да | да | да |
| `vpnmanager.subscriptions` | `add`, `update`, `remove`, `reorder`, `set_autoupdate` | — | да | — |
| `vpnmanager.profiles` | `list`, `get` | да | да | да |
| `vpnmanager.profiles` | `save`, `remove`, `assign` | — | да | — |
| `vpnmanager.devices` | `list` | да | да | да |
| `vpnmanager.devices` | `set_mode`, `merge` | — | да | — |
| `vpnmanager.devices` | `forget` | — | — | да |
| `vpnmanager.policy` | `status` | да | да | да |
| `vpnmanager.policy` | `set_udp` | — | да | — |
| `vpnmanager.maintenance` | `status` | да | да | да |
| `vpnmanager.maintenance` | `enter`, `exit` | — | да | — |
| `vpnmanager.audit` | `list` | да | да | да |
| `vpnmanager.audit` | `undo` | — | да | — |
| `vpnmanager.logs` | `list`, `tail`, `get_rotation` | да | да | да |
| `vpnmanager.logs` | `set_rotation` | — | да | — |
| `vpnmanager.diagnostics` | `report`, `listeners` | да | да | да |
| `vpnmanager.backup` | `list` | да | да | да |
| `vpnmanager.backup` | `create`, `restore` | — | — | да |
| `vpnmanager.operation` | `status` | да | да | да |
| `vpnmanager.operation` | `cancel` | — | да | — |

Файл `/usr/share/rpcd/acl.d/luci-app-vpnmanager.json` (имя — [П]):

```json
{
  "luci-app-vpnmanager": {
    "description": "VPN Manager: просмотр состояния и диагностика",
    "read": {
      "ubus": {
        "vpnmanager": ["status"],
        "vpnmanager.operation": ["status"],
        "vpnmanager.servers": ["list", "get", "get_strategy"],
        "vpnmanager.subscriptions": ["list", "report", "get_autoupdate"],
        "vpnmanager.profiles": ["list", "get"],
        "vpnmanager.devices": ["list"],
        "vpnmanager.policy": ["status"],
        "vpnmanager.maintenance": ["status"],
        "vpnmanager.audit": ["list"],
        "vpnmanager.logs": ["list", "tail", "get_rotation"],
        "vpnmanager.diagnostics": ["report", "listeners"],
        "vpnmanager.backup": ["list"]
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
        "vpnmanager.servers": ["set_strategy", "set_override"],
        "vpnmanager.subscriptions": ["add", "update", "remove", "reorder", "set_autoupdate"],
        "vpnmanager.profiles": ["save", "remove", "assign"],
        "vpnmanager.devices": ["set_mode", "merge"],
        "vpnmanager.policy": ["set_udp"],
        "vpnmanager.maintenance": ["enter", "exit"],
        "vpnmanager.audit": ["undo"],
        "vpnmanager.logs": ["set_rotation"],
        "vpnmanager.operation": ["cancel"]
      }
    }
  },
  "luci-app-vpnmanager-admin": {
    "description": "VPN Manager: backup, restore, удаление записей",
    "write": {
      "ubus": {
        "vpnmanager.backup": ["create", "restore"],
        "vpnmanager.devices": ["forget"]
      }
    }
  }
}
```

Правила:
1. Чтение статуса не даёт права менять. Права read-группы на `logs` включают `list`/`tail`/`get_rotation` и **не** включают `set_rotation`.
2. Backup/restore и `devices.forget` — отдельная группа (секреты в архиве; удаление записи).
3. Демон повторно проверяет право на метод по `ubus`-сессии; при отсутствии — `VM_ERR_FORBIDDEN`.
4. UI не получает `uci`-права на запись: конфигурацию меняет только демон.
5. `diagnostics` доступна read-группе, но не содержит секретов.
6. В JSON ACL нет повторяющихся ключей объектов; каждая группа объявляет объект один раз.
7. `vpnmanager.dns`, `vpnmanager.blocklists`, `vpnmanager.ipv6` в ACL отсутствуют.

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
| Разделение прав в реальной сессии LuCI (включая `logs.set_rotation`) | [Т] T-05 / W-11 |
| Синтаксис ACL для новых объектов `logs`/`maintenance`/`audit` | [Т] W-11 |

---

## 9. Идентичность вызывающего субъекта (ТЗ §4, решение №9; OQ-11 закрыт 24.11.2026)

**Требование:** нельзя доверять имени пользователя, переданному браузером. Механизм должен быть документирован, а не подразумеваться.

**Решение владельца (OQ-11, вариант А):** идентификация выполняется через действующую rpcd-сессию и системный UID; Go daemon самостоятельно проверяет разрешения; JSON-поле `user` не используется для авторизации и должно игнорироваться; дополнительный daemon token **не вводится**. Схема ниже — утверждённая архитектура; фактическая передача identity на целевой прошивке остаётся предметом технической проверки C3-13 (W-11).

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
5. Механизм считается **неподтверждённым** до W-11 (C3-13): до его прохождения нельзя утверждать, что API защищён, и **функциональную реализацию API не начинать**. Если rpcd не передаёт identity надёжно, молчаливый переход к более слабой схеме запрещён — требуется отдельное решение владельца.
6. `VM_ERR_UNSUPPORTED` возвращается для `vpnmanager.ipv6.*`, `vpnmanager.dns.*`, `vpnmanager.blocklists.*` и любых методов вне реестра §6.

---

## 10. Ограничения U9, влияющие на контракт

1. **Ручной override и обновление подписки.** Успешное обновление сохраняет override, если выбранный сервер найден в новом наборе по точному совпадению нормализованной конфигурации подключения (№13 §3). Если сервер исчез — `mode` становится `auto`, публикуется `override.reset`. При сохранении `srv_id` перепривязывается к новой записи; публикуется `override.preserved`.
2. **Автообновление.** Первый плановый запуск — по выбранному интервалу; немедленный первый запрос не утверждён. Ручное обновление доступно всегда.
3. **Логи.** Ручной очистки и удаления нет; `set_rotation` меняет только порог, не запускает ротацию немедленно.
4. **Нет собственного здоровья серверов.** API не отдаёт `health`, счётчики отказов, приоритеты, `last_working_server`.
5. **Нет общего журнала событий.** `diagnostics` не имеет `tail`; аудит настроек — отдельная сущность.
