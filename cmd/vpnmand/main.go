// Command vpnmand — демон VPN Manager (этап 1: каркас).
// Работает в foreground, управление процессами оставлено procd.
// Сетевые функции, Xray и nftables не реализуются.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/vvidomina/vpn-manager-openwrt/internal/api"
	"github.com/vvidomina/vpn-manager-openwrt/internal/config"
	"github.com/vvidomina/vpn-manager-openwrt/internal/core"
	"github.com/vvidomina/vpn-manager-openwrt/internal/devices"
	"github.com/vvidomina/vpn-manager-openwrt/internal/logging"
	"github.com/vvidomina/vpn-manager-openwrt/internal/policy"
	"github.com/vvidomina/vpn-manager-openwrt/internal/profiles"
	"github.com/vvidomina/vpn-manager-openwrt/internal/state"
	"github.com/vvidomina/vpn-manager-openwrt/internal/storage"
	"github.com/vvidomina/vpn-manager-openwrt/internal/subscriptions"
	"github.com/vvidomina/vpn-manager-openwrt/internal/txn"
)

// Version — версия каркаса.
const Version = "0.1.1-stage1"

func main() {
	os.Exit(run(os.Args[1:], os.Stderr))
}

// run — обычный запуск демона с обработкой реальных сигналов.
func run(args []string, stderr io.Writer) int {
	return runWith(args, stderr, nil)
}

// runWith выполняет запуск. done != nil используется тестами: закрытие канала
// завершает ожидание вместо SIGINT/SIGTERM. Демон остаётся в foreground.
func runWith(args []string, stderr io.Writer, done <-chan struct{}) int {
	fs := flag.NewFlagSet("vpnmand", flag.ContinueOnError)
	fs.SetOutput(stderr)
	cfgPath := fs.String("config", "/etc/vpnmanager/config.json", "путь к файлу конфигурации")
	stateDir := fs.String("state-dir", "/var/lib/vpnmanager", "каталог данных демона")
	logLevel := fs.String("log-level", "info", "уровень логирования: debug|info|warn|error")
	showVersion := fs.Bool("version", false, "показать версию и выйти")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *showVersion {
		fmt.Fprintf(stderr, "vpnmand %s\n", Version)
		return 0
	}

	level, err := logging.ParseLevel(*logLevel)
	if err != nil {
		fmt.Fprintf(stderr, "vpnmand: %v\n", err)
		return 2
	}
	log := logging.New(stderr, level).With("daemon", "vpnmand", "version", Version)

	// Конфигурация читается через границу Storage: каталог файла — корень
	// хранилища, имя файла — его относительное имя.
	cfgStore, err := storage.New(filepath.Dir(*cfgPath))
	if err != nil {
		log.Error("каталог конфигурации недоступен", "dir", filepath.Dir(*cfgPath), "error", err)
		return 1
	}
	cfg, found, cfgErr := config.Load(cfgStore, filepath.Base(*cfgPath))
	cfgState := state.ConfigSaved
	cfgErrText := ""
	if cfgErr != nil {
		// Невалидная конфигурация не завершает демон: он остаётся запущенным
		// и отдаёт причину через Snapshot/Report (№44 §4.1 п.3).
		cfgState = state.ConfigInvalid
		cfgErrText = cfgErr.Error()
		log.Error("конфигурация невалидна, демон останется в диагностическом режиме",
			"path", *cfgPath, "error", cfgErr)
	} else if !found {
		log.Warn("файл конфигурации не найден, используются значения по умолчанию", "path", *cfgPath)
	}

	store, err := storage.New(*stateDir)
	if err != nil {
		log.Error("каталог данных недоступен", "error", err)
		return 1
	}

	st := state.NewStore()
	c := core.New(cfg, log, st, core.WithConfigState(cfgState, cfgErrText))
	for _, comp := range []core.Component{
		policy.NewStub(),
		devices.NewStub(),
		subscriptions.NewStub(),
		profiles.NewStub(),
		txn.NewStub(),
	} {
		if err := c.Register(comp); err != nil {
			log.Error("компонент не зарегистрирован", "error", err)
			return 1
		}
	}

	ctx, stop := daemonContext(done)
	defer stop()

	if err := c.Start(ctx); err != nil {
		// Диагностический режим: процесс не завершается, состояние DEGRADED
		// уже опубликовано, API остаётся доступным.
		log.Error("ядро не готово, демон остаётся в диагностическом режиме",
			"error", err, "state", string(c.Status().State))
	}
	log.Info("демон запущен в foreground",
		"state_dir", store.Root(), "state", string(c.Status().State), "ready", c.Ready())

	svc := api.New(c)
	if health, hErr := svc.Health(ctx); hErr == nil {
		log.Debug("внутренний health-ping", "ok", health.OK, "state", health.State, "ready", health.Ready)
	}

	waitForSignals(ctx, log, hupChannel(done))
	log.Info("получен сигнал завершения")

	stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := c.Stop(stopCtx); err != nil {
		log.Error("завершение с ошибкой", "error", err)
		return 1
	}
	log.Info("демон остановлен")
	return 0
}

// daemonContext возвращает контекст завершения: SIGINT/SIGTERM в обычном
// режиме или закрытие done в тестах.
func daemonContext(done <-chan struct{}) (context.Context, context.CancelFunc) {
	if done == nil {
		return signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		select {
		case <-done:
			cancel()
		case <-ctx.Done():
		}
	}()
	return ctx, cancel
}

// hupChannel подписывается на SIGHUP в обычном режиме. В тестах сигналы
// реальному процессу не отправляются, поэтому канал не создаётся.
func hupChannel(done <-chan struct{}) <-chan os.Signal {
	if done != nil {
		return nil
	}
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGHUP)
	return ch
}

// waitForSignals ждёт завершения демона. SIGHUP не завершает и не
// перезапускает демон: на этапе 1 конфигурация не перезагружается, событие
// только журналируется (№44 §4.1 п.6). Перезагрузка — через procd reload.
func waitForSignals(ctx context.Context, log logging.Logger, hup <-chan os.Signal) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-hup:
			log.Info("получен SIGHUP: конфигурация на этапе 1 не перезагружается",
				"action", "ignored", "hint", "используйте procd reload")
		}
	}
}
