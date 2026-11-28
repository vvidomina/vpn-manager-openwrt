# Документ №51. Очистка Xiaomi AX3000T от beta v1 и старых VPN-компонентов

Дата: 28.11.2026.
Основание: задание №55.
Результат: старые компоненты beta v1, Passwall2, Zapret, V2RayA, Xray и подтверждённые остатки Forkop/Bridger удалены с роутера. U9 не устанавливалась, новые VPN-компоненты и Xray не ставились, VPN-трафик не тестировался.

## 1. Устройство и окружение

| Параметр | Значение | Источник |
|---|---|---|
| Устройство | Xiaomi Mi Router AX3000T (`xiaomi,mi-router-ax3000t`) | `/tmp/sysinfo/board_name`, `model` |
| OpenWrt | 25.12.6 r33311-981c978eff, `mediatek/filogic`, `aarch64_cortex-a53` | `/etc/openwrt_release` |
| Ядро | Linux 6.12.112 aarch64 | `uname -a` |
| Пакетный менеджер | apk-tools 3.0.5 | `apk --version` |
| Инвентаризация | 28.11.2026, 12:17 MSK | `date` на устройстве |
| Очистка | 28.11.2026, 12:19–12:23 MSK | журнал команд |
| Доступ | SSH `root@192.168.1.1`, ключ `~/.ssh/id_ax3000t` | `~/.ssh/config` |
| Перезагрузка | **не выполнялась**; uptime 352 286 → 352 408 с (та же загрузка) | `/proc/uptime` |

## 2. Резервная копия

| Параметр | Значение |
|---|---|
| Команда | `sysupgrade -b /tmp/vpnmanager-beta-backup-2026-11-28.tar.gz` |
| Размер | 17 082 байта, 57 записей (включая `etc/config/network`, `etc/config/dhcp`, `etc/config/xray`, `etc/xray/*`) |
| SHA-256 на роутере | `8206f83d5d5d128e4668ecb651b1784a86ebee112bb208c3e36c8f6ee5c05a79` |
| SHA-256 локальной копии | `8206f83d5d5d128e4668ecb651b1784a86ebee112bb208c3e36c8f6ee5c05a79` (совпадает) |
| Местоположение вне роутера | `C:\Users\user\vpn-manager-router-backups\vpnmanager-beta-backup-2026-11-28.tar.gz` |
| В Git / репозитории | не помещена; в публикацию и отчёт не включается |

`scp` не сработал: на устройстве отсутствует `/usr/libexec/sftp-server`. Копия перенесена потоком `ssh ax3000t 'cat <файл>' > <локальный файл>`; размер и SHA-256 сверены. После сверки копия в `/tmp` роутера удалена, локальная сохранена.

Резервная копия содержит секреты (конфигурация, ключи, данные подключений) — её содержимое не выводилось, не копировалось в репозиторий и не цитируется.

## 3. Инвентаризация до очистки

### 3.1. Пакеты

| Пакет | Состояние |
|---|---|
| `xray-core` | **установлен**, `26.10.21-r1` (`aarch64_cortex-a53`) |
| `passwall2`, `luci-app-passwall2` | не установлены |
| `zapret`, `zapret2` | не установлены |
| `v2raya`, `luci-app-v2raya` | не установлены |
| `forkop`, `bridger` | не установлены |

Всего установленных пакетов до очистки — 183. Обратные зависимости `xray-core`: отсутствуют (`xray-core-26.10.21-r1 is required by:` — пусто); `apk del --simulate` удалял только сам пакет (1/1).

Файлы, принадлежащие пакету `xray-core` (`apk info -L`): `/etc/config/xray`, `/etc/init.d/xray`, `/etc/xray/config.json.example`, `/usr/bin/xray`, `/lib/apk/packages/xray-core.*`, `/lib/upgrade/keep.d/xray-core`.

Файлы **без владельца-пакета** (`apk info -W` → «Could not find owner package»): все beta-скрипты, `/usr/share/xray/{geoip,geosite}.dat`, `/etc/config/{forkop,bridger,v2raya,zapret.tmp}`, `/usr/lib/xray-setup/*.uc`, `/root/state.log`, `/root/.xray`, `/root/xray-backups`.

