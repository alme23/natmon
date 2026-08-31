package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/a-h/templ"
	"github.com/alme23/natmon/internal/config"
	"github.com/alme23/natmon/internal/handler"
	"github.com/alme23/natmon/internal/repository"
	"github.com/alme23/natmon/internal/service"
	"github.com/alme23/natmon/internal/snmp"
	"github.com/alme23/natmon/web/templates/pages"
	"github.com/gorilla/mux"
)

const (
	shutdownTimeout = 10 * time.Second // Максимальное время на завершение
)

func main() {
	// Загружаем конфигурацию
	cfg := config.Load()

	// Создаем корневой контекст
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Инициализируем базу данных
	repo, err := repository.NewDeviceSQLite(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Инициализируем SNMP клиент
	snmpClient := snmp.NewManager(cfg.SNMPTimeout, cfg.SNMPRetries)

	// Инициализируем сервис
	deviceService := service.NewDeviceService(repo, snmpClient)

	// Запускаем сервис с пулом воркеров
	deviceService.Start(ctx)

	// Инициализируем обработчик
	deviceHandler := handler.NewDeviceHandler(deviceService)

	// Настраиваем роутер
	router := mux.NewRouter()

	// Статические файлы
	router.PathPrefix("/static/").Handler(
		http.StripPrefix("/static/", http.FileServer(http.Dir("./web/static"))),
	)

	// Главная страница
	router.Handle("/", templ.Handler(pages.IndexPage())).Methods("GET")

	// htmx маршруты
	router.HandleFunc("/devices/table", deviceHandler.GetDevicesTable).Methods("GET")
	router.HandleFunc("/devices/{id}/row", deviceHandler.GetDeviceRow).Methods("GET")
	router.HandleFunc("/devices/{id}/details", deviceHandler.GetDeviceDetails).Methods("GET")
	router.HandleFunc("/devices/create", deviceHandler.CreateDeviceForm).Methods("POST")
	router.HandleFunc("/devices/{id}/edit", deviceHandler.EditDeviceForm).Methods("GET")
	router.HandleFunc("/devices/{id}/update", deviceHandler.UpdateDeviceForm).Methods("POST")
	router.HandleFunc("/devices/{id}/delete", deviceHandler.DeleteDeviceForm).Methods("POST")
	router.HandleFunc("/devices/{id}/poll", deviceHandler.PollDeviceForm).Methods("POST")

	// Создаем HTTP сервер
	server := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Канал для ошибок сервера
	serverErr := make(chan error, 1)

	// Запускаем сервер в горутине
	go func() {
		log.Printf("NATMon server starting on port %s", cfg.ServerPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// Канал для сигналов ОС
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	// Ожидаем сигнал или ошибку
	select {
	case sig := <-sigChan:
		log.Printf("Received signal: %v", sig)
	case err := <-serverErr:
		log.Printf("Server error: %v", err)
		cancel()
		deviceService.Stop()
		snmpClient.Close()
		repo.Close()
		os.Exit(1)
	}

	// Начинаем graceful shutdown
	log.Println("Starting graceful shutdown...")

	// Создаем контекст с таймаутом для shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()

	// 1. Отменяем корневой контекст (останавливает фоновые задачи)
	cancel()

	// 2. Останавливаем пул воркеров
	log.Println("Stopping device pollers...")
	deviceService.Stop()

	// 3. Закрываем SNMP клиент
	log.Println("Closing SNMP client...")
	if err := snmpClient.Close(); err != nil {
		log.Printf("Error closing SNMP client: %v", err)
	}

	// 4. Останавливаем HTTP сервер
	log.Println("Shutting down HTTP server...")
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
		if err := server.Close(); err != nil {
			log.Printf("HTTP server close error: %v", err)
		}
	}

	// 5. Закрываем базу данных
	log.Println("Closing database...")
	if err := repo.Close(); err != nil {
		log.Printf("Error closing database: %v", err)
	}

	log.Println("Graceful shutdown completed")
}
