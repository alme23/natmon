package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/alme23/natmon/internal/model"
	_ "modernc.org/sqlite"
)

type DeviceSQLite struct {
	db *sql.DB
}

func NewDeviceSQLite(path string) (*DeviceSQLite, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA foreign_keys=ON",
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			log.Printf("Warning: failed to set pragma %s: %v", pragma, err)
		}
	}

	repo := &DeviceSQLite{db: db}
	if err := repo.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to migrate: %w", err)
	}

	return repo, nil
}

func (r *DeviceSQLite) migrate() error {
	// Создаем таблицы, если их нет
	queries := []string{
		`CREATE TABLE IF NOT EXISTS devices (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ip_address TEXT NOT NULL UNIQUE,
			mac_address TEXT,
			snmp_port INTEGER DEFAULT 161,
			snmp_community TEXT DEFAULT 'private',
			snmp_version INTEGER DEFAULT 1,
			name TEXT,
			location TEXT,
			contact TEXT,
			last_poll_time TIMESTAMP,
			last_poll_success BOOLEAN DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS device_metrics (
			device_id INTEGER PRIMARY KEY,
			radio_output_enabled INTEGER DEFAULT 0,
			temperature TEXT,
			internal_power_1 TEXT,
			internal_power_2 TEXT,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS device_alarms (
			device_id INTEGER PRIMARY KEY,
			eth_los BOOLEAN DEFAULT 0,
			shortcut BOOLEAN DEFAULT 0,
			power_up_limit BOOLEAN DEFAULT 0,
			power_bottom_limit BOOLEAN DEFAULT 0,
			temperature_limit BOOLEAN DEFAULT 0,
			channel_1_not_connected BOOLEAN DEFAULT 0,
			channel_2_not_connected BOOLEAN DEFAULT 0,
			channel_3_not_connected BOOLEAN DEFAULT 0,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS device_channels (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			device_id INTEGER NOT NULL,
			channel_id INTEGER NOT NULL,
			state_link INTEGER DEFAULT 0,
			stream TEXT,
			server TEXT,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE,
			UNIQUE(device_id, channel_id)
		)`,
	}

	for _, query := range queries {
		if _, err := r.db.Exec(query); err != nil {
			return err
		}
	}

	// Проверяем, есть ли колонка mac_address
	if !r.columnExists("devices", "mac_address") {
		log.Println("Adding mac_address column to devices table...")
		_, err := r.db.Exec(`ALTER TABLE devices ADD COLUMN mac_address TEXT`)
		if err != nil {
			log.Printf("Failed to add mac_address column: %v", err)
		}
	}

	return nil
}

// columnExists проверяет существование колонки в таблице
func (r *DeviceSQLite) columnExists(tableName, columnName string) bool {
	query := fmt.Sprintf("SELECT COUNT(*) FROM pragma_table_info('%s') WHERE name = '%s'", tableName, columnName)
	var count int
	err := r.db.QueryRow(query).Scan(&count)
	if err != nil {
		log.Printf("Error checking column %s in table %s: %v", columnName, tableName, err)
		return false
	}
	return count > 0
}

func (r *DeviceSQLite) Close() error {
	return r.db.Close()
}

