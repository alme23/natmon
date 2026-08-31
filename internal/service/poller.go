package service

import (
	"context"
	"log"
	"sync"
	"time"
)

type Poller struct {
	service  *DeviceService
	mu       sync.RWMutex
	workers  map[int64]*DevicePoller
	stopCh   chan struct{}
	wg       sync.WaitGroup
	stopped  bool
	stopOnce sync.Once
}

type DevicePoller struct {
	deviceID  int64
	service   *DeviceService
	stopCh    chan struct{}
	doneCh    chan struct{}
	isPolling bool
	mu        sync.Mutex
	stopOnce  sync.Once
}

func NewPoller(service *DeviceService) *Poller {
	return &Poller{
		service: service,
		workers: make(map[int64]*DevicePoller),
		stopCh:  make(chan struct{}),
	}
}

func (p *Poller) Start(ctx context.Context) {
	p.wg.Add(1)
	go p.run(ctx)
}

func (p *Poller) run(ctx context.Context) {
	defer p.wg.Done()

	log.Println("Poller started")

	for {
		select {
		case <-ctx.Done():
			log.Println("Poller stopped by context")
			p.stopAllWorkers()
			return
		case <-p.stopCh:
			log.Println("Poller stopped")
			p.stopAllWorkers()
			return
		}
	}
}

func (p *Poller) ensureWorker(deviceID int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.stopped {
		return
	}

	if _, exists := p.workers[deviceID]; exists {
		return
	}

	worker := &DevicePoller{
		deviceID: deviceID,
		service:  p.service,
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}

	p.workers[deviceID] = worker

	go worker.run()

	log.Printf("Worker started for device %d", deviceID)
}

func (p *Poller) AddDevice(deviceID int64) {
	p.ensureWorker(deviceID)
}

func (p *Poller) RemoveDevice(deviceID int64) {
	p.mu.Lock()
	worker, exists := p.workers[deviceID]
	if exists {
		delete(p.workers, deviceID)
	}
	p.mu.Unlock()

	if exists && worker != nil {
		worker.Stop()
		log.Printf("Worker stopped for device %d", deviceID)
	}
}

func (p *Poller) PollNow(deviceID int64) error {
	p.mu.RLock()
	worker, exists := p.workers[deviceID]
	p.mu.RUnlock()

	if !exists {
		p.ensureWorker(deviceID)
		p.mu.RLock()
		worker = p.workers[deviceID]
		p.mu.RUnlock()
	}

	if worker == nil {
		return nil
	}

	return worker.pollAll()
}

func (p *Poller) PollChannelStatus(deviceID int64) error {
	p.mu.RLock()
	worker, exists := p.workers[deviceID]
	p.mu.RUnlock()

	if !exists {
		p.ensureWorker(deviceID)
		p.mu.RLock()
		worker = p.workers[deviceID]
		p.mu.RUnlock()
	}

	if worker == nil {
		return nil
	}

	return worker.pollChannelStatus()
}

func (p *Poller) stopAllWorkers() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.stopped = true

	for deviceID, worker := range p.workers {
		worker.Stop()
		delete(p.workers, deviceID)
		log.Printf("Worker stopped for device %d", deviceID)
	}
}

func (p *Poller) Stop() {
	p.stopOnce.Do(func() {
		close(p.stopCh)
		p.wg.Wait()
	})
}

func (w *DevicePoller) run() {
	log.Printf("Device poller %d started", w.deviceID)

	// Таймеры для разных типов опроса
	channelTicker := time.NewTicker(10 * time.Second) // Каналы - каждые 10 секунд
	metricsTicker := time.NewTicker(1 * time.Minute)  // Метрики - каждую минуту
	infoTicker := time.NewTicker(1 * time.Hour)       // Общая информация - раз в час

	defer channelTicker.Stop()
	defer metricsTicker.Stop()
	defer infoTicker.Stop()

	for {
		select {
		case <-w.stopCh:
			log.Printf("Device poller %d stopped", w.deviceID)
			close(w.doneCh)
			return
		case <-channelTicker.C:
			// Опрос статуса каналов (легкий)
			if err := w.pollChannelStatus(); err != nil {
				log.Printf("Channel status poll failed for device %d: %v", w.deviceID, err)
			}
		case <-metricsTicker.C:
			// Опрос метрик (средний)
			if err := w.pollMetrics(); err != nil {
				log.Printf("Metrics poll failed for device %d: %v", w.deviceID, err)
			}
		case <-infoTicker.C:
			// Опрос общей информации (полный)
			if err := w.pollInfo(); err != nil {
				log.Printf("Info poll failed for device %d: %v", w.deviceID, err)
			}
		}
	}
}

// pollChannelStatus опрашивает только статусы каналов
func (w *DevicePoller) pollChannelStatus() error {
	w.mu.Lock()
	if w.isPolling {
		w.mu.Unlock()
		return nil
	}
	w.isPolling = true
	w.mu.Unlock()

	defer func() {
		w.mu.Lock()
		w.isPolling = false
		w.mu.Unlock()
	}()

	ctx := context.Background()
	return w.service.PollChannelStatus(ctx, w.deviceID)
}

// pollMetrics опрашивает метрики и алармы
func (w *DevicePoller) pollMetrics() error {
	w.mu.Lock()
	if w.isPolling {
		w.mu.Unlock()
		return nil
	}
	w.isPolling = true
	w.mu.Unlock()

	defer func() {
		w.mu.Lock()
		w.isPolling = false
		w.mu.Unlock()
	}()

	ctx := context.Background()
	return w.service.PollMetrics(ctx, w.deviceID)
}

// pollInfo опрашивает общую информацию
func (w *DevicePoller) pollInfo() error {
	w.mu.Lock()
	if w.isPolling {
		w.mu.Unlock()
		return nil
	}
	w.isPolling = true
	w.mu.Unlock()

	defer func() {
		w.mu.Lock()
		w.isPolling = false
		w.mu.Unlock()
	}()

	ctx := context.Background()
	return w.service.PollInfo(ctx, w.deviceID)
}

// pollAll выполняет полный опрос
func (w *DevicePoller) pollAll() error {
	w.mu.Lock()
	if w.isPolling {
		w.mu.Unlock()
		return nil
	}
	w.isPolling = true
	w.mu.Unlock()

	defer func() {
		w.mu.Lock()
		w.isPolling = false
		w.mu.Unlock()
	}()

	ctx := context.Background()
	return w.service.PollDeviceFull(ctx, w.deviceID)
}

func (w *DevicePoller) Stop() {
	w.stopOnce.Do(func() {
		close(w.stopCh)
		select {
		case <-w.doneCh:
		case <-time.After(5 * time.Second):
			log.Printf("Timeout waiting for device poller %d to stop", w.deviceID)
		}
	})
}
