function natmonApp() {
    return {
        // Состояние
        devices: [],
        selectedDevice: null,
        editingDevice: null,
        showAddDeviceModal: false,
        showDeviceDetailsModal: false,
        showEditDeviceModal: false,
        updatingDevices: new Set(),

        // Инициализация
        init() {
            console.log('Initializing NATMon app...');
            this.loadDevices();
            window.natmonInstance = this;

            setInterval(() => {
                if (this.devices.length > 0 && this.updatingDevices.size > 0) {
                    this.checkUpdateStatuses();
                }
            }, 3000);
        },

        // Функция для иконок
        icon(name, className = 'w-5 h-5') {
            return `<img src="/static/icons/${name}.svg" class="${className}" alt="${name}" />`;
        },

        // API запросы
        async apiRequest(endpoint, options = {}) {
            const headers = {
                'Content-Type': 'application/json',
                ...options.headers
            };

            const response = await fetch(endpoint, { ...options, headers });

            if (!response.ok) {
                const error = await response.text();
                throw new Error(error);
            }

            if (response.status === 204) {
                return null;
            }

            return response.json();
        },

        // Загрузка устройств
        async loadDevices() {
            const container = document.getElementById('devicesTable');

            try {
                const devices = await this.apiRequest('/api/devices');
                this.devices = devices;
                this.renderDevicesTable(devices);
            } catch (error) {
                console.error('Failed to load devices:', error);
                if (container) {
                    container.innerHTML = `
                        <div class="text-center py-8">
                            <p class="text-red-600 text-sm">Не удалось загрузить устройства: ${error.message}</p>
                            <button onclick="window.natmonInstance.loadDevices()"
                                    class="mt-2 text-sm text-blue-600 hover:text-blue-800">
                                Повторить
                            </button>
                        </div>
                    `;
                }
            }
        },

        // Отображение таблицы устройств
        renderDevicesTable(devices) {
            const container = document.getElementById('devicesTable');
            if (!container) return;

            if (!devices || devices.length === 0) {
                container.innerHTML = `
                    <div class="text-center py-8">
                        <p class="text-gray-500">Нет устройств</p>
                    </div>
                `;
                return;
            }

            let tableHtml = `
                <table class="min-w-full divide-y divide-gray-200">
                    <thead class="bg-gray-50">
                        <tr>
                            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">IP</th>
                            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Имя</th>
                            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Локация</th>
                            <th class="px-4 py-3 text-center text-xs font-medium text-gray-500 uppercase tracking-wider">Канал 1</th>
                            <th class="px-4 py-3 text-center text-xs font-medium text-gray-500 uppercase tracking-wider">Канал 2</th>
                            <th class="px-4 py-3 text-center text-xs font-medium text-gray-500 uppercase tracking-wider">Канал 3</th>
                            <th class="px-4 py-3 text-center text-xs font-medium text-gray-500 uppercase tracking-wider">Алармы</th>
                            <th class="px-4 py-3 text-center text-xs font-medium text-gray-500 uppercase tracking-wider">Действия</th>
                        </tr>
                    </thead>
                    <tbody class="bg-white divide-y divide-gray-200">
            `;

            devices.forEach(device => {
                const hasAlarms = device.alarms ? this.checkAlarms(device.alarms) : false;

                tableHtml += `
                    <tr class="hover:bg-gray-50 transition-colors duration-150 ${hasAlarms ? 'bg-red-50' : ''}">
                        <td class="px-4 py-3 text-sm text-gray-900 cursor-pointer" onclick="window.showDeviceDetails(${device.id})">
                            ${device.ip_address || '-'}
                        </td>
                        <td class="px-4 py-3 text-sm text-gray-900">${device.name || '-'}</td>
                        <td class="px-4 py-3 text-sm text-gray-900">${device.location || '-'}</td>
                        <td class="px-4 py-3 text-center">${this.renderChannelBadge(device, 1)}</td>
                        <td class="px-4 py-3 text-center">${this.renderChannelBadge(device, 2)}</td>
                        <td class="px-4 py-3 text-center">${this.renderChannelBadge(device, 3)}</td>
                        <td class="px-4 py-3 text-center">
                            ${hasAlarms ?
                                '<span class="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-red-100 text-red-800">Есть</span>' :
                                '<span class="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-green-100 text-green-800">Нет</span>'}
                        </td>
                        <td class="px-4 py-3 text-center">
                            <button onclick="window.pollDevice(${device.id})"
                                    class="text-blue-600 hover:text-blue-800 p-1 mr-2" title="Опросить">
                                ${this.icon('refresh', 'w-5 h-5 inline')}
                            </button>
                            <button onclick="window.editDevice(${device.id})"
                                    class="text-blue-600 hover:text-blue-800 p-1 mr-2" title="Редактировать">
                                ${this.icon('edit', 'w-4 h-4 inline')}
                            </button>
                            <button onclick="window.deleteDevice(${device.id})"
                                    class="text-red-600 hover:text-red-800 p-1" title="Удалить">
                                ${this.icon('delete', 'w-4 h-4 inline')}
                            </button>
                        </td>
                    </tr>
                `;
            });

            tableHtml += `
                    </tbody>
                </table>
            `;

            container.innerHTML = tableHtml;
        },

        // Отображение статуса канала
        renderChannelBadge(device, channelNum) {
            const channel = device.channels ? device.channels.find(c => c.channel_id === channelNum) : null;
            if (!channel) {
                return '<span class="inline-block w-3 h-3 rounded-full bg-gray-300" title="Нет данных"></span>';
            }

            let color, label;
            switch (channel.state_link) {
                case 0: color = 'bg-gray-500'; label = 'off'; break;
                case 1: color = 'bg-yellow-500'; label = 'to_be_stopped'; break;
                case 2: color = 'bg-orange-500'; label = 'starting'; break;
                case 3: color = 'bg-blue-500'; label = 'started'; break;
                case 4: color = 'bg-green-500'; label = 'connected'; break;
                case 5: color = 'bg-orange-500'; label = 'reconnecting'; break;
                default: color = 'bg-gray-300'; label = 'unknown'; break;
            }

            return `<span class="inline-block w-3 h-3 rounded-full ${color}" title="Канал ${channelNum}: ${label}"></span>`;
        },

        // Проверка алармов
        checkAlarms(alarms) {
            return alarms.eth_los ||
                alarms.shortcut ||
                alarms.power_up_limit ||
                alarms.power_bottom_limit ||
                alarms.temperature_limit ||
                alarms.channel_1_not_connected ||
                alarms.channel_2_not_connected ||
                alarms.channel_3_not_connected;
        },

        // Добавление устройства
        async addDevice(event) {
            event.preventDefault();

            const formData = new FormData(event.target);
            const deviceData = {
                ip_address: formData.get('ip_address'),
                snmp_port: parseInt(formData.get('snmp_port')) || 161,
                snmp_community: formData.get('snmp_community') || 'private'
            };

            try {
                const result = await this.apiRequest('/api/devices', {
                    method: 'POST',
                    body: JSON.stringify(deviceData)
                });

                this.showAddDeviceModal = false;
                event.target.reset();
                await this.loadDevices();
                this.showNotification(`Устройство "${result.ip_address}" добавлено`, 'success');
            } catch (error) {
                console.error('Failed to add device:', error);
                this.showNotification('Не удалось добавить устройство: ' + error.message, 'error');
            }
        },

        // Обновление устройства
        async updateDevice(event) {
            event.preventDefault();

            if (!this.editingDevice || !this.editingDevice.id) {
                return;
            }

            const formData = new FormData(event.target);
            const deviceData = {
                ip_address: formData.get('ip_address'),
                snmp_port: parseInt(formData.get('snmp_port')) || 161,
                snmp_community: formData.get('snmp_community') || 'private'
            };

            try {
                await this.apiRequest(`/api/devices/${this.editingDevice.id}`, {
                    method: 'PUT',
                    body: JSON.stringify(deviceData)
                });

                this.showEditDeviceModal = false;
                await this.loadDevices();
                this.showNotification('Устройство обновлено', 'success');
            } catch (error) {
                console.error('Failed to update device:', error);
                this.showNotification('Не удалось обновить устройство: ' + error.message, 'error');
            }
        },

        // Опросить устройство
        async pollDevice(deviceId) {
            try {
                this.showNotification('Опрос устройства...', 'info');

                await this.apiRequest(`/api/devices/${deviceId}/poll`, {
                    method: 'POST'
                });

                this.showNotification('Опрос завершен', 'success');
                await this.loadDevices();
            } catch (error) {
                console.error('Failed to poll device:', error);
                this.showNotification('Не удалось опросить устройство: ' + error.message, 'error');
            }
        },

        // Показать детали устройства
        async showDeviceDetails(deviceId) {
            const device = this.devices.find(d => d.id === deviceId);
            if (!device) return;

            this.selectedDevice = device;
            this.showDeviceDetailsModal = true;

            const modalContent = document.getElementById('deviceDetailsModalContent');
            if (!modalContent) return;

            modalContent.innerHTML = this.renderDeviceDetailsModal(device);
        },

        renderDeviceDetailsModal(device) {
            return `
                <div class="flex justify-between items-start mb-4">
                    <h3 class="text-lg font-medium text-gray-900">Детали устройства</h3>
                    <button onclick="window.closeDeviceDetails()" class="text-gray-400 hover:text-gray-600">
                        ${this.icon('close', 'w-6 h-6')}
                    </button>
                </div>

                <div class="space-y-4">
                    <div class="grid grid-cols-2 gap-4">
                        <div>
                            <label class="text-sm text-gray-500">IP адрес</label>
                            <p class="font-medium">${device.ip_address || '-'}</p>
                        </div>
                        <div>
                            <label class="text-sm text-gray-500">Имя</label>
                            <p class="font-medium">${device.name || '-'}</p>
                        </div>
                        <div>
                            <label class="text-sm text-gray-500">Локация</label>
                            <p class="font-medium">${device.location || '-'}</p>
                        </div>
                        <div>
                            <label class="text-sm text-gray-500">Контакт</label>
                            <p class="font-medium">${device.contact || '-'}</p>
                        </div>
                        <div>
                            <label class="text-sm text-gray-500">SNMP Port</label>
                            <p class="font-medium">${device.snmp_port || 161}</p>
                        </div>
                        <div>
                            <label class="text-sm text-gray-500">Community</label>
                            <p class="font-medium">${device.snmp_community || 'private'}</p>
                        </div>
                    </div>

                    ${device.metrics ? `
                        <div class="border-t pt-4">
                            <h4 class="font-medium text-gray-900 mb-3">Метрики</h4>
                            <div class="grid grid-cols-2 gap-4">
                                <div>
                                    <label class="text-sm text-gray-500">Температура</label>
                                    <p class="font-medium">${device.metrics.temperature || '-'}</p>
                                </div>
                                <div>
                                    <label class="text-sm text-gray-500">Питание 1</label>
                                    <p class="font-medium">${device.metrics.internal_power_1 || '-'}</p>
                                </div>
                                <div>
                                    <label class="text-sm text-gray-500">Питание 2</label>
                                    <p class="font-medium">${device.metrics.internal_power_2 || '-'}</p>
                                </div>
                            </div>
                        </div>
                    ` : ''}

                    <div class="border-t pt-4">
                        <h4 class="font-medium text-gray-900 mb-3">Каналы</h4>
                        <div class="space-y-2">
                            ${[1, 2, 3].map(ch => this.renderChannelDetails(device, ch)).join('')}
                        </div>
                    </div>
                </div>
            `;
        },

        renderChannelDetails(device, channelNum) {
            const channel = device.channels ? device.channels.find(c => c.channel_id === channelNum) : null;
            if (!channel) {
                return `
                    <div class="flex items-center justify-between p-2 bg-gray-50 rounded">
                        <span class="text-sm font-medium">Канал ${channelNum}</span>
                        <span class="text-sm text-gray-400">Нет данных</span>
                    </div>
                `;
            }

            let stateColor, stateText;
            switch (channel.state_link) {
                case 0: stateColor = 'text-gray-600'; stateText = 'off'; break;
                case 1: stateColor = 'text-yellow-600'; stateText = 'to_be_stopped'; break;
                case 2: stateColor = 'text-orange-600'; stateText = 'starting'; break;
                case 3: stateColor = 'text-blue-600'; stateText = 'started'; break;
                case 4: stateColor = 'text-green-600'; stateText = 'connected'; break;
                case 5: stateColor = 'text-orange-600'; stateText = 'reconnecting'; break;
                default: stateColor = 'text-gray-600'; stateText = 'unknown'; break;
            }

            return `
                <div class="p-3 bg-gray-50 rounded">
                    <div class="flex items-center justify-between mb-2">
                        <span class="text-sm font-medium">Канал ${channelNum}</span>
                        <span class="text-sm ${stateColor}">${stateText}</span>
                    </div>
                    ${channel.stream ? `<div class="text-xs text-gray-600">Stream: ${channel.stream}</div>` : ''}
                    ${channel.server ? `<div class="text-xs text-gray-600">Server: ${channel.server}</div>` : ''}
                </div>
            `;
        },

        // Уведомления
        showNotification(message, type = 'info') {
            const container = document.getElementById('notificationContainer');
            if (!container) return;

            const notification = document.createElement('div');
            const bgColor = type === 'success' ? 'bg-green-500' : type === 'error' ? 'bg-red-500' : 'bg-blue-500';
            const iconName = type === 'success' ? 'success' : type === 'error' ? 'error' : 'info';

            notification.className = `${bgColor} text-white px-6 py-3 rounded-md shadow-lg flex items-center transition-all duration-300`;
            notification.innerHTML = `
                <span class="mr-2">${this.icon(iconName, 'w-5 h-5')}</span>
                ${message}
            `;

            container.appendChild(notification);

            setTimeout(() => {
                notification.style.opacity = '0';
                notification.style.transform = 'translateX(100%)';
                setTimeout(() => notification.remove(), 300);
            }, type === 'error' ? 5000 : 3000);
        }
    };
}

// Глобальные функции
window.showDeviceDetails = function(deviceId) {
    const app = window.natmonInstance;
    if (app) app.showDeviceDetails(deviceId);
};

window.closeDeviceDetails = function() {
    const app = window.natmonInstance;
    if (app) app.showDeviceDetailsModal = false;
};

window.pollDevice = function(deviceId) {
    const app = window.natmonInstance;
    if (app) app.pollDevice(deviceId);
};

window.editDevice = function(deviceId) {
    const app = window.natmonInstance;
    if (!app) return;
    const device = app.devices.find(d => d.id === deviceId);
    if (device) {
        app.editingDevice = { ...device };
        app.showEditDeviceModal = true;
    }
};

window.deleteDevice = async function(deviceId) {
    const app = window.natmonInstance;
    if (!app) return;

    if (confirm('Вы уверены, что хотите удалить это устройство?')) {
        try {
            await app.apiRequest(`/api/devices/${deviceId}`, { method: 'DELETE' });
            await app.loadDevices();
            app.showNotification('Устройство удалено', 'success');
        } catch (error) {
            app.showNotification('Не удалось удалить устройство', 'error');
        }
    }
};
