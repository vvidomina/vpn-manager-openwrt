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

return baseclass.extend({
	title: _('VPN Manager'),

	load: function() {
		return L.resolveDefault(callStatus(), null);
	},

	render: function(status) {
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
				])
			])
		]);
	}
});
