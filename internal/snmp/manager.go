package snmp

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/alme23/natmon/internal/model"
	"github.com/gosnmp/gosnmp"
)

type Manager struct {
	mu      sync.RWMutex
	workers map[int64]*DeviceWorker
	timeout time.Duration
	retries int
	closed  bool
}

func NewManager(timeout int, retries int) *Manager {
	return &Manager{
		workers: make(map[int64]*DeviceWorker),
		timeout: time.Duration(timeout) * time.Second,
		retries: retries,
	}
}

// getWorker возвращает или создает воркера для устройства
func (m *Manager) getWorker(device *model.Device) (*DeviceWorker, error) {
	m.mu.RLock()
	if m.closed {
		m.mu.RUnlock()
		return nil, fmt.Errorf("manager is closed")
	}
	worker, exists := m.workers[device.ID]
	m.mu.RUnlock()

	if exists {
		return worker, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil, fmt.Errorf("manager is closed")
	}

	// Double-checked locking
	if worker, exists := m.workers[device.ID]; exists {
		return worker, nil
	}

	worker = NewDeviceWorker(device, m.timeout, m.retries)
	m.workers[device.ID] = worker

	log.Printf("Worker created for device %d (%s)", device.ID, device.IPAddress)

	return worker, nil
}

// GetDeviceData получает все данные устройства (полный опрос)
func (m *Manager) GetDeviceData(ctx context.Context, device *model.Device) error {
	log.Printf("=== Full poll for device %s ===", device.IPAddress)

	// Получаем системную информацию
	if err := m.GetSystemInfo(ctx, device); err != nil {
		log.Printf("Failed to get system info: %v", err)
	}

	// Получаем метрики
	if err := m.GetMetrics(ctx, device); err != nil {
		log.Printf("Failed to get metrics: %v", err)
	}

	// Получаем алармы
	if err := m.GetAlarms(ctx, device); err != nil {
		log.Printf("Failed to get alarms: %v", err)
	}

	// Получаем каналы
	if err := m.GetChannels(ctx, device); err != nil {
		log.Printf("Failed to get channels: %v", err)
	}

	return nil
}

// GetSystemInfo получает только системную информацию (редко меняется)
// GetSystemInfo получает только системную информацию (редко меняется)
func (m *Manager) GetSystemInfo(ctx context.Context, device *model.Device) error {
	log.Printf("Getting system info for %s...", device.IPAddress)

	oids := []string{
		OIDSysName,
		OIDSysContact,
		OIDSysLocation,
		OIDMACAddress,
	}

	values, err := m.GetBulk(ctx, device, oids)
	if err != nil {
		return fmt.Errorf("failed to get system info: %w", err)
	}

	if name, ok := values[OIDSysName].(string); ok {
		device.Name = name
		log.Printf("  Name: %s", name)
	}

	if contact, ok := values[OIDSysContact].(string); ok {
		device.Contact = contact
		log.Printf("  Contact: %s", contact)
	}

	if location, ok := values[OIDSysLocation].(string); ok {
		device.Location = location
		log.Printf("  Location: %s", location)
	}

	// Обработка MAC-адреса
	if macValue, exists := values[OIDMACAddress]; exists {
		device.MACAddress = formatMACAddress(macValue)
		log.Printf("  MAC: %s", device.MACAddress)
	}

	return nil
}

// formatMACAddress форматирует MAC-адрес из различных типов данных
func formatMACAddress(value interface{}) string {
	switch v := value.(type) {
	case string:
		// Если это уже строка, проверяем, не бинарные ли данные
		if len(v) == 6 {
			// Похоже на бинарные данные в строке
			return formatMACBytes([]byte(v))
		}
		return v
	case []byte:
		return formatMACBytes(v)
	default:
		return fmt.Sprintf("%v", value)
	}
}

// formatMACBytes форматирует байты MAC-адреса в строку
func formatMACBytes(mac []byte) string {
	if len(mac) == 0 {
		return ""
	}

	// MAC-адрес обычно 6 байт
	parts := make([]string, len(mac))
	for i, b := range mac {
		parts[i] = fmt.Sprintf("%02x", b)
	}

	return strings.Join(parts, ":")
}

