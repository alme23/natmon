package presenter

import (
	"github.com/alme23/natmon/internal/model"
)

type MetricsPresenter struct {
	metrics *model.DeviceMetrics
}

func NewMetricsPresenter(metrics *model.DeviceMetrics) *MetricsPresenter {
	return &MetricsPresenter{metrics: metrics}
}

func (p *MetricsPresenter) HasMetrics() bool {
	return p.metrics != nil
}

func (p *MetricsPresenter) Temperature() string {
	if p.metrics == nil || p.metrics.Temperature == "" {
		return "-"
	}
	return p.metrics.Temperature
}

func (p *MetricsPresenter) InternalPower1() string {
	if p.metrics == nil || p.metrics.InternalPower1 == "" {
		return "-"
	}
	return p.metrics.InternalPower1
}

func (p *MetricsPresenter) InternalPower2() string {
	if p.metrics == nil || p.metrics.InternalPower2 == "" {
		return "-"
	}
	return p.metrics.InternalPower2
}
