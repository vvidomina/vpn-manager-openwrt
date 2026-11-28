# Отчёт №49 — устранение замечаний ревью Go-каркаса U9 (этап 1)

Дата: 28.11.2026
Задание: №53
Репозиторий: `https://github.com/vvidomina/vpn-manager-openwrt`
Ветка: `main`
Основание: документы №10, №11, №44, №45, №48; план этапа 1 и контракты компонентов

Область: только подготовительный Go-каркас этапа 1. VPN-функциональность, Xray, nftables, DNS, маршрутизация, firewall, ubus-транспорт, IPK и работа на AX3000T в задании не реализовывались.

## 1. Исходное состояние и Git

| Параметр | Значение |
|---|---|
| Исходный HEAD | `a5343dc292c4bf50de5ff2c23a018c12bf2d856d` (`feat(stage1): add Go daemon skeleton`, задание №52) |
| Предыдущий коммит | `7a4d34c40f61162af5e9d74a6a9088c1ad3e3ad2` — документация U9 |
| Проверка перед работой | `git status --short` — пусто; ветка `main`; `git fetch origin`; `git rev-parse HEAD origin/main` — оба `a5343dc` |
| Локальная ветка `architecture` | `fb0baad` — сохранена без изменений |
| Расхождения истории | отсутствовали; чужие изменения не перезаписывались |
| Force push, переписывание истории, временные удалённые ветки | не выполнялись |

## 2. Исправленные замечания

### 2.1 Контракты компонентов (§2 задания)

Приведены к документу №11; сокращённые методы с другой семантикой удалены.

| Компонент | Было | Стало (№11) |
|---|---|---|
| Policy | `Apply(Intent)`, `Verify`, `Remove` | `Stage(Desired) (Plan, error)`, `Apply(Plan) (Applied, error)`, `Rollback(Applied) error`, `Remove`, `Status() (Status, error)` (§3) |
| Devices | `List(ctx)`, `SetExcluded(mac, bool)` | `List(ctx, Filter)`, `SetMode(ctx, mac, Mode)` (`default`/`always_direct`), `Merge(ctx, macs)`, `Resolve(ctx) ([]Assignment, error)` (§7) |
| Subscriptions | `List`, `Refresh(id)` | `Add(Source)`, `Update(id) (Report, error)`, `Remove`, `Reorder(ids)`, `List` (§8) |
| Profiles | `List`, `Activate(id)` | `Save(Profile)`, `Assign(serverID, profileID)`, `Compile(Profile) (Ruleset, error)`, `SetPending(profileID)`, `List` (§9) |
| Transactions | `Run(steps []Step)` | `Begin(Request) (Operation, error)`, `Status(opID)`, `Rollback(opID, reason)`, `Recover() (RecoveryReport, error)`; фазы P0–P8 и состояния операции (§11, §10 §7.1) |

Все методы заглушек возвращают `VM_ERR_UNSUPPORTED` и не сообщают о выполненной работе: `Plan`/`Applied`/`Status` пустые, списки `nil`, `Ruleset` без правил. Побочных системных действий нет. Типобезопасность сохранена: типы аргументов и результатов объявлены в пакетах, строковых «команд» нет.

Модель фаз: `P0_PREPARE`, `P1_VALIDATE`, `P2_PREFLIGHT_OUTBOUND`, `P3_STAGE_POLICY`, `P4_START_ENGINE`, `P5_APPLY_POLICY`, `P6_TRIAL`, `P7_COMMIT`, `P8_ROLLBACK`; `Phases()` отдаёт копию последовательности применения (P0–P7), `Phase.Valid()` распознаёт P8.

### 2.2 Невалидная конфигурация (§3.1)

Было: `config.Load` с ошибкой → `return 1`, демон завершался.

Стало:

- демон остаётся запущенным в foreground; `main` логирует ошибку и продолжает работу;
- состояние публикуется как `DEGRADED/CONFIG_STALE`, ось конфигурации — `config=Invalid` с текстом причины;
- `Snapshot`/`Report` содержат `config`, `config_error`; `api.status` отдаёт `config` и `config_error`;
- готовность (`Ready`) не публикуется; `Submit` возвращает `VM_ERR_INVALID_STATE` с причиной — операции изменения недоступны;
- `ACTIVE` при `config=Invalid` отклоняется в `Snapshot.Validate` (дополнительно к требованию S2–S4/S6);
- компоненты при невалидной конфигурации не запускаются;
- отсутствие файла конфигурации остаётся штатным первым запуском со значениями по умолчанию (`intent=disabled`).