// GetMetrics получает только метрики (температура, питание)
func (m *Manager) GetMetrics(ctx context.Context, device *model.Device) error {
	log.Printf("Getting metrics for %s...", device.IPAddress)

	oids := []string{
		OIDRadioOutput,
		OIDRadioTemperature,
		OIDRadioInternalPower1,
		OIDRadioInternalPower2,
	}

	values, err := m.GetBulk(ctx, device, oids)
	if err != nil {
		return fmt.Errorf("failed to get metrics: %w", err)
	}

	metrics := &model.DeviceMetrics{}

	if v, ok := values[OIDRadioOutput].(int); ok {
		metrics.RadioOutputEnabled = v
		log.Printf("  RadioOutput: %d", v)
	}

	if v, ok := values[OIDRadioTemperature].(string); ok {
		metrics.Temperature = v
		log.Printf("  Temperature: %s", v)
	}

	if v, ok := values[OIDRadioInternalPower1].(string); ok {
		metrics.InternalPower1 = v
		log.Printf("  Power1: %s", v)
	}

	if v, ok := values[OIDRadioInternalPower2].(string); ok {
		metrics.InternalPower2 = v
		log.Printf("  Power2: %s", v)
	}

	device.Metrics = metrics
	return nil
}

// GetAlarms получает только алармы
// GetAlarms получает только алармы
func (m *Manager) GetAlarms(ctx context.Context, device *model.Device) error {
	log.Printf("Getting alarms for %s...", device.IPAddress)

	oids := []string{
		OIDAlarmEthLos,
		OIDAlarmShortcut,
		OIDAlarmPowerUpLimit,
		OIDAlarmPowerBottomLimit,
		OIDAlarmTemperatureLimit,
		OIDAlarmChannel1NotConnected,
		OIDAlarmChannel2NotConnected,
		OIDAlarmChannel3NotConnected,
	}

	values, err := m.GetBulk(ctx, device, oids)
	if err != nil {
		return fmt.Errorf("failed to get alarms: %w", err)
	}

	alarms := &model.DeviceAlarms{}

	if v, ok := values[OIDAlarmEthLos].(int); ok {
		alarms.EthLos = v == 1
	}
	if v, ok := values[OIDAlarmShortcut].(int); ok {
		alarms.Shortcut = v == 1
	}
	if v, ok := values[OIDAlarmPowerUpLimit].(int); ok {
		alarms.PowerUpLimit = v == 1
	}
	if v, ok := values[OIDAlarmPowerBottomLimit].(int); ok {
		alarms.PowerBottomLimit = v == 1
	}
	if v, ok := values[OIDAlarmTemperatureLimit].(int); ok {
		alarms.TemperatureLimit = v == 1
	}
	if v, ok := values[OIDAlarmChannel1NotConnected].(int); ok {
		alarms.Channel1NotConnected = v == 1
	}
	if v, ok := values[OIDAlarmChannel2NotConnected].(int); ok {
		alarms.Channel2NotConnected = v == 1
	}
	if v, ok := values[OIDAlarmChannel3NotConnected].(int); ok {
		alarms.Channel3NotConnected = v == 1
	}

	device.Alarms = alarms

	// Логируем активные алармы (проверяем напрямую)
	hasAlarms := alarms.EthLos ||
		alarms.Shortcut ||
		alarms.PowerUpLimit ||
		alarms.PowerBottomLimit ||
		alarms.TemperatureLimit ||
		alarms.Channel1NotConnected ||
		alarms.Channel2NotConnected ||
		alarms.Channel3NotConnected

	if hasAlarms {
		log.Printf("  Active alarms detected for %s", device.IPAddress)
	}

	return nil
}

// GetChannels получает только статусы каналов (легкий запрос)
func (m *Manager) GetChannels(ctx context.Context, device *model.Device) error {
	var channels []model.ChannelStatus

	for i := 1; i <= 3; i++ {
		stateOID := GetChannelOID(i, ChannelStateField)
		streamOID := GetChannelOID(i, ChannelStreamField)
		serverOID := GetChannelOID(i, ChannelServerField)

		values, err := m.GetBulk(ctx, device, []string{stateOID, streamOID, serverOID})
		if err != nil {
			log.Printf("Failed to get channel %d: %v", i, err)
			continue
		}

		ch := model.ChannelStatus{ChannelID: i}

		if v, ok := values[stateOID].(int); ok {
			ch.StateLink = v
		}

		if v, ok := values[streamOID].(string); ok {
			ch.Stream = v
		}

		if v, ok := values[serverOID].(string); ok {
			ch.Server = v
		}

		channels = append(channels, ch)
	}

	device.Channels = channels
	return nil
}

// Get выполняет SNMP GET запрос
func (m *Manager) Get(ctx context.Context, device *model.Device, oid string) (interface{}, error) {
	values, err := m.GetBulk(ctx, device, []string{oid})
	if err != nil {
		return nil, err
	}
	return values[oid], nil
}

// GetBulk выполняет получение нескольких OID
func (m *Manager) GetBulk(ctx context.Context, device *model.Device, oids []string) (map[string]interface{}, error) {
	worker, err := m.getWorker(device)
	if err != nil {
		return nil, err
	}

	return worker.Submit(ctx, oids)
}

