'use strict';
'require baseclass';
'require rpc';

// Граница LuCI → демон. Транспорт ubus подключается на следующем этапе;
// до его появления вызов возвращает ошибку доступности, а не вымышленное
// состояние.
var callStatus = rpc.declare({
	object: 'vpnmanager',
	method: 'status',
	expect: { }
});

var callVersion = rpc.declare({
	object: 'vpnmanager',
	method: 'version',
	expect: { }
});

return baseclass.extend({
	title: _('VPN Manager'),

	load: function() {
		return Promise.all([
			L.resolveDefault(callStatus(), null),
			L.resolveDefault(callVersion(), null)
		]);
	},

	render: function(data) {
		var status = data[0];
		var version = data[1];

		if (!status) {
			return E('p', { 'class': 'alert-message warning' },
				_('Демон VPN Manager недоступен: ubus-объект ещё не реализован (этап 1).'));
		}

		return E('div', { 'class': 'cbi-map' }, [
			E('h2', {}, _('Состояние')),
			E('table', { 'class': 'table' }, [
				E('tr', { 'class': 'tr' }, [
					E('td', { 'class': 'td left', 'width': '33%' }, _('Состояние')),
					E('td', { 'class': 'td left' }, status.state || _('неизвестно'))
				]),
				E('tr', { 'class': 'tr' }, [
					E('td', { 'class': 'td left' }, _('Причина')),
					E('td', { 'class': 'td left' }, status.reason || '—')
				]),
				E('tr', { 'class': 'tr' }, [
					E('td', { 'class': 'td left' }, _('Версия')),
					E('td', { 'class': 'td left' }, version ? version.version : '—')
				])
			])
		]);
	}
});
