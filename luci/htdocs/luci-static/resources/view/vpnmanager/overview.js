'use strict';
'require view';
'require rpc';

/*
 * Overview — единственная вью этапа 1. Обращается к демону только через
 * ubus-метод vpnmanager.status из реестра №12 §6; при отсутствии объекта
 * показывает сообщение о недоступности, а не вымышленный успешный статус
 * (№44 §3.8, №11 §13). Отдельного метода версии в реестре нет, поэтому
 * версия в интерфейсе не запрашивается и не подставляется.
 * Транспорт ubus в этапе 1 не реализован.
 */

var callStatus = rpc.declare({
	object: 'vpnmanager',
	method: 'status',
	expect: { }
});

function row(label, value) {
	return E('tr', { 'class': 'tr' }, [
		E('td', { 'class': 'td left', 'width': '33%' }, label),
		E('td', { 'class': 'td left' }, value)
	]);
}

return view.extend({
	load: function() {
		return L.resolveDefault(callStatus(), null);
	},

	render: function(status) {
		if (!status) {
			return E('p', { 'class': 'alert-message warning' },
				_('Демон VPN Manager недоступен: ubus-объект vpnmanager не зарегистрирован.'));
		}

		var rows = [
			row(_('Состояние'), status.state || _('нет данных')),
			row(_('Причина'), status.reason || '—'),
			row(_('Конфигурация'), status.config ? (status.config + (status.config_error ? ': ' + status.config_error : '')) : '—'),
			row(_('Готовность'), status.ready ? _('ядро запущено') : _('не готово')),
			row(_('Наблюдение'), status.observed_at || _('нет данных'))
		];

		return E('div', { 'class': 'cbi-map' }, [
			E('h2', {}, _('VPN Manager: состояние')),
			E('table', { 'class': 'table' }, rows)
		]);
	},

	handleSaveApply: null,
	handleSave: null,
	handleReset: null
});
