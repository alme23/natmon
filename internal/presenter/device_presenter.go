package presenter

import (
	"fmt"
	"time"

	"github.com/alme23/natmon/internal/model"
)

type DevicePresenter struct {
	device *model.Device
}

func NewDevicePresenter(device *model.Device) *DevicePresenter {
	return &DevicePresenter{device: device}
}

// HasAlarms возвращает true, если есть активные алармы
func (p *DevicePresenter) HasAlarms() bool {
	if p.device.Alarms == nil {
		return false
	}
	alarms := p.device.Alarms
	return alarms.EthLos ||
		alarms.Shortcut ||
		alarms.PowerUpLimit ||
		alarms.PowerBottomLimit ||
		alarms.TemperatureLimit ||
		alarms.Channel1NotConnected ||
		alarms.Channel2NotConnected ||
		alarms.Channel3NotConnected
}

// CanPoll возвращает true, если устройство можно опросить
func (p *DevicePresenter) CanPoll() bool {
	if p.device.LastPollTime.IsZero() {
		return true
	}
	return time.Since(p.device.LastPollTime) >= 3*time.Minute
}

// TimeUntilNextPoll возвращает время до следующего опроса
func (p *DevicePresenter) TimeUntilNextPoll() time.Duration {
	if p.device.LastPollTime.IsZero() {
		return 0
	}
	elapsed := time.Since(p.device.LastPollTime)
	if elapsed >= 3*time.Minute {
		return 0
	}
	return 3*time.Minute - elapsed
}

// FormatTimeUntilNextPoll форматирует время до следующего опроса
func (p *DevicePresenter) FormatTimeUntilNextPoll() string {
	d := p.TimeUntilNextPoll()
	if d <= 0 {
		return "0 сек"
	}
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	if minutes > 0 {
		return fmt.Sprintf("%d мин %d сек", minutes, seconds)
	}
	return fmt.Sprintf("%d сек", seconds)
}

// FormatPollTime форматирует время последнего опроса
func (p *DevicePresenter) FormatPollTime() string {
	if p.device.LastPollTime.IsZero() {
		return "Никогда"
	}
	return p.device.LastPollTime.Format("02.01.2006 15:04:05")
}

// GetChannel возвращает презентер канала
func (p *DevicePresenter) GetChannel(channelID int) *ChannelPresenter {
	for i := range p.device.Channels {
		if p.device.Channels[i].ChannelID == channelID {
			return &ChannelPresenter{channel: &p.device.Channels[i]}
		}
	}
	// Возвращаем пустой канал, если не найден
	return &ChannelPresenter{
		channel: &model.ChannelStatus{
			ChannelID: channelID,
			StateLink: -1,
		},
	}
}

// GetChannels возвращает презентеры всех каналов
func (p *DevicePresenter) GetChannels() []*ChannelPresenter {
	var presenters []*ChannelPresenter
	for i := range p.device.Channels {
		presenters = append(presenters, &ChannelPresenter{channel: &p.device.Channels[i]})
	}
	return presenters
}

// GetMetrics возвращает презентер метрик
func (p *DevicePresenter) GetMetrics() *MetricsPresenter {
	return NewMetricsPresenter(p.device.Metrics)
}

// GetAlarms возвращает презентер алармов
func (p *DevicePresenter) GetAlarms() *AlarmPresenter {
	return NewAlarmPresenter(p.device.Alarms)
}