### 3.2. Состояние системы до очистки

- Процессы старых компонентов: не запущены. Слушающих сокетов старых сервисов нет.
- nftables: единственная таблица — `inet fw4`. Таблицы `inet xray` нет, правил TPROXY и mark `0x400000` нет.
- `ip rule`: только штатные `0/32766/32767`. Таблицы 777 и 1000 в ядре отсутствуют, маршрутов нет.
- `/etc/iproute2/rt_tables`: присутствовала запись `100 zeroblock` (остаток Zapret).
- Firewall UCI: две include-секции Passwall2 — `firewall.passwall2` и `firewall.passwall2_server`, обе ссылались на отсутствующие файлы `/var/etc/passwall2.include` и `/var/etc/passwall2_server.include`.
- Cron: единственная строка — закомментированная `#17 */12 * * * /usr/sbin/xray-update >/dev/null 2>&1`.
- Init/rc.d: `/etc/init.d/xray` (пакетный) и `xray-manager`, `xray-monitor`, `xray-watchdog` (без владельца); symlink-ов в `/etc/rc.d` для них не было.
- `/etc/xray`: `DEFAULT.JSON`, `config.json`, `config.json.example`, `emergency`, `outbounds.map`, `rules.json`, `servers.json`, `state`.
- `/usr/share/xray`: `geoip.dat`, `geosite.dat`.
- `/usr/lib/xray-setup`: `build-config.uc`, `gen-rules.uc`, `parse-sub.uc` (ucode-генераторы beta).
- `/root`: `.xray/sub.raw`, `state.log` (143 502 байта, вывод старого монитора), `xray-backups/20261119-103822/` (40 kB: `ip-route-all.txt`, `nft-ruleset.nft`, `crontab.txt`, `etc-config.tar.gz`, `rt_tables.orig`, `ip-rule.txt`, `uci-all.txt`).

Классификация: пакетные файлы — только перечисленные выше файлы `xray-core`; всё остальное из этого перечня — пользовательские beta-скрипты, конфигурация и state; штатные компоненты OpenWrt не затронуты (см. §6).

## 4. Удалено

### 4.1. Через APK

`apk del xray-core` → `(1/1) Purging xray-core (26.10.21-r1)`, освобождено 28.6 MiB, установленных пакетов 183 → 182. Вместе с пакетом удалены `/usr/bin/xray`, `/etc/init.d/xray`, `/etc/xray/config.json.example`, метаданные `/lib/apk/packages/xray-core.*` и `/lib/upgrade/keep.d/xray-core`, конфигурация `/etc/config/xray`.

Pre-deinstall скрипт пакета вывел `ubus call service delete {"name":"xray"} (Not found)` — сервис не был зарегистрирован; на результат удаления не повлияло (exit 0).

### 4.2. Вручную (после проверки владельца: пакетных среди них нет)

| Группа | Удалено |
|---|---|
| Скрипты beta `/usr/sbin` | `xray-manager`, `xray-monitor`, `xray-watchdog`, `xray-dns`, `xray-tproxy`, `xray-update`, `xray-led` |
| Init-скрипты beta | `/etc/init.d/xray-manager`, `/etc/init.d/xray-monitor`, `/etc/init.d/xray-watchdog` |
| Конфигурация и state Xray | `/etc/xray/DEFAULT.JSON`, `config.json`, `emergency`, `outbounds.map`, `rules.json`, `servers.json`, `state`; каталог `/etc/xray` |
| GeoIP/GeoSite | `/usr/share/xray/geoip.dat`, `/usr/share/xray/geosite.dat`; каталог `/usr/share/xray` (единственные ссылки на каталог были в удалённых `/etc/config/xray` и `/etc/init.d/xray`) |
| Beta-генераторы ucode | `/usr/lib/xray-setup/build-config.uc`, `gen-rules.uc`, `parse-sub.uc`; каталог `/usr/lib/xray-setup` (ссылок из других файлов не найдено) |
| Старый state и данные Xray | `/root/state.log`; `/root/.xray` (содержал `sub.raw`); `/root/xray-backups` (7 файлов, 40 kB) |
| Конфиги неустановленных компонентов | `/etc/config/forkop`, `/etc/config/bridger`, `/etc/config/v2raya`, `/etc/config/zapret.tmp` |
| Временные файлы очистки | `/tmp/crontab.root.before`, `/tmp/vpnmanager-beta-backup-2026-11-28.tar.gz` (копия резервной копии на роутере) |

