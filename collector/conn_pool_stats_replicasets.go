package collector

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	pingTime = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: "connpoolstats",
		Name:      "pingtime",
		Help:      "Corresponds to the total number of client connections to mongo.",
	}, []string{"host", "rs"})

	// Lock for using these metrics
	connPoolReplicaSetStatsLock = sync.Mutex{}
)

type ReplicaSetHostStats struct {
	Host     string  `bson:"addr"`
	PingTime float64 `bson:"pingTimeMillis"`
}

type ReplicaSetStats struct {
	Hosts []*ReplicaSetHostStats `bson:"hosts"`
}

// Export exports the server status to be consumed by prometheus.
func (stats *ReplicaSetStats) Export(replicaSet string, ch chan<- prometheus.Metric) {
	connPoolReplicaSetStatsLock.Lock()
	defer connPoolReplicaSetStatsLock.Unlock()

	for _, rsHostStat := range stats.Hosts {
		pingTime.WithLabelValues(rsHostStat.Host, replicaSet).Set(rsHostStat.PingTime)
		pingTime.Collect(ch)
		pingTime.Reset()
	}
}

// Describe describes the server status for prometheus.
func (stats *ReplicaSetStats) Describe(ch chan<- *prometheus.Desc) {
	pingTime.Describe(ch)
}
