package model

import (
	"time"
)

type Device struct {
	ID              int64     `json:"id"`
	IPAddress       string    `json:"ip_address"`
	MACAddress      string    `json:"mac_address"`
	SNMPPort        uint16    `json:"snmp_port"`
	SNMPCommunity   string    `json:"snmp_community"`
	SNMPVersion     uint8     `json:"snmp_version"`
	Name            string    `json:"name"`
	Location        string    `json:"location"`
	Contact         string    `json:"contact"`
	LastPollTime    time.Time `json:"last_poll_time"`
	LastPollSuccess bool      `json:"last_poll_success"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	Metrics  *DeviceMetrics  `json:"metrics,omitempty"`
	Alarms   *DeviceAlarms   `json:"alarms,omitempty"`
	Channels []ChannelStatus `json:"channels,omitempty"`
}

type DeviceMetrics struct {
	RadioOutputEnabled int    `json:"radio_output_enabled"`
	Temperature        string `json:"temperature"`
	InternalPower1     string `json:"internal_power_1"`
	InternalPower2     string `json:"internal_power_2"`
}

type DeviceAlarms struct {
	EthLos               bool `json:"eth_los"`
	Shortcut             bool `json:"shortcut"`
	PowerUpLimit         bool `json:"power_up_limit"`
	PowerBottomLimit     bool `json:"power_bottom_limit"`
	TemperatureLimit     bool `json:"temperature_limit"`
	Channel1NotConnected bool `json:"channel_1_not_connected"`
	Channel2NotConnected bool `json:"channel_2_not_connected"`
	Channel3NotConnected bool `json:"channel_3_not_connected"`
}

type ChannelStatus struct {
	ChannelID int    `json:"channel_id"`
	StateLink int    `json:"state_link"`
	Stream    string `json:"stream"`
	Server    string `json:"server"`
}

// Константы состояний
const (
	RadioOutputOff = iota
	RadioOutputOn
	RadioOutputOffTemperatureLimit
	RadioOutputOffTemperatureHyst
	RadioOutputOffPowerUpLimit
	RadioOutputOffPowerBottomLimit
	RadioOutputOffPowerPause
	RadioOutputOffProtectionMinor
	RadioOutputOffProtectionMajor
	RadioOutputOffProtectionPause
)

const (
	ChannelStateOff = iota
	ChannelStateToBeStopped
	ChannelStateStarting
	ChannelStateStarted
	ChannelStateConnected
	ChannelStateReconnecting
)
