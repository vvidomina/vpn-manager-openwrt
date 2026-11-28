# Документ №50. Задание №54: финальные исправления каркаса по результатам ревью

Дата: 28.11.2026.
Основание: задание №54 (три замечания повторного ревью задания №53).
Область: Go-каркас этапа 1 (№44), граница API (№12), LuCI Overview (№11 §13), матрица готовности (№45).
Результат задания — закрытие трёх замечаний ревью. Это **не** готовность VPN-функциональности, IPK или установки на роутер.

## 1. Исходное состояние

| Проверка | Команда | Результат |
|---|---|---|
| Ветка | `git rev-parse --abbrev-ref HEAD` | `main` |
| HEAD до работы | `git rev-parse HEAD` | `be4682354d15e45f157d33a224f966018db342de` |
| Рабочее дерево | `git status --porcelain` | пусто |
| Синхронизация | `git fetch origin`, `git rev-parse main origin/main` | `main = origin/main = be4682354d15e45f157d33a224f966018db342de` |
| Ветка `architecture` | `git rev-parse architecture` | `fb0baad5dd4fa44a3dba270670b99a7a15379aec` (не изменялась) |
| Удалённые ветки | `git branch -a` | только `origin/main`; временные ветки не создавались |

Неожиданных изменений не было; работа велась в `main`, force push не использовался.

## 2. Выполненные исправления

### 2.1. Makefile: каталог сборки создаётся до компилятора

Причина замечания: цели `build` и `cross` писали в `$(OUTDIR)` (`build/`), но каталог не создавали. В чистом клоне репозитория `build/` отсутствует, и `$(GO) build -o build/vpnmand` завершается ошибкой.

Изменение (только две строки в рецептах, без изменения назначения и параметров сборки):

```make
build:
	@mkdir -p $(OUTDIR)
	$(GO) build -o $(OUT) $(PKG)

cross:
	@mkdir -p $(OUTDIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build -o $(OUTDIR)/$(BIN)-linux-arm64 $(PKG)
```

`clean` не менялся (`rm -rf $(OUTDIR)`), поэтому артефакты по-прежнему удаляются штатной целью. Новых зависимостей и файлов сборки в Git не добавлено.

### 2.2. Удаление неподтверждённого RPC-метода `vpnmanager.version`

Метода `version` нет в реестре №12 §6: объект `vpnmanager` объявляет `status`, `enable`, `disable`, `reload`, а статус длительных операций описан отдельным объектом `vpnmanager.operation`. Регистрация метода вне реестра запрещена (№44 §3.5), поэтому метод удалён, а не заменён другим.

Удалено/изменено:

| Файл | Изменение |
|---|---|
| `internal/api/api.go` | убраны `MethodVersion`, тип `VersionResult`, поле `version` в `Service`, ветка `case MethodVersion` в `Call`; `New(core Core, version string)` → `New(core Core)`; в комментарии к реестру методов зафиксировано, что версия публикуется только CLI-флагом и в diagnostics |
| `internal/api/api_test.go` | тест `TestVersion` (проверял успешный ответ с версией) заменён на `TestVersionMethodNotRegistered`: вызов `vpnmanager.version` обязан вернуть `VM_ERR_UNSUPPORTED` без результата; вызовы `New` приведены к новой сигнатуре |
| `cmd/vpnmand/main.go` | `api.New(c, Version)` → `api.New(c)`. Константа `Version` сохранена: она используется в `-version`, в журнале и в diagnostics |
| `luci/htdocs/luci-static/resources/view/vpnmanager/overview.js` | удалён `rpc.declare` для `version`; `load` возвращает только `vpnmanager.status`; строка «Версия» удалена — вымышленное значение не подставляется |
| `luci/ucode/controller.uc` | удалён `callVersion` и строка «Версия»; контроллер читает только `status` |
| `luci/acl.d/luci-app-vpnmanager.json` | из списка методов чтения объекта `vpnmanager` удалён `version` |

LuCI Overview остаётся работоспособной без удалённого вызова: единственный запрос — `vpnmanager.status`; при недоступности ubus-объекта показывается предупреждение «Демон VPN Manager недоступен: ubus-объект vpnmanager не зарегистрирован», а не успешный статус. Данные берутся только из полей ответа `status` (`state`, `reason`, `config`, `config_error`, `ready`, `observed_at`).

### 2.3. RD-10 в документе №45

Было: `main` = `d56a71f` — читалось как текущая вершина `main`.