Содержимое `/root/.xray/sub.raw`, `/etc/xray/servers.json` и архивов `xray-backups` не читалось и в отчёт не включается; зафиксированы только имена и размеры.

### 4.3. UCI, cron, маршрутизация

| Элемент | Действие | Проверка |
|---|---|---|
| `firewall.passwall2`, `firewall.passwall2_server` | удалены `uci delete`, затем `uci commit firewall` | `uci show firewall \| grep -i passwall` — пусто |
| Cron-строка `xray-update` | удалена строка (была закомментирована); сам `/etc/crontabs/root` сохранён | файл существует, 0 строк; других записей не было |
| `/etc/iproute2/rt_tables`: `100 zeroblock` | строка удалена после проверки: маршрутов в таблице 100 нет, `ip rule` на неё не ссылается, в nftables ссылок нет, единственное упоминание в файлах — комментарий удалённого `xray-tproxy` | `grep zeroblock` — пусто; штатные записи `prelocal/local/main/default/unspec` на месте |
| `/var/run/procd_xray*.lock` | к моменту очистки уже отсутствовали (procd снял их вместе с сервисами) | — |
| `/var/etc/passwall2.include`, `passwall2_server.include` | отсутствовали до очистки | — |

Firewall **не перезапускался**, `nft` не сбрасывался, `inet fw4` не изменялась. Изменение UCI вступит в силу при следующей штатной перезагрузке firewall; на текущее состояние `inet fw4` это не влияет.

## 5. Проверки после очистки (§10 задания)

| № | Проверка | Результат |
|---|---|---|
| 1 | Пакет `xray-core` отсутствует | подтверждено: `apk info -e xray-core` → absent; установленных пакетов 182 |
| 2 | Пакеты Passwall2 / Zapret / V2RayA отсутствуют | не были установлены; `apk list --installed \| grep -Ei 'xray\|v2ray\|zapret\|passwall\|forkop\|bridger\|sing-box'` — пусто |
| 3 | Конфиги Forkop/Bridger и подтверждённые остатки удалены | подтверждено: файлов нет, ссылок нет |
| 4 | Старые Xray beta-скрипты и пользовательские init-скрипты отсутствуют | подтверждено: `/etc/init.d`, `/etc/rc.d`, `/usr/sbin`, `/usr/bin`, `/usr/lib/xray-setup` — совпадений нет |
| 5 | Старые конфигурации и state Xray отсутствуют | подтверждено: `/etc/config/xray`, `/etc/xray`, `/root/state.log`, `/root/.xray`, `/root/xray-backups` отсутствуют |
| 6 | GeoIP/GeoSite отсутствуют | подтверждено: `/usr/share/xray` удалён |
| 7 | Старые cron, include, route- и nft-следы удалены | cron-строка, две include-секции и `zeroblock` удалены; nft- и route-следов не было |
| 8 | Нет процессов и слушателей старых сервисов | подтверждено: совпадений в `ps`/`netstat` нет |
| 9 | Штатный `inet fw4` сохранён | `nft list tables` → только `table inet fw4`; чтение таблицы без ошибок |
| 10 | `network.wan.type='bridge'` без изменений | `uci -q get network.wan.type` → `bridge`; `/etc/config/network` — 41 строка, файл не изменялся |
| 11 | Три DNS-записи без изменений | `/etc/config/dhcp`, строки 25–27: `/mask.icloud.com/`, `/mask-h2.icloud.com/`, `/use-application-dns.net/`; это единственные `server` у dnsmasq |
| 12 | Сеть и административный доступ сохранены | WAN `up`, default route через `pppoe-wan`, `ping 1.1.1.1` — ok, DNS `example.com` через 127.0.0.1 — отвечает, LuCI `http://127.0.0.1/` — ok, SSH :22 слушает |

