package presenter

import (
	"github.com/alme23/natmon/internal/model"
)

type AlarmPresenter struct {
	alarms *model.DeviceAlarms
}

func NewAlarmPresenter(alarms *model.DeviceAlarms) *AlarmPresenter {
	return &AlarmPresenter{alarms: alarms}
}

type AlarmItem struct {
	Label  string
	Active bool
}

// GetAlarms возвращает список алармов с их состоянием
func (p *AlarmPresenter) GetAlarms() []AlarmItem {
	if p.alarms == nil {
		return nil
	}

	return []AlarmItem{
		{Label: "Ethernet LOS", Active: p.alarms.EthLos},
		{Label: "Короткое замыкание", Active: p.alarms.Shortcut},
		{Label: "Верхний предел питания", Active: p.alarms.PowerUpLimit},
		{Label: "Нижний предел питания", Active: p.alarms.PowerBottomLimit},
		{Label: "Предел температуры", Active: p.alarms.TemperatureLimit},
		{Label: "Канал 1 не подключен", Active: p.alarms.Channel1NotConnected},
		{Label: "Канал 2 не подключен", Active: p.alarms.Channel2NotConnected},
		{Label: "Канал 3 не подключен", Active: p.alarms.Channel3NotConnected},
	}
}