Стало: строка различает **историческую исходную точку U9** `d56a71f8b329f6d803db9863cae0e7a18aa7b0f4` (первая редакция №00) и **опубликованный результат заданий №52–53** `be4682354d15e45f157d33a224f966018db342de` (на момент начала задания №54), с явной оговоркой, что это не вершина `main` и последующие изменения отражаются в Git history. SHA коммитов задания №54 в документ заранее не вписаны.

## 3. Изменённые файлы

**Коммит 1 — код и Makefile** (`fix(stage1): correct build and API registry`), 7 файлов, +35/−61:

- `Makefile`
- `cmd/vpnmand/main.go`
- `internal/api/api.go`
- `internal/api/api_test.go`
- `luci/acl.d/luci-app-vpnmanager.json`
- `luci/htdocs/luci-static/resources/view/vpnmanager/overview.js`
- `luci/ucode/controller.uc`

**Коммит 2 — документация** (`docs(stage1): close task 54 review findings`), 2 файла:

- `analysis/45-implementation-readiness-matrix.md`
- `analysis/50-task-54-final-review-fixes-report.md`

Другие компоненты не менялись: контракты №11, реестр №12, состояния, Storage, lifecycle, txn и заглушки не затронуты.

## 4. Результаты проверок

Окружение: Windows 10/11, Git Bash, `go version go1.27.4 windows/amd64` (`C:\Program Files\Go\bin\go.exe`), Node.js доступен. Python не использовался.

| № | Проверка | Команда | Результат |
|---|---|---|---|
| 1 | Форматирование | `gofmt -l .` | пусто (файлов, требующих форматирования, нет) |
| 2 | Тесты | `go test -count=1 ./...` | exit 0; 15 пакетов — `ok` |
| 3 | Статический анализ | `go vet ./...` | exit 0, замечаний нет |
| 4 | Кросс-сборка | `CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o /tmp/vpnmand-linux-arm64 ./cmd/vpnmand` | exit 0; 4 011 970 байт; файл удалён из `/tmp` |
| 5 | Синтаксис LuCI Overview | обёртка файла в функцию и `node --check` | exit 0 (формат LuCI-модуля: `'require view'` + top-level `return`, поэтому проверяется обёрнутая копия во временном каталоге) |
| 6 | Синтаксис ucode-контроллера | та же обёртка + `node --check` | exit 0 (проверен только синтаксис JS-подобного файла; ucode не исполнялся) |
| 7 | JSON ACL | `node -e 'JSON.parse(...)'` | `acl_json_ok read={"vpnmanager":["status","operation"]} write={"vpnmanager":["operation"]}` |
| 8 | Методы API по реестру №12 | `grep -n "Method[A-Z][a-zA-Z]* *=" internal/api/api.go` | только `MethodStatus` и `MethodOperation` |
| 9 | Отсутствие ссылок на метод версии | `grep -rn -e "MethodVersion" -e "vpnmanager\.version" .` | в коде, ACL и LuCI — только отрицательный тест; остальные совпадения — исторические отчёты №48/№49 (см. §8) |
| 10 | Ссылки `callVersion` / `method: 'version'` в `luci`, `internal`, `cmd` | `grep -rn ...` | совпадений нет (exit 1) |
| 11 | Неизвестный метод | `go test -run 'TestUnknownMethod|TestVersionMethodNotRegistered' ./internal/api` (в составе п.2) | оба теста проходят: ответ `VM_ERR_UNSUPPORTED`, `Result` пуст |
| 12 | CLI `-version` | `go build -o /tmp/vpnmand.exe ./cmd/vpnmand && /tmp/vpnmand.exe -version` | stderr: `vpnmand 0.1.1-stage1`, exit 0 |
| 13 | Чистое состояние перед сборкой | `rm -rf build; ls -d build` | каталога нет (exit 2) |
| 14 | Рецепт `build` построчно | `mkdir -p build && go build -o build/vpnmand ./cmd/vpnmand` | exit 0; `build/vpnmand` 4 389 376 байт |
| 15 | Рецепт `cross` построчно | `mkdir -p build && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o build/vpnmand-linux-arm64 ./cmd/vpnmand` | exit 0; 4 011 970 байт |
| 16 | Рецепт `clean` | `rm -rf build; ls -d build` | каталог удалён (exit 2) |
| 17 | Структура Makefile | `awk '/^\t/{c++} END{print c}' Makefile`; `grep -n '^    ' Makefile` | 9 строк рецептов с табуляцией, строк с пробельным отступом нет |
| 18 | Пробельные ошибки в диффе | `git diff --check` | exit 0 |
| 19 | Бинарники и `build/` в коммите | `git status --porcelain`, `git diff --cached --name-status` | в коммите только перечисленные в §3 файлы; `build/` отсутствует |