Новые состояния и подпричины не вводились: использованы `DEGRADED` и подпричина `CONFIG_STALE` из №10 §5.1. Ось `config` — первая ось модели №10 §5.1, а не новое состояние сервиса.

Тест `TestInvalidConfigFails` (ожидал код выхода 1) заменён на `TestInvalidConfigKeepsDaemonRunning` (код 0, `VM_ERR_VALIDATION` в журнале, `DEGRADED`, `ready=false`).

### 2.3 Паники компонентов (§3.2)

- Паника синхронного `Start`/`Stop` компонента перехватывается на границе Core (`safeStart`/`safeStop`) и превращается в `VM_ERR_INTERNAL` с текстом «паника компонента …».
- При панике старта уже запущенные компоненты останавливаются в обратном порядке, публикуется `DEGRADED` с причиной; демон не завершается.
- При панике остановки остальные компоненты останавливаются, публикуется `DEGRADED`, а не «чистое» `UNKNOWN`.
- Ограничение зафиксировано в godoc: `recover` действует только в той goroutine, где произошла паника; паника в собственной goroutine компонента этой границей не перехватывается и не заявляется перехваченной.

Тесты: `TestPanicOnStartDegradesAndRollsBack`, `TestPanicOnStopDegradesButStopsOthers`.

### 2.4 SIGHUP (§3.3)

- `SIGHUP` перехватывается отдельным каналом: демон не завершается и не перезапускается, событие пишется в журнал (`action=ignored`, подсказка про `procd reload`).
- `SIGINT`/`SIGTERM` остаются штатным завершением через `signal.NotifyContext`; в тестах контекст завершается закрытием канала, реальные сигналы процессу не отправляются.
- Поведение соответствует №44 §4.1 п.6: конфигурация на этапе 1 не перезагружается.

Тесты: `TestSIGHUPIsLoggedAndDoesNotStopDaemon`, `TestSIGTERMStopsDaemon`.

### 2.5 LuCI Overview (§4)

Было: `'require vpnmanager.controller as controller'` и обращение к `this.controller.load()`/`this.controller.render()` — у `view` нет поля `controller`, а ucode-модуль не является JS-модулем для клиентского `require`; вью не загружалась.

Стало: `overview.js` самодостаточна — `rpc.declare` для `vpnmanager.status` и `vpnmanager.version`, `L.resolveDefault`, честное сообщение «ubus-объект vpnmanager не зарегистрирован» при недоступности. Показаны состояние, причина, конфигурация и её ошибка, готовность, время наблюдения, версия. Фиктивный успешный статус не подставляется. Транспорт ubus и новые зависимости не добавлялись.

`luci/ucode/controller.uc` сохранён как серверный контроллер для этапа подключения ubus; он не импортируется JS-вью, поскольку ucode нельзя подключить клиентским `require`. Это ограничение отмечено в §9.

### 2.6 Storage и загрузка конфигурации (§5)

- `internal/config` больше не открывает файлы: удалён `os.Open`, введена минимальная абстракция `config.Reader` (`Read(name) ([]byte, error)`), которую реализует `*storage.Store`.
- `config.Load(r Reader, name string)` читает через хранилище; `VM_ERR_NOT_FOUND` трактуется как штатный первый запуск, прочие ошибки — `VM_ERR_STORAGE` и не маскируются под отсутствие файла.
- Циклической зависимости нет: `config` объявляет интерфейс, `storage` его реализует; обратного импорта `storage → config` не появилось. Таблица №44 §2.2 указывала зависимость `storage → config`, которая была бы циклической при `config → storage`; выбрана абстракция чтения — отклонение зафиксировано в §4.
- Абсолютные пути и `..` по-прежнему отклоняются `storage.Path`; строгая валидация и тесты путей сохранены.
- Запись конфигурации на диск не добавлялась: `WriteAtomic` существует и протестирован, но демоном не используется.

