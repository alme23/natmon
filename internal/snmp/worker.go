package snmp

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/alme23/natmon/internal/model"
	"github.com/gosnmp/gosnmp"
)

type DeviceWorker struct {
	device    *model.Device
	client    *gosnmp.GoSNMP
	jobs      chan DeviceJob
	mu        sync.Mutex
	connected bool
	stopCh    chan struct{}
	doneCh    chan struct{}
	wg        sync.WaitGroup
	stopOnce  sync.Once
}

type DeviceJob struct {
	OIDs     []string
	ResultCh chan DeviceResult
	Ctx      context.Context
}

type DeviceResult struct {
	Values map[string]interface{}
	Err    error
}

func NewDeviceWorker(device *model.Device, timeout time.Duration, retries int) *DeviceWorker {
	worker := &DeviceWorker{
		device: device,
		jobs:   make(chan DeviceJob, 100),
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}

	worker.client = &gosnmp.GoSNMP{
		Target:    device.IPAddress,
		Port:      uint16(device.SNMPPort),
		Community: device.SNMPCommunity,
		Version:   gosnmp.Version1,
		Timeout:   timeout,
		Retries:   retries,
		MaxOids:   60,
	}

	worker.wg.Add(1)
	go worker.run()

	return worker
}

func (w *DeviceWorker) run() {
	defer w.wg.Done()
	defer close(w.doneCh)

	log.Printf("Worker for device %s started", w.device.IPAddress)

	for {
		select {
		case job := <-w.jobs:
			result := w.processJob(job)
			select {
			case job.ResultCh <- result:
			case <-job.Ctx.Done():
				log.Printf("Job cancelled for device %s", w.device.IPAddress)
			}
		case <-w.stopCh:
			log.Printf("Worker for device %s stopped", w.device.IPAddress)
			w.disconnect()
			return
		}
	}
}

func (w *DeviceWorker) processJob(job DeviceJob) DeviceResult {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Проверяем контекст
	select {
	case <-job.Ctx.Done():
		return DeviceResult{Err: job.Ctx.Err()}
	default:
	}

	if !w.connected {
		err := w.client.Connect()
		if err != nil {
			return DeviceResult{Err: fmt.Errorf("connection failed: %w", err)}
		}
		w.connected = true
	}

	result, err := w.client.Get(job.OIDs)
	if err != nil {
		w.disconnect()
		return DeviceResult{Err: fmt.Errorf("SNMP GET failed: %w", err)}
	}

	values := make(map[string]interface{})
	for _, variable := range result.Variables {
		values[variable.Name] = formatSNMPValue(variable)
	}

	return DeviceResult{Values: values}
}

func (w *DeviceWorker) disconnect() {
	if w.connected && w.client.Conn != nil {
		w.client.Conn.Close()
	}
	w.connected = false
}

func (w *DeviceWorker) Submit(ctx context.Context, oids []string) (map[string]interface{}, error) {
	resultCh := make(chan DeviceResult, 1)
	job := DeviceJob{
		OIDs:     oids,
		ResultCh: resultCh,
		Ctx:      ctx,
	}

	select {
	case w.jobs <- job:
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("worker is busy")
	}

	select {
	case result := <-resultCh:
		return result.Values, result.Err
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(10 * time.Second):
		return nil, fmt.Errorf("SNMP request timeout")
	}
}

func (w *DeviceWorker) Stop() {
	w.stopOnce.Do(func() {
		close(w.stopCh)
		w.wg.Wait()
	})
}
