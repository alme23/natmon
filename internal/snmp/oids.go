package snmp

import "fmt"

// OID константы для Nateks NetRadio
const (
	// Системная информация (RFC1213)
	OIDSysName     = ".1.3.6.1.2.1.1.5.0"
	OIDSysContact  = ".1.3.6.1.2.1.1.4.0"
	OIDSysLocation = ".1.3.6.1.2.1.1.6.0"
	OIDSysUpTime   = ".1.3.6.1.2.1.1.3.0"

	// MAC-адрес (IF-MIB, интерфейс 2)
	OIDMACAddress = ".1.3.6.1.2.1.2.2.1.6.2"

	// Nateks NetRadio MIB
	// MaintenanceMgt = .1.3.6.1.4.1.4249.1.31.3

	// RadioStatusTable = MaintenanceMgt.1
	OIDRadioStatusTable = ".1.3.6.1.4.1.4249.1.31.3.1"

	// ChannelStatusEntry = RadioStatusTable.1
	OIDChannelStatusEntry = ".1.3.6.1.4.1.4249.1.31.3.1.1"

	// RadioOutput = MaintenanceMgt.1.2 (скаляр - с .0)
	OIDRadioOutput = ".1.3.6.1.4.1.4249.1.31.3.1.2.0"

	// RadioTemperature = MaintenanceMgt.1.3 (скаляр - с .0)
	OIDRadioTemperature = ".1.3.6.1.4.1.4249.1.31.3.1.3.0"

	// RadioInternalPower1 = MaintenanceMgt.1.4 (скаляр - с .0)
	OIDRadioInternalPower1 = ".1.3.6.1.4.1.4249.1.31.3.1.4.0"

	// RadioInternalPower2 = MaintenanceMgt.1.5 (скаляр - с .0)
	OIDRadioInternalPower2 = ".1.3.6.1.4.1.4249.1.31.3.1.5.0"

	// Alarms = MaintenanceMgt.3 (скаляры - с .0)
	OIDAlarmBase                 = ".1.3.6.1.4.1.4249.1.31.3.3"
	OIDAlarmEthLos               = ".1.3.6.1.4.1.4249.1.31.3.3.1.0"
	OIDAlarmShortcut             = ".1.3.6.1.4.1.4249.1.31.3.3.2.0"
	OIDAlarmPowerUpLimit         = ".1.3.6.1.4.1.4249.1.31.3.3.3.0"
	OIDAlarmPowerBottomLimit     = ".1.3.6.1.4.1.4249.1.31.3.3.4.0"
	OIDAlarmTemperatureLimit     = ".1.3.6.1.4.1.4249.1.31.3.3.5.0"
	OIDAlarmChannel1NotConnected = ".1.3.6.1.4.1.4249.1.31.3.3.6.0"
	OIDAlarmChannel2NotConnected = ".1.3.6.1.4.1.4249.1.31.3.3.7.0"
	OIDAlarmChannel3NotConnected = ".1.3.6.1.4.1.4249.1.31.3.3.8.0"
)

// Поля ChannelStatusEntry (табличные - БЕЗ .0)
const (
	ChannelStateField        = 2
	ChannelStreamField       = 3
	ChannelServerField       = 4
	ChannelMetaintervalField = 5
	ChannelBitrateField      = 6
	ChannelSamplerateField   = 7
)

// GetChannelOID возвращает OID для поля канала
// Формат: ChannelStatusEntry.field.channel (БЕЗ .0 в конце)
func GetChannelOID(channelNum int, field int) string {
	return fmt.Sprintf("%s.%d.%d", OIDChannelStatusEntry, field, channelNum)
}
