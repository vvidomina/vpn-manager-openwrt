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
const Version = "0.1.0-stage1"

func main() {
	os.Exit(run(os.Args[1:], os.Stderr))
}

func run(args []string, stderr io.Writer) int {
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

	cfg, found, err := config.Load(*cfgPath)
	if err != nil {
		log.Error("конфигурация не загружена", "path", *cfgPath, "error", err)
		return 1
	}
	if !found {
		log.Warn("файл конфигурации не найден, используются значения по умолчанию", "path", *cfgPath)
	}

	store, err := storage.New(*stateDir)
	if err != nil {
		log.Error("каталог данных недоступен", "error", err)
		return 1
	}

	st := state.NewStore()
	c := core.New(cfg, log, st)
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

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := c.Start(ctx); err != nil {
		log.Error("запуск не выполнен", "error", err)
		return 1
	}
	log.Info("демон запущен в foreground", "state_dir", store.Root(), "state", string(c.Status().State))

	svc := api.New(c, Version)
	resp := svc.Call(ctx, api.Request{Method: api.MethodStatus})
	log.Debug("граница API доступна", "method", api.MethodStatus, "ok", resp.OK)

	<-ctx.Done()
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
