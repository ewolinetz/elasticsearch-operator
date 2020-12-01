package k8shandler

type Alerts struct {
	heap_high        bool
	low_watermark    bool
	disk_usage_low   bool
	write_rejections bool
}

type AlertManager interface {
	Alerts() *Alerts
}