Тесты: `TestLoadMissingFile`, `TestLoadInvalidFileIsReported`, `TestLoadStorageFailureIsNotMissingFile`, `TestLoadRejectsEmptyArguments`, `TestLoadThroughStorage` (реальная запись и чтение через `storage.Store`).

### 2.7 Пропуски плана №44 (§6)

| Элемент плана | Статус |
|---|---|
| `Makefile` (§2.1) | добавлен: `build`, `test`, `vet`, `fmt`, `fmt-check`, `cross`, `clean`; целей установки на роутер нет |
| `files/vpnmanager.config` (§2.3) | добавлен: UCI-файл с `enabled '0'`; демон этапа 1 его не читает (конфигурация — JSON), это отмечено в самом файле |
| API health-ping (§2.2) | реализован как внутренняя проверка `api.Service.Health`: читает снимок, возвращает `ok/state/config/ready/reason/observed_at`; успех невозможен при `config=Invalid`, `DEGRADED`, `UNKNOWN`, `EMERGENCY_DIRECT` и при неготовом ядре |
| Фазы P0–P8 (§2.2) | реализованы как перечисление и последовательность в `internal/txn` |
| Транспорт ubus | не реализуется: требует `libubox`/cgo, что выходит за границы этапа и запрещено заданием |

Health-ping **не** зарегистрирован как метод ubus: в реестре №12 §6 такого метода нет, а регистрация метода вне реестра запрещена (№44 §3.5). Это внутренняя проверка границы API, а не новое публичное API.

## 3. Соответствие контрактам №11

| Контракт №11 | Реализация | Примечание |
|---|---|---|
| §3 Policy: `Stage/Apply/Rollback/Remove/Status` | `internal/policy` | `Mode` — `capture`/`direct`/`block`; `Desired` без пользовательских строк-команд |
| §7 Devices: `List/SetMode/Merge/Resolve` | `internal/devices` | режимы `default`/`always_direct` |
| §8 Subscriptions: `Add/Update/Remove/Reorder/List` | `internal/subscriptions` | отчёт импорта `Report` с `Accepted/Rejected/Reasons` |
| §9 Profiles: `Save/Assign/Compile/SetPending/List` | `internal/profiles` | `Restricted` — признак профиля подписки |
| §11 Transactions: `Begin/Status/Rollback/Recover` | `internal/txn` | фазы P0–P8 по №10 §7.1 |
| §1 Core: паника модуля → `DEGRADED` | `internal/core` | перехват синхронных вызовов; см. §2.3 |
| §5 Xray Manager | не изменялся | контракт `Build/Validate/Preflight/Start/Stop/VerifyTraffic/Switch` в задании №53 не перечислен; текущая заглушка остаётся вне `core.Component` — вопрос в §9 |

## 4. Отклонения от плана №44 и их обоснование

1. **`internal/vmerrors` вместо `internal/errors`** — сохранено отклонение №52: имя `errors` перекрывает stdlib.
2. **`config` объявляет `Reader`, `storage` его реализует** — вместо прямого импорта `config → storage` с обратной зависимостью в таблице плана; иначе возникает цикл. Поведение и граница Storage сохранены.
3. **Health-ping — внутренний метод, не ubus-метод** — в реестре №12 метода нет (см. §2.7).
4. **`files/vpnmanager.config` не читается демоном** — план предусматривает файл как стартовую UCI-конфигурацию пакета; потребителей UCI в этапе 1 нет.
5. **LuCI-вью не использует `controller.uc`** — ucode-модуль не подключается клиентским `require`; контроллер сохранён для этапа ubus.
6. **Снимок состояния по-прежнему не сохраняется на диск** — персистентность в этапе 1 не определена (№48 §6).
7. **Singleton/`flock`, ubus, procd на устройстве** — не реализовывались и не проверялись (№45 RT).

## 5. Изменённые файлы

Коммит кода и тестов (25 файлов, +1537/−229):

