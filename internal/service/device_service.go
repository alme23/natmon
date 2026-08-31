package service

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/alme23/natmon/internal/model"
	"github.com/alme23/natmon/internal/repository"
)

type SNMPClient interface {
	GetDeviceData(ctx context.Context, device *model.Device) error
	GetSystemInfo(ctx context.Context, device *model.Device) error
	GetMetrics(ctx context.Context, device *model.Device) error
	GetAlarms(ctx context.Context, device *model.Device) error
	GetChannels(ctx context.Context, device *model.Device) error
	Get(ctx context.Context, device *model.Device, oid string) (interface{}, error)
	Set(ctx context.Context, device *model.Device, oid string, value interface{}) error
	Walk(ctx context.Context, device *model.Device, oid string) (interface{}, error)
	Close() error
}

type DeviceService struct {
	repo     repository.DeviceRepository
	snmp     SNMPClient
	poller   *Poller
	mu       sync.RWMutex
	updating map[int64]bool
	stopped  bool
	stopOnce sync.Once
}

func NewDeviceService(repo repository.DeviceRepository, snmp SNMPClient) *DeviceService {
	service := &DeviceService{
		repo:     repo,
		snmp:     snmp,
		updating: make(map[int64]bool),
	}

	service.poller = NewPoller(service)

	return service
}

// Start запускает сервис
func (s *DeviceService) Start(ctx context.Context) {
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()

	log.Println("Starting device service...")

	s.poller.Start(ctx)

	devices, err := s.repo.GetAll(ctx)
	if err != nil {
		log.Printf("Failed to load devices: %v", err)
		return
	}

	for _, device := range devices {
		s.poller.AddDevice(device.ID)
	}

	log.Printf("Device service started with %d devices", len(devices))
}

// Stop останавливает сервис
func (s *DeviceService) Stop() {
	s.stopOnce.Do(func() {
		s.mu.Lock()
		s.stopped = true
		s.mu.Unlock()

		s.poller.Stop()
		log.Println("Device service stopped")
	})
}

// PollChannelStatus опрашивает только статусы каналов
func (s *DeviceService) PollChannelStatus(ctx context.Context, id int64) error {
	device, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.snmp.GetChannels(ctx, device); err != nil {
		device.LastPollSuccess = false
		s.repo.Update(ctx, device)
		return err
	}

	device.LastPollSuccess = true
	device.LastPollTime = time.Now()

	if device.Channels != nil {
		if err := s.repo.SaveChannels(ctx, device.ID, device.Channels); err != nil {
			log.Printf("Failed to save channels: %v", err)
		}
	}

	return nil
}

// PollMetrics опрашивает метрики и алармы
func (s *DeviceService) PollMetrics(ctx context.Context, id int64) error {
	device, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Получаем метрики
	if err := s.snmp.GetMetrics(ctx, device); err != nil {
		log.Printf("Failed to get metrics: %v", err)
	} else {
		s.repo.SaveMetrics(ctx, device.ID, device.Metrics)
	}

	// Получаем алармы
	if err := s.snmp.GetAlarms(ctx, device); err != nil {
		log.Printf("Failed to get alarms: %v", err)
	} else {
		s.repo.SaveAlarms(ctx, device.ID, device.Alarms)
	}

	device.LastPollTime = time.Now()
	s.repo.Update(ctx, device)

	return nil
}

// PollInfo опрашивает общую информацию
func (s *DeviceService) PollInfo(ctx context.Context, id int64) error {
	device, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.snmp.GetSystemInfo(ctx, device); err != nil {
		device.LastPollSuccess = false
		s.repo.Update(ctx, device)
		return err
	}

	device.LastPollSuccess = true
	device.LastPollTime = time.Now()

	if err := s.repo.Update(ctx, device); err != nil {
		return err
	}

	return nil
}

// PollDeviceFull выполняет полный опрос устройства (все данные)
func (s *DeviceService) PollDeviceFull(ctx context.Context, id int64) error {
	s.mu.Lock()
	if s.updating[id] {
		s.mu.Unlock()
		return fmt.Errorf("device is already being polled")
	}
	s.updating[id] = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.updating, id)
		s.mu.Unlock()
	}()

	device, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	log.Printf("Full poll for device %s (ID: %d)...", device.IPAddress, id)

	if err := s.snmp.GetDeviceData(ctx, device); err != nil {
		device.LastPollSuccess = false
		device.LastPollTime = time.Now()
		s.repo.Update(ctx, device)
		return err
	}

	device.LastPollSuccess = true
	device.LastPollTime = time.Now()

	if err := s.repo.Update(ctx, device); err != nil {
		return err
	}

	if device.Metrics != nil {
		s.repo.SaveMetrics(ctx, device.ID, device.Metrics)
	}

	if device.Alarms != nil {
		s.repo.SaveAlarms(ctx, device.ID, device.Alarms)
	}

	if device.Channels != nil {
		s.repo.SaveChannels(ctx, device.ID, device.Channels)
	}

	log.Printf("Full poll completed for device %s", device.IPAddress)
	return nil
}

// GetAllDevices возвращает все устройства
func (s *DeviceService) GetAllDevices(ctx context.Context) ([]model.Device, error) {
	devices, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	for i := range devices {
		metrics, _ := s.repo.GetMetrics(ctx, devices[i].ID)
		alarms, _ := s.repo.GetAlarms(ctx, devices[i].ID)
		channels, _ := s.repo.GetChannels(ctx, devices[i].ID)

		devices[i].Metrics = metrics
		devices[i].Alarms = alarms
		devices[i].Channels = channels
	}

	return devices, nil
}

// GetDevice возвращает устройство
func (s *DeviceService) GetDevice(ctx context.Context, id int64) (*model.Device, error) {
	device, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	metrics, _ := s.repo.GetMetrics(ctx, id)
	alarms, _ := s.repo.GetAlarms(ctx, id)
	channels, _ := s.repo.GetChannels(ctx, id)

	device.Metrics = metrics
	device.Alarms = alarms
	device.Channels = channels

	return device, nil
}

// CreateDevice создает устройство
func (s *DeviceService) CreateDevice(ctx context.Context, device *model.Device) error {
	if err := s.repo.Create(ctx, device); err != nil {
		return err
	}

	s.poller.AddDevice(device.ID)
	go s.poller.PollNow(device.ID)

	return nil
}

// UpdateDevice обновляет устройство
func (s *DeviceService) UpdateDevice(ctx context.Context, device *model.Device) error {
	if err := s.repo.Update(ctx, device); err != nil {
		return err
	}

	go s.poller.PollNow(device.ID)

	return nil
}

// DeleteDevice удаляет устройство
func (s *DeviceService) DeleteDevice(ctx context.Context, id int64) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	s.poller.RemoveDevice(id)

	s.mu.Lock()
	delete(s.updating, id)
	s.mu.Unlock()

	return nil
}

// PollNow запускает полный опрос
func (s *DeviceService) PollNow(ctx context.Context, id int64) error {
	return s.poller.PollNow(id)
}

// PollChannelStatusNow запускает опрос каналов
func (s *DeviceService) PollChannelStatusNow(ctx context.Context, id int64) error {
	return s.poller.PollChannelStatus(id)
}