// GetAll возвращает все устройства
func (r *DeviceSQLite) GetAll(ctx context.Context) ([]model.Device, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, ip_address, mac_address, snmp_port, snmp_community, snmp_version,
			   name, location, contact, last_poll_time, last_poll_success,
			   created_at, updated_at
		FROM devices ORDER BY ip_address`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []model.Device
	for rows.Next() {
		var d model.Device
		var macAddress sql.NullString
		var lastPoll sql.NullTime

		err := rows.Scan(&d.ID, &d.IPAddress, &macAddress, &d.SNMPPort, &d.SNMPCommunity, &d.SNMPVersion,
			&d.Name, &d.Location, &d.Contact, &lastPoll, &d.LastPollSuccess,
			&d.CreatedAt, &d.UpdatedAt)
		if err != nil {
			return nil, err
		}

		d.MACAddress = macAddress.String
		if lastPoll.Valid {
			d.LastPollTime = lastPoll.Time
		}

		devices = append(devices, d)
	}

	return devices, nil
}

// GetByID возвращает устройство по ID
func (r *DeviceSQLite) GetByID(ctx context.Context, id int64) (*model.Device, error) {
	var d model.Device
	var macAddress sql.NullString
	var lastPoll sql.NullTime

	err := r.db.QueryRowContext(ctx, `
		SELECT id, ip_address, mac_address, snmp_port, snmp_community, snmp_version,
			   name, location, contact, last_poll_time, last_poll_success,
			   created_at, updated_at
		FROM devices WHERE id = ?`, id).Scan(
		&d.ID, &d.IPAddress, &macAddress, &d.SNMPPort, &d.SNMPCommunity, &d.SNMPVersion,
		&d.Name, &d.Location, &d.Contact, &lastPoll, &d.LastPollSuccess,
		&d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, err
	}

	d.MACAddress = macAddress.String
	if lastPoll.Valid {
		d.LastPollTime = lastPoll.Time
	}

	return &d, nil
}

// Create создает новое устройство
func (r *DeviceSQLite) Create(ctx context.Context, device *model.Device) error {
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO devices (ip_address, mac_address, snmp_port, snmp_community, snmp_version,
						   name, location, contact)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		device.IPAddress, device.MACAddress, device.SNMPPort, device.SNMPCommunity, device.SNMPVersion,
		device.Name, device.Location, device.Contact)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	device.ID = id

	return nil
}

// Update обновляет устройство
func (r *DeviceSQLite) Update(ctx context.Context, device *model.Device) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE devices SET
			ip_address = ?, mac_address = ?, snmp_port = ?, snmp_community = ?, snmp_version = ?,
			name = ?, location = ?, contact = ?,
			last_poll_time = ?, last_poll_success = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		device.IPAddress, device.MACAddress, device.SNMPPort, device.SNMPCommunity, device.SNMPVersion,
		device.Name, device.Location, device.Contact,
		device.LastPollTime, device.LastPollSuccess,
		device.ID)
	return err
}

// Delete удаляет устройство
func (r *DeviceSQLite) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM devices WHERE id = ?", id)
	return err
}

// SaveMetrics сохраняет метрики
func (r *DeviceSQLite) SaveMetrics(ctx context.Context, deviceID int64, metrics *model.DeviceMetrics) error {
	if metrics == nil {
		return nil
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO device_metrics
		(device_id, radio_output_enabled, temperature, internal_power_1, internal_power_2, updated_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`,
		deviceID, metrics.RadioOutputEnabled, metrics.Temperature,
		metrics.InternalPower1, metrics.InternalPower2)
	return err
}

// SaveAlarms сохраняет алармы
func (r *DeviceSQLite) SaveAlarms(ctx context.Context, deviceID int64, alarms *model.DeviceAlarms) error {
	if alarms == nil {
		return nil
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO device_alarms
		(device_id, eth_los, shortcut, power_up_limit, power_bottom_limit,
		 temperature_limit, channel_1_not_connected, channel_2_not_connected,
		 channel_3_not_connected, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`,
		deviceID, alarms.EthLos, alarms.Shortcut, alarms.PowerUpLimit, alarms.PowerBottomLimit,
		alarms.TemperatureLimit, alarms.Channel1NotConnected, alarms.Channel2NotConnected,
		alarms.Channel3NotConnected)
	return err
}

// SaveChannels сохраняет каналы
func (r *DeviceSQLite) SaveChannels(ctx context.Context, deviceID int64, channels []model.ChannelStatus) error {
	for _, ch := range channels {
		_, err := r.db.ExecContext(ctx, `
			INSERT OR REPLACE INTO device_channels
			(device_id, channel_id, state_link, stream, server, updated_at)
			VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`,
			deviceID, ch.ChannelID, ch.StateLink, ch.Stream, ch.Server)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetMetrics возвращает метрики
func (r *DeviceSQLite) GetMetrics(ctx context.Context, deviceID int64) (*model.DeviceMetrics, error) {
	var m model.DeviceMetrics
	err := r.db.QueryRowContext(ctx, `
		SELECT radio_output_enabled, temperature, internal_power_1, internal_power_2
		FROM device_metrics WHERE device_id = ?`, deviceID).Scan(
		&m.RadioOutputEnabled, &m.Temperature, &m.InternalPower1, &m.InternalPower2)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

// GetAlarms возвращает алармы
func (r *DeviceSQLite) GetAlarms(ctx context.Context, deviceID int64) (*model.DeviceAlarms, error) {
	var a model.DeviceAlarms
	err := r.db.QueryRowContext(ctx, `
		SELECT eth_los, shortcut, power_up_limit, power_bottom_limit,
			   temperature_limit, channel_1_not_connected, channel_2_not_connected,
			   channel_3_not_connected
		FROM device_alarms WHERE device_id = ?`, deviceID).Scan(
		&a.EthLos, &a.Shortcut, &a.PowerUpLimit, &a.PowerBottomLimit,
		&a.TemperatureLimit, &a.Channel1NotConnected, &a.Channel2NotConnected,
		&a.Channel3NotConnected)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

// GetChannels возвращает каналы
func (r *DeviceSQLite) GetChannels(ctx context.Context, deviceID int64) ([]model.ChannelStatus, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT channel_id, state_link, stream, server
		FROM device_channels WHERE device_id = ? ORDER BY channel_id`, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []model.ChannelStatus
	for rows.Next() {
		var ch model.ChannelStatus
		err := rows.Scan(&ch.ChannelID, &ch.StateLink, &ch.Stream, &ch.Server)
		if err != nil {
			return nil, err
		}
		channels = append(channels, ch)
	}
	return channels, nil
}