Финальный поиск ссылок `grep -rln -iE 'xray|passwall|zapret|v2ray|forkop|bridger' /etc /usr/sbin /usr/lib /www /root /lib/upgrade` — совпадений нет. В `/www`, `/usr/share/luci/menu.d`, `/usr/share/luci/acl.d`, `/usr/share/rpcd` страниц и пунктов меню старых компонентов не было.

## 6. Что сохранено намеренно

| Элемент | Причина |
|---|---|
| `/etc/config/network` (включая `wan.type='bridge'`), `/etc/config/wireless`, интерфейсы, маршруты, DNS-настройки | прямо запрещено менять заданием №55 §9 |
| Три записи `/mask.icloud.com/`, `/mask-h2.icloud.com/`, `/use-application-dns.net/` | отдельное решение владельца |
| `firewall.@rule[8]` (`target='REJECT'`) и остальные правила fw4 | происхождение не доказано как остаток старого компонента; правило не тронуто |
| `/etc/config/dhcp.apk-new`, `/etc/config/luci.apk-new`, `/etc/config/uhttpd.apk-new` | промежуточные файлы apk, к старым VPN-компонентам не относятся |
| `/var/etc/dnsmasq.conf.cfg01411c`, `/var/etc/uhttpd` | штатные runtime-файлы dnsmasq и uhttpd |
| Штатные init-скрипты, LuCI, rpcd, dnsmasq, dropbear, uhttpd, `firewall4`/`nftables`, kernel-модули | системные компоненты OpenWrt |
| Репозиторий `vpn-manager-openwrt`, документы №00–№50, `router/audit/audit.sh`, `stage1/*`, `.freebuff.zip` | вне задачи очистки, не изменялись |

## 7. Не выполнялось

- Перезагрузка роутера, перезапуск firewall, `nft flush`, сброс маршрутизации — запрещено заданием.
- Установка U9/IPK/нового Xray, настройка VPN, тесты VPN-трафика — вне задания.
- Изменение `network.wan.type`, интерфейсов, адресов, маршрутов, DNS-записей — запрещено.
- Чтение и копирование содержимого старых конфигураций, ключей и подписок — не выполнялось.

## 8. Ограничения и замечания

1. `scp` недоступен (нет `sftp-server`); копия резервной копии снята потоком через SSH, целостность подтверждена SHA-256.
2. `apk` выводил предупреждения об отсутствующих файлах кеша индексов (`packages.adb`) — это состояние кеша до очистки; на локальные операции `info`/`del` не влияло.
3. `diff` на роутере отсутствует (busybox), поэтому сравнение crontab выполнено по числу строк и содержимому до/после, а не утилитой `diff`.
4. До очистки не снимался отдельный замер свободного места по `/overlay`; освобождение оценено по сообщению apk (28.6 MiB) и удалённым каталогам. После очистки `/overlay` — 55 724 kB свободно.
5. UCI-изменение (удаление include-секций Passwall2) закоммичено, но firewall не перезапускался по требованию задания; фактическое применение произойдёт при следующей штатной перезагрузке firewall.
6. Проверка рендера LuCI не выполнялась: пунктов меню и страниц старых компонентов в файловой системе не обнаружено.
7. Сетевые функции U9 (перехват, маршрутизация, Observatory, `flock`) на роутере не проверялись — это задачи последующих этапов.

## 9. Итог

Роутер очищен от beta v1 и старых VPN-компонентов; `xray-core` удалён через APK, все подтверждённые остатки удалены, неподтверждённые элементы и системные компоненты сохранены. WAN (`type='bridge'`, PPPoE), три DNS-записи, `inet fw4`, административный доступ и связь с интернетом сохранены, перезагрузка не выполнялась.

Это очистка, а не установка U9: новых VPN-компонентов и Xray на роутере нет, VPN-функциональность не проверялась.
