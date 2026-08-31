package handler

import (
	"log"
	"net"
	"net/http"
	"strconv"

	"github.com/a-h/templ"
	"github.com/alme23/natmon/internal/model"
	"github.com/alme23/natmon/internal/service"
	"github.com/alme23/natmon/web/templates/components"
	"github.com/gorilla/mux"
)

type DeviceHandler struct {
	deviceService *service.DeviceService
}

func NewDeviceHandler(deviceService *service.DeviceService) *DeviceHandler {
	return &DeviceHandler{
		deviceService: deviceService,
	}
}

// GetDevicesTable возвращает всю таблицу (только при первой загрузке)
func (h *DeviceHandler) GetDevicesTable(w http.ResponseWriter, r *http.Request) {
	devices, err := h.deviceService.GetAllDevices(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templ.Handler(components.DeviceTable(devices)).ServeHTTP(w, r)
}

// GetDeviceRow возвращает только одну строку таблицы
func (h *DeviceHandler) GetDeviceRow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid device ID", http.StatusBadRequest)
		return
	}

	device, err := h.deviceService.GetDevice(r.Context(), id)
	if err != nil {
		http.Error(w, "Device not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templ.Handler(components.DeviceRow(*device)).ServeHTTP(w, r)
}

// GetDeviceDetails возвращает детали устройства
func (h *DeviceHandler) GetDeviceDetails(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid device ID", http.StatusBadRequest)
		return
	}

	device, err := h.deviceService.GetDevice(r.Context(), id)
	if err != nil {
		http.Error(w, "Device not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templ.Handler(components.DeviceDetails(*device)).ServeHTTP(w, r)
}

// CreateDeviceForm обрабатывает создание устройства
func (h *DeviceHandler) CreateDeviceForm(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	ipAddress := r.FormValue("ip_address")
	snmpPort, _ := strconv.ParseUint(r.FormValue("snmp_port"), 10, 16)
	snmpCommunity := r.FormValue("snmp_community")

	if net.ParseIP(ipAddress) == nil {
		http.Error(w, "Invalid IP address", http.StatusBadRequest)
		return
	}

	device := &model.Device{
		IPAddress:     ipAddress,
		SNMPPort:      161,
		SNMPCommunity: "private",
		SNMPVersion:   1,
	}

	if snmpPort > 0 {
		device.SNMPPort = uint16(snmpPort)
	}
	if snmpCommunity != "" {
		device.SNMPCommunity = snmpCommunity
	}

	if err := h.deviceService.CreateDevice(r.Context(), device); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Возвращаем только новую строку
	device, _ = h.deviceService.GetDevice(r.Context(), device.ID)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templ.Handler(components.DeviceRow(*device)).ServeHTTP(w, r)
}

// EditDeviceForm возвращает форму редактирования
func (h *DeviceHandler) EditDeviceForm(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid device ID", http.StatusBadRequest)
		return
	}

	device, err := h.deviceService.GetDevice(r.Context(), id)
	if err != nil {
		http.Error(w, "Device not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templ.Handler(components.EditDeviceForm(*device)).ServeHTTP(w, r)
}

// UpdateDeviceForm обрабатывает обновление устройства
func (h *DeviceHandler) UpdateDeviceForm(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid device ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	device, err := h.deviceService.GetDevice(r.Context(), id)
	if err != nil {
		http.Error(w, "Device not found", http.StatusNotFound)
		return
	}

	ipAddress := r.FormValue("ip_address")
	snmpPort, _ := strconv.ParseUint(r.FormValue("snmp_port"), 10, 16)
	snmpCommunity := r.FormValue("snmp_community")

	if ipAddress != "" && net.ParseIP(ipAddress) != nil {
		device.IPAddress = ipAddress
	}
	if snmpPort > 0 {
		device.SNMPPort = uint16(snmpPort)
	}
	if snmpCommunity != "" {
		device.SNMPCommunity = snmpCommunity
	}

	if err := h.deviceService.UpdateDevice(r.Context(), device); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Возвращаем обновленную строку
	device, _ = h.deviceService.GetDevice(r.Context(), id)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templ.Handler(components.DeviceRow(*device)).ServeHTTP(w, r)
}

// DeleteDeviceForm обрабатывает удаление устройства
func (h *DeviceHandler) DeleteDeviceForm(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid device ID", http.StatusBadRequest)
		return
	}

	if err := h.deviceService.DeleteDevice(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Возвращаем обновленную таблицу
	devices, err := h.deviceService.GetAllDevices(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templ.Handler(components.DeviceTable(devices)).ServeHTTP(w, r)
}

// PollDeviceForm обрабатывает опрос устройства
func (h *DeviceHandler) PollDeviceForm(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid device ID", http.StatusBadRequest)
		return
	}

	log.Printf("Polling device %d...", id)

	err = h.deviceService.PollNow(r.Context(), id)
	if err != nil {
		log.Printf("Failed to poll device %d: %v", id, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Возвращаем обновленную строку
	device, err := h.deviceService.GetDevice(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templ.Handler(components.DeviceRow(*device)).ServeHTTP(w, r)
}