| Файл | Изменение |
|---|---|
| `internal/policy/policy.go`, `policy_test.go` | контракт §11 §3, тесты всех операций |
| `internal/devices/devices.go`, `devices_test.go` | контракт §7, режим Always DIRECT |
| `internal/subscriptions/subscriptions.go`, `subscriptions_test.go` | контракт §8 |
| `internal/profiles/profiles.go`, `profiles_test.go` | контракт §9 |
| `internal/txn/txn.go`, `txn_test.go` | контракт §11, фазы P0–P8 |
| `internal/config/config.go`, `config_test.go` | `Reader`, `Load` через Storage, тесты |
| `internal/state/state.go`, `state_test.go` | ось `config`, `DEGRADED/CONFIG_STALE`, инвариант `ACTIVE` |
| `internal/core/core.go`, `core_test.go` | паники, диагностический режим, блокировка операций |
| `internal/api/api.go`, `api_test.go` | `config` в статусе, `Health` |
| `internal/diagnostics/diagnostics.go`, `diagnostics_test.go` | предупреждение `config=Invalid` |
| `cmd/vpnmand/main.go`, `main_test.go` | без выхода при невалидной конфигурации, `SIGHUP`, загрузка через Storage, тестовый канал завершения |
| `luci/htdocs/luci-static/resources/view/vpnmanager/overview.js` | самодостаточная вью без `this.controller` |
| `Makefile` | новый |
| `files/vpnmanager.config` | новый |

Коммит документации: `analysis/00-README.md`, `analysis/44-stage1-go-skeleton-plan.md`, `analysis/45-implementation-readiness-matrix.md`, `analysis/48-stage1-go-skeleton-implementation-report.md`, `analysis/49-task-53-review-corrections-report.md` (этот файл).

## 6. Проверки

| Команда | Результат |
|---|---|
| `gofmt -l .` | пусто (расхождений нет) |
| `go vet ./...` | exit 0 |
| `go test ./...` | exit 0; 15 пакетов, 78 тестовых функций |
| `go test -count=1 ./...` | exit 0 (без кэша) |
| `CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build ./cmd/vpnmand` | успешно, 4 011 826 байт; бинарник удалён из `/tmp`, в репозиторий не попал |
| `node --check` (обёртка модуля вокруг `overview.js`) | синтаксис ok |
| `node -e "JSON.parse(...)"` для `menu.d` и `acl.d` | JSON ok |
| `git diff --check` | exit 0 |
| `go test -race ./...` | **не выполнено**: `cgo: C compiler "gcc" not found`, `CGO_ENABLED=0`; окружение не менялось, инструменты не устанавливались |
| `make` | **не выполнено**: `make` в окружении отсутствует; цели продублированы прямыми командами из таблицы |

