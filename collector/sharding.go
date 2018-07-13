package collector

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	optimeTimestamp = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: "sharding",
		Name:      "last_seen_configserver_optime_timestamp",
		Help:      "Last seen config server optime's timestamp",
	})
	optimeTerm = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: "sharding",
		Name:      "last_seen_configserver_optime_term",
		Help:      "Last seen config server optime's term",
	})
	maxChunkSizeBytes = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: "sharding",
		Name:      "max_chunk_size_bytes",
		Help:      "Maximum chunk size allowed in bytes",
	})
)

// Cursors are the cursor metrics
type Sharding struct {
	LastSeenConfigServerOptime `bson:"lastSeenConfigServerOpTime"`
	MaxChunkSizeInBytes        float64 `bson:"maxChunkSizeInBytes"`
}

type LastSeenConfigServerOptime struct {
	Timestamp float64 `bson:"ts"`
	Term      float64 `bson:"t"`
}

// Export exports the data to prometheus.
func (sharding *Sharding) Export(ch chan<- prometheus.Metric) {
	optimeTimestamp.Set(sharding.LastSeenConfigServerOptime.Timestamp)
	optimeTerm.Set(sharding.LastSeenConfigServerOptime.Term)
	maxChunkSizeBytes.Set(sharding.MaxChunkSizeInBytes)

	optimeTimestamp.Collect(ch)
	optimeTerm.Collect(ch)
	maxChunkSizeBytes.Collect(ch)
}

// Describe describes the metrics for prometheus
func (sharding *Sharding) Describe(ch chan<- *prometheus.Desc) {
	optimeTimestamp.Describe(ch)
	optimeTerm.Describe(ch)
	maxChunkSizeBytes.Describe(ch)
}