// Set выполняет SNMP SET запрос
func (m *Manager) Set(ctx context.Context, device *model.Device, oid string, value interface{}) error {
	m.mu.RLock()
	if m.closed {
		m.mu.RUnlock()
		return fmt.Errorf("manager is closed")
	}
	m.mu.RUnlock()

	client := &gosnmp.GoSNMP{
		Target:    device.IPAddress,
		Port:      uint16(device.SNMPPort),
		Community: device.SNMPCommunity,
		Version:   gosnmp.Version1,
		Timeout:   m.timeout,
		Retries:   m.retries,
	}

	err := client.Connect()
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer client.Conn.Close()

	pdu := gosnmp.SnmpPDU{
		Name:  oid,
		Type:  detectSNMPType(value),
		Value: value,
	}

	_, err = client.Set([]gosnmp.SnmpPDU{pdu})
	if err != nil {
		return fmt.Errorf("SNMP SET failed: %w", err)
	}

	return nil
}

// Walk выполняет SNMP WALK запрос
func (m *Manager) Walk(ctx context.Context, device *model.Device, oid string) (interface{}, error) {
	m.mu.RLock()
	if m.closed {
		m.mu.RUnlock()
		return nil, fmt.Errorf("manager is closed")
	}
	m.mu.RUnlock()

	client := &gosnmp.GoSNMP{
		Target:    device.IPAddress,
		Port:      uint16(device.SNMPPort),
		Community: device.SNMPCommunity,
		Version:   gosnmp.Version1,
		Timeout:   m.timeout,
		Retries:   m.retries,
	}

	err := client.Connect()
	if err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}
	defer client.Conn.Close()

	results, err := client.WalkAll(oid)
	if err != nil {
		return nil, fmt.Errorf("SNMP WALK failed: %w", err)
	}

	return results, nil
}

// RemoveWorker удаляет воркера для устройства
func (m *Manager) RemoveWorker(deviceID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if worker, exists := m.workers[deviceID]; exists {
		worker.Stop()
		delete(m.workers, deviceID)
		log.Printf("Worker removed for device %d", deviceID)
	}
}

// Close закрывает менеджер и останавливает всех воркеров
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil
	}

	log.Println("Closing SNMP manager...")
	m.closed = true

	for deviceID, worker := range m.workers {
		worker.Stop()
		delete(m.workers, deviceID)
		log.Printf("Worker stopped for device %d", deviceID)
	}

	m.workers = make(map[int64]*DeviceWorker)
	log.Println("SNMP manager closed")

	return nil
}

// detectSNMPType определяет тип SNMP для значения
func detectSNMPType(value interface{}) gosnmp.Asn1BER {
	switch value.(type) {
	case int, int32, int64:
		return gosnmp.Integer
	case uint, uint32, uint64:
		return gosnmp.Uinteger32
	case string:
		return gosnmp.OctetString
	case bool:
		return gosnmp.Boolean
	case net.IP:
		return gosnmp.IPAddress
	default:
		return gosnmp.OctetString
	}
}

// formatSNMPValue преобразует значение SNMP в Go тип
func formatSNMPValue(pdu gosnmp.SnmpPDU) interface{} {
	switch pdu.Type {
	case gosnmp.OctetString:
		if bytes, ok := pdu.Value.([]byte); ok {
			// Проверяем, является ли это MAC-адресом (6 байт)
			if len(bytes) == 6 {
				// Проверяем, все ли байты печатные
				isPrintable := true
				for _, b := range bytes {
					if b < 32 || b > 126 {
						isPrintable = false
						break
					}
				}
				if !isPrintable {
					// Это бинарные данные (MAC-адрес)
					return formatMACBytes(bytes)
				}
			}
			return string(bytes)
		}
		return fmt.Sprintf("%v", pdu.Value)
	case gosnmp.ObjectIdentifier:
		return pdu.Value.(string)
	case gosnmp.TimeTicks:
		return pdu.Value.(uint32)
	case gosnmp.Counter32, gosnmp.Counter64:
		return pdu.Value.(uint64)
	case gosnmp.Gauge32:
		return pdu.Value.(uint32)
	case gosnmp.Integer:
		if v, ok := pdu.Value.(int); ok {
			return v
		}
		if v, ok := pdu.Value.(int32); ok {
			return int(v)
		}
		if v, ok := pdu.Value.(int64); ok {
			return int(v)
		}
		return 0
	case gosnmp.IPAddress:
		return pdu.Value.(string)
	case gosnmp.Null:
		return nil
	default:
		return pdu.Value
	}
}