Go: `go1.27.4 windows/amd64` (интерпретатор `C:\Program Files\Go\bin\go.exe`, `go` вне `PATH`, `gofmt` в `PATH` есть). Тесты выполнялись на Windows (Git Bash); кроссплатформенные особенности путей учтены (`storage.Path` проверяет `/` и `\`).

## 7. Добавленные тесты

- интерфейсные контракты Policy, Devices, Subscriptions, Profiles, Transactions (все пять методов/четыре метода, отказ `VM_ERR_UNSUPPORTED`, отсутствие имитации успеха);
- фазы P0–P8: порядок, копия среза, `Phase.Valid()`;
- невалидная конфигурация: `DEGRADED/CONFIG_STALE`, `config_error`, `ready=false`, блокировка `Submit`, отсутствие запуска компонентов, код выхода 0 вместо 1;
- паника при старте (откат запущенных, `VM_ERR_INTERNAL`, `DEGRADED`) и при остановке (остальные компоненты остановлены, `DEGRADED`);
- `SIGHUP`: журналирование, отсутствие завершения; `SIGINT`/`SIGTERM`-путь через тестовый канал;
- загрузка конфигурации через Storage: отсутствие файла, ошибка валидации с `found=true`, ошибка хранилища ≠ отсутствие файла, пустые аргументы, чтение реального файла через `storage.Store`;
- API health-ping: успех при готовом ядре, отказ при `ready=false`, `config=Invalid`, `DEGRADED`, `VM_ERR_TIMEOUT` при отменённом контексте;
- диагностика: `config=Invalid` в `Warnings`, неизменность снимка.

## 8. Известные ограничения

- `go test -race` не выполнялся: нет cgo и C-компилятора. Гонки не проверялись инструментом; конкурентный доступ покрыт только тестом `Store`.
- `make` не запускался (нет `make`); команды выполнены напрямую.
- `ucode` в окружении отсутствует: `controller.uc` не исполнялся и не линтился. Проверен только JS и JSON LuCI.
- ubus-объект не зарегистрирован: LuCI показывает сообщение о недоступности; сквозной путь UI → ubus → демон не проверялся.
- `flock`/singleton, `respawn 3600 5 5`, procd, foreground-режим, работа на AX3000T не проверялись.
- Ни одна сетевая функция не реализована и не проверялась; IPK не создавался; GitHub Actions не настраивались.
- Динамическое поведение LuCI в браузере не проверялось: выполнена только статическая проверка синтаксиса.

## 9. Оставшиеся открытые вопросы

1. **Контракт Xray Manager (§11 §5) не приведён** к `Build(Selection)/Validate/Preflight/Start/Stop/VerifyTraffic/Switch`: задание №53 перечисляет только пять компонентов. Кандидат на следующее задание; текущая заглушка честно возвращает `VM_ERR_UNSUPPORTED` и не является `core.Component`.
2. **Подпричина для паники компонента.** В №10 §5.1 перечислены `ENGINE_DOWN`, `POLICY_NOT_APPLIED`, `TRAFFIC_UNVERIFIED`, `CONFIG_STALE`, `STALE_SIGNALS`, `RESIDUAL`, `COLD_START`; паника модуля ни одной из них не соответствует. Используется `DEGRADED` с текстовой причиной без изобретённой подпричины — требуется решение владельца, нужна ли отдельная подпричина.
3. **Транспорт ubus** без `libubox`/cgo: способ регистрации объекта остаётся открытым (№48 §9).
4. **Роль `luci/ucode/controller.uc`** после перевода вью на `rpc.declare`: удалить, оставить как заготовку или подключить через серверный rpcd-контроллер — решение не принималось.
5. **`config.Reader` и запись конфигурации**: абстракция чтения минимальна; понадобится ли общий интерфейс чтения/записи для `config` и менеджеров данных — вопрос этапа 2.
6. **Health-ping**: остаётся внутренним. Нужен ли отдельный ubus-метод (и правка реестра №12), если он потребуется procd или LuCI, — открыто.

## 10. Соответствие критериям завершения

| Критерий задания №53 | Статус |
|---|---|
| Интерфейсы соответствуют утверждённым контрактам | выполнено для Policy, Devices, Subscriptions, Profiles, Transactions; Xray — вне объёма (§9.1) |
| Ошибки конфигурации не завершают демон | выполнено, покрыто тестами |
| Lifecycle обрабатывает сигналы и паники | выполнено для синхронных вызовов; `SIGHUP` журналируется |
| В LuCI Overview нет ошибочного обращения к контроллеру | выполнено, страница самодостаточна |
| Конфигурация использует границу Storage | выполнено (`config.Reader` ← `storage.Store`) |
| Элементы плана №44 присутствуют или обоснованы | `Makefile`, `files/vpnmanager.config`, health-ping, фазы P0–P8 добавлены; ubus обоснованно не реализован |
| Тесты и проверки выполнены и зафиксированы | выполнено, кроме `-race` и `make` (§6) |
| Документация синхронизирована | №00, №44, №45, №48 обновлены |
| Коммиты опубликованы в `main` | см. §11 |

## 11. Итоговые SHA

| Коммит | SHA | Состав |
|---|---|---|
| `fix(stage1): align skeleton with approved contracts` | `71f25a1e4fe3e9679f17129f69c32eeb1f5aae23` | 25 файлов, +1537/−229 |
| `docs(stage1): record review corrections` | указывается в сообщении приёмки и `git log` | 5 файлов документации |

SHA коммита документации не может быть вписан в его собственный файл (самоссылка невозможна); он фиксируется в `git log`, в сообщении приёмки и в §2/§5 документа №00. Публикация — в `origin/main` без force push; родительская последовательность: `a5343dc` → `71f25a1` → docs-коммит.

**Результат задания №53 — исправленный Go-каркас этапа 1.** Он не означает готовность VPN, IPK или возможность установки на роутер. После публикации требуется повторное ревью перед переходом к следующему этапу.
