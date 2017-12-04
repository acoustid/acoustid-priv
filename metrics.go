package priv

import "github.com/prometheus/client_golang/prometheus"

var catalogActionCount = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "acoustid_priv",
		Name: "catalog_action_total",
		Help: "Number of catalog actions partitioned by action type",
	}, []string{"action"})

var trackActionCount = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "acoustid_priv",
		Name: "track_action_total",
		Help: "Number of track actions partitioned by action type",
	}, []string{"action"})

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
	prometheus.MustRegister(catalogActionCount)
	prometheus.MustRegister(trackActionCount)
	prometheus.MustRegister(searchCount)
	prometheus.MustRegister(searchDuration)
}