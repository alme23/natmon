package presenter

import (
	"github.com/alme23/natmon/internal/model"
)

type ChannelPresenter struct {
	channel *model.ChannelStatus
}

func NewChannelPresenter(channel *model.ChannelStatus) *ChannelPresenter {
	return &ChannelPresenter{channel: channel}
}

// StateString возвращает текстовое описание состояния
func (p *ChannelPresenter) StateString() string {
	switch p.channel.StateLink {
	case model.ChannelStateOff:
		return "off"
	case model.ChannelStateToBeStopped:
		return "to_be_stopped"
	case model.ChannelStateStarting:
		return "starting"
	case model.ChannelStateStarted:
		return "started"
	case model.ChannelStateConnected:
		return "connected"
	case model.ChannelStateReconnecting:
		return "reconnecting"
	default:
		return "unknown"
	}
}

// BadgeColor возвращает CSS класс для бейджа
func (p *ChannelPresenter) BadgeColor() string {
	switch p.channel.StateLink {
	case model.ChannelStateOff:
		return "bg-gray-500"
	case model.ChannelStateToBeStopped:
		return "bg-yellow-500"
	case model.ChannelStateStarting:
		return "bg-orange-500"
	case model.ChannelStateStarted:
		return "bg-blue-500"
	case model.ChannelStateConnected:
		return "bg-green-500"
	case model.ChannelStateReconnecting:
		return "bg-orange-500"
	default:
		return "bg-gray-300"
	}
}

// TextColor возвращает CSS класс для текста
func (p *ChannelPresenter) TextColor() string {
	switch p.channel.StateLink {
	case model.ChannelStateOff:
		return "text-gray-600"
	case model.ChannelStateToBeStopped:
		return "text-yellow-600"
	case model.ChannelStateStarting:
		return "text-orange-600"
	case model.ChannelStateStarted:
		return "text-blue-600"
	case model.ChannelStateConnected:
		return "text-green-600"
	case model.ChannelStateReconnecting:
		return "text-orange-600"
	default:
		return "text-gray-600"
	}
}
