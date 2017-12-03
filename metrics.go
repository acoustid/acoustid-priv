package priv

import "github.com/prometheus/client_golang/prometheus"

var searchCount = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "acoustid_priv",
		Name: "search_total",
		Help: "Number of searches partitioned by type",
	}, []string{"type"})

var searchDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Namespace: "acoustid_priv",
		Name: "search_duration_seconds",
		Help: "Histogram of search durations partitioned by type",
		Buckets: prometheus.ExponentialBuckets(0.025, 1.5, 10),
	}, []string{"type"})

func init() {
	prometheus.MustRegister(searchCount)
	prometheus.MustRegister(searchDuration)
}