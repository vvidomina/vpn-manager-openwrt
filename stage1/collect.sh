#!/usr/bin/env bash
# Stage 1: read-only survey of the AX3000T.
# Only reading commands: no config changes, no service restarts, no package installs.
set -u
HOST=ax3000t
OUT=stage1/raw
mkdir -p "$OUT"

sect() {
  num="$1"; name="$2"
  echo "=== [$num] $name ==="
  ssh -o BatchMode=yes -o ConnectTimeout=12 "$HOST" 'sh -s' >"$OUT/${num}-${name}.txt" 2>&1
  rc=$?
  echo "exit=$rc bytes=$(wc -c <"$OUT/${num}-${name}.txt")"
  return 0
}

sect 01 system <<'EOS'
date '+%Y-%m-%d %H:%M:%S %Z'
echo '--- openwrt_release ---'; cat /etc/openwrt_release
echo '--- uname ---'; uname -a
echo '--- board ---'; ubus call system board
echo '--- info ---'; ubus call system info
echo '--- uptime/load ---'; cat /proc/uptime; cat /proc/loadavg
EOS

sect 02 memory-flash <<'EOS'
date '+%Y-%m-%d %H:%M:%S %Z'
echo '--- free ---'; free
echo '--- meminfo ---'; grep -E 'MemTotal|MemFree|MemAvailable|Buffers|Cached|Swap' /proc/meminfo
echo '--- df ---'; df -h
echo '--- overlay top ---'; du -sk /overlay/upper/* 2>/dev/null | sort -rn | head -20
echo '--- mount ---'; mount | grep -E 'overlay|ubi|root'
EOS

sect 03 packages <<'EOS'
date '+%Y-%m-%d %H:%M:%S %Z'
echo '--- apk version ---'; apk --version
echo '--- installed ---'; apk list --installed 2>/dev/null | sort
echo '--- key pkg ---'
for p in xray-core kmod-nft-tproxy nftables dnsmasq dnsmasq-full odhcpd odhcp6c https-dns-proxy jq python3 openssl-util logrotate conntrack luci-app-commands curl wget-ssl iperf3; do
  printf '%s: ' "$p"; apk info -e "$p" 2>/dev/null || echo '(not installed)'
done
EOS

sect 04 services <<'EOS'
date '+%Y-%m-%d %H:%M:%S %Z'
echo '--- init.d ---'; ls /etc/init.d
echo '--- rc.d enabled ---'; ls /etc/rc.d
echo '--- service list ---'; ubus call service list 2>/dev/null
EOS

sect 05 interfaces-routes <<'EOS'
date '+%Y-%m-%d %H:%M:%S %Z'
echo '--- ip -br addr ---'; ip -br addr
echo '--- ip route (main) ---'; ip route show table main
echo '--- ip route all ---'; ip route show table all
echo '--- ip rule ---'; ip rule show
echo '--- rt_tables ---'; cat /etc/iproute2/rt_tables 2>/dev/null
echo '--- ubus network.interface dump ---'; ubus call network.interface dump 2>/dev/null
EOS

sect 06 ipv6 <<'EOS'
date '+%Y-%m-%d %H:%M:%S %Z'
echo '--- ip -6 addr ---'; ip -6 addr
echo '--- ip -6 route all ---'; ip -6 route show table all
echo '--- ip -6 rule ---'; ip -6 rule show
echo '--- ip -6 neigh ---'; ip -6 neigh
echo '--- sysctl ipv6 all ---'
for k in forwarding disable_ipv6 accept_ra accept_ra_defrtr accept_ra_pinfo autoconf use_tempaddr; do
  printf 'all/%s=' "$k"; cat /proc/sys/net/ipv6/conf/all/$k 2>/dev/null || echo '?'
done
echo '--- per-interface sysctl ---'
for i in $(ls /proc/sys/net/ipv6/conf | grep -v '^all$' | grep -v '^default$'); do
  echo "[$i]"; for k in forwarding disable_ipv6 accept_ra accept_ra_defrtr accept_ra_pinfo accept_ra_fromlocal autoconf; do
    printf '  %s=' "$k"; cat /proc/sys/net/ipv6/conf/$i/$k 2>/dev/null || echo '?'
  done
done
echo '--- default sysctl ---'
for k in forwarding accept_ra disable_ipv6; do printf 'default/%s=' "$k"; cat /proc/sys/net/ipv6/conf/default/$k 2>/dev/null || echo '?'; done
echo '--- ip6tables/nft ip6 families ---'; nft list tables
EOS

sect 07 config-network <<'EOS'
date '+%Y-%m-%d %H:%M:%S %Z'
echo '--- /etc/config/network ---'; cat /etc/config/network
echo '--- /etc/config/dhcp ---'; cat /etc/config/dhcp
echo '--- /etc/config/odhcpd ---'; cat /etc/config/odhcpd 2>/dev/null || echo '(no /etc/config/odhcpd)'
EOS

sect 08 firewall <<'EOS'
date '+%Y-%m-%d %H:%M:%S %Z'
echo '--- /etc/config/firewall ---'; cat /etc/config/firewall
echo '--- nftables.d ---'; ls -la /etc/nftables.d 2>/dev/null; cat /etc/nftables.d/* 2>/dev/null
echo '--- fw4 includes in /etc/firewall.user ---'; cat /etc/firewall.user 2>/dev/null || echo '(none)'
EOS

sect 09 nft-ruleset <<'EOS'
date '+%Y-%m-%d %H:%M:%S %Z'
echo '--- nft list ruleset ---'; nft list ruleset
echo '--- nft tables ---'; nft list tables
echo '--- nft version ---'; nft --version
EOS

sect 10 dns <<'EOS'
date '+%Y-%m-%d %H:%M:%S %Z'
echo '--- resolv.conf ---'; cat /etc/resolv.conf
echo '--- resolv.conf.auto ---'; cat /tmp/resolv.conf.d/resolv.conf.auto 2>/dev/null; cat /tmp/resolv.conf.auto 2>/dev/null
echo '--- listeners ---'; (ss -lntup 2>/dev/null || netstat -lntup 2>/dev/null)
echo '--- dnsmasq procs ---'; ps w 2>/dev/null | grep -E 'dnsmasq|odhcpd|https-dns' | grep -v grep
echo '--- /etc/config/dhcp dnsmasq section ---'; uci show dhcp 2>/dev/null | grep -E 'dnsmasq|domain|server|noresolv|rebind|cache' 
EOS

sect 11 wan <<'EOS'
date '+%Y-%m-%d %H:%M:%S %Z'
echo '--- wan status ---'; ubus call network.interface.wan status 2>/dev/null
echo '--- wan6 status ---'; ubus call network.interface.wan6 status 2>/dev/null
echo '--- ifstatus wan ---'; ifstatus wan 2>/dev/null | head -60
echo '--- pppoe iface ---'; ip -4 addr show dev $(uci get network.wan.device 2>/dev/null || echo eth1) 2>/dev/null
echo '--- default route ---'; ip route show default
echo '--- wan dns ---'; uci show network 2>/dev/null | grep -iE 'dns|peerdns|proto|device|ipv6|reqaddress|reqprefix'
EOS

sect 12 old-project <<'EOS'
date '+%Y-%m-%d %H:%M:%S %Z'
echo '--- etc/xray ---'; ls -la /etc/xray 2>/dev/null
echo '--- usr/share/xray ---'; ls -la /usr/share/xray 2>/dev/null
echo '--- init.d xray* ---'; ls -la /etc/init.d/ 2>/dev/null | grep -iE 'xray|zapret|passwall|v2ray|sing'
echo '--- rc.d xray* ---'; ls -la /etc/rc.d/ | grep -iE 'xray|zapret|passwall|v2ray|sing'
echo '--- uci xray (masked) ---'; uci show xray 2>/dev/null | sed -E 's#(vless|vmess|trojan|ss|https?)://[^ ]+#<redacted-url>#g; s#(uuid|pbk|sid|password|token|url)[=:][^ ]+#\1=<redacted>#g'
echo '--- state ---'; cat /etc/xray/state 2>/dev/null; echo; cat /etc/xray/emergency 2>/dev/null
echo '--- tmp state ---'; cat /tmp/xray-state.vol /tmp/xray-wd.vol /tmp/xray-tproxy.sig /tmp/xray-tproxy.revert 2>/dev/null
echo '--- procs ---'; ps w 2>/dev/null | grep -iE 'xray|zapret|passwall|v2ray|sing' | grep -v grep
echo '--- cron ---'; crontab -l 2>/dev/null
echo '--- leftover dirs ---'; ls -la /usr/bin/ 2>/dev/null | grep -iE 'xray|zapret|passwall|v2ray|sing'
echo '--- xray version ---'; xray version 2>/dev/null
EOS

sect 13 logs-errors <<'EOS'
date '+%Y-%m-%d %H:%M:%S %Z'
echo '--- logread tail ---'; logread 2>/dev/null | tail -80
echo '--- dmesg errors ---'; dmesg 2>/dev/null | grep -iE 'error|fail|warn|throttl|thermal|oom' | tail -40
echo '--- thermal ---'; cat /sys/class/thermal/thermal_zone0/temp 2>/dev/null
echo '--- pstore ---'; ls -la /sys/fs/pstore/ 2>/dev/null
echo '--- conntrack ---'; cat /proc/sys/net/netfilter/nf_conntrack_count /proc/sys/net/netfilter/nf_conntrack_max 2>/dev/null
echo '--- state.log ---'; ls -la /root/state.log* 2>/dev/null
EOS

echo "=== done ==="
ls -la "$OUT"
