package repository

import (
	"context"

	"github.com/alme23/natmon/internal/model"
)

type DeviceRepository interface {
	GetAll(ctx context.Context) ([]model.Device, error)
	GetByID(ctx context.Context, id int64) (*model.Device, error)
	Create(ctx context.Context, device *model.Device) error
	Update(ctx context.Context, device *model.Device) error
	Delete(ctx context.Context, id int64) error

	SaveMetrics(ctx context.Context, deviceID int64, metrics *model.DeviceMetrics) error
	SaveAlarms(ctx context.Context, deviceID int64, alarms *model.DeviceAlarms) error
	SaveChannels(ctx context.Context, deviceID int64, channels []model.ChannelStatus) error

	GetMetrics(ctx context.Context, deviceID int64) (*model.DeviceMetrics, error)
	GetAlarms(ctx context.Context, deviceID int64) (*model.DeviceAlarms, error)
	GetChannels(ctx context.Context, deviceID int64) ([]model.ChannelStatus, error)

	Close() error
}
