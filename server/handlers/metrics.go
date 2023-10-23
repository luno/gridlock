package handlers

import "github.com/prometheus/client_golang/prometheus"

var httpHandle = prometheus.NewHistogramVec(prometheus.HistogramOpts{
	Namespace: "gridlock",
	Subsystem: "server",
	Name:      "http_handled_seconds",
	Help:      "Handled HTTP request latency",
}, []string{"path"})

func init() {
	prometheus.MustRegister(httpHandle)
}