## 5. Проверки, которые не удалось выполнить

| Проверка | Причина |
|---|---|
| `make build`, `make cross`, `make clean` фактическим запуском | GNU Make в окружении отсутствует: `make --version` → `make: command not found`, exit 127. Устанавливать Make только ради задания не требовалось. Вместо этого рецепты выполнены построчно (пп.14–16) из состояния, где `build/` нет, а структура рецептов проверена отдельно (п.17). Сами Make-цели не запускались |
| `go test -race` | `CGO_ENABLED=0`, компилятор `gcc` недоступен (`cgo: C compiler "gcc" not found`). Системное окружение не изменялось |
| Исполнение `controller.uc` | Интерпретатор `ucode` не установлен; файл не используется вью и проверен только на синтаксис |
| Загрузка LuCI Overview в браузере/на роутере | Требует LuCI, rpcd и ubus; транспорт ubus в этапе 1 не реализован. Проверены синтаксис и отсутствие ссылок на удалённый метод, не фактический рендер |
| `ubus`/`procd`/`flock`/AX3000T | Вне этапа 1 и вне задания №54; на устройстве ничего не запускалось и не менялось |

## 6. Проверка отсутствия неподтверждённого RPC-метода

- Константы методов в `internal/api`: `MethodStatus = "vpnmanager.status"`, `MethodOperation = "vpnmanager.operation"`. Обе соответствуют реестру №12 §6.
- `MethodVersion` и тип `VersionResult` в репозитории отсутствуют.
- Успешного ответа на `vpnmanager.version` не существует: метод попадает в ветку `default` и возвращает `{ok:false, error:{code:"VM_ERR_UNSUPPORTED"}}`. Это закреплено тестом `TestVersionMethodNotRegistered`, который проверяет и код ошибки, и отсутствие результата.
- LuCI Overview, ucode-контроллер и ACL не содержат ссылок на метод версии; ACL разрешает чтение `status` и `operation` и запись `operation`.
- CLI-флаг `vpnmand -version` сохранён и печатает `vpnmand 0.1.1-stage1` (exit 0) — это флаг точки входа, а не RPC-метод.

## 7. Соответствие контрактам №11, №12 и плану №44

- №12 §2, §6: набор методов сокращён до утверждённого; новых методов не добавлено.
- №11 §13, №44 §3.8: Overview показывает данные только из `status` и честное сообщение при недоступности ubus.
- №44 §3.5: метод вне реестра не регистрируется, а вызывается как неизвестный → `VM_ERR_UNSUPPORTED`.
- №44 §5.2: цель установки на роутер не появлялась; Makefile по-прежнему только собирает и проверяет.

## 8. Известные ограничения (вне задания №54)

1. **Исторические отчёты №48 и №49** содержат упоминание `vpnmanager.version` как описание состояния на момент заданий №52/№53. Они не переписаны: задание запрещает редактировать старые отчёты так, будто новые проверки выполнялись в прошлом. Актуальное состояние зафиксировано в настоящем отчёте.
2. **Гранулярность ACL.** Реестр №12 §6 описывает `vpnmanager.operation` отдельным объектом, тогда как ACL каркаса перечисляет методы `status` и `operation` внутри объекта `vpnmanager` (решение задания №53). Задание №54 требует удалить только метод версии; перестройка объектов ACL не выполнялась и остаётся открытым вопросом для следующего ревью.
3. **`build/` не внесён в `.gitignore`.** Каталог сборки удаляется целью `clean`, в коммиты не попадал (п.19), но правило игнорирования потребовало бы правки файла вне разрешённого перечня задания.
4. **Не покрыто этапом 1:** ubus-транспорт, персистентность конфигурации, Xray Manager, nftables/TPROXY, IPK, установка на AX3000T. Паника в собственной goroutine компонента по-прежнему не перехватывается (RI-04).

## 9. Коммиты задания №54

| Коммит | SHA | Содержимое |
|---|---|---|
| 1 | `6e7e69241b68665dc2506ce93941d53a659d108c` | `fix(stage1): correct build and API registry` — Makefile, API, LuCI |
| 2 | см. `git log` / сообщение приёмки | `docs(stage1): close task 54 review findings` — №45 RD-10, настоящий отчёт |

SHA второго коммита не может быть указан в нём самом: он становится известен после создания коммита и фиксируется в Git history и сообщении о приёмке.
