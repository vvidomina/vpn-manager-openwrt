'use strict';
'require view';
'require vpnmanager.controller as controller';

return view.extend({
	load: function() {
		return this.controller.load();
	},

	render: function(data) {
		return this.controller.render(data);
	},

	handleSaveApply: null,
	handleSave: null,
	handleReset: null
});
