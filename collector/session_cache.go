package collector

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	activeSessionsCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: Namespace,
		Name:      "active_sessions_count",
		Help:      "total number of active sessions in cache",
	})
)

type SessionCacheStats struct {
	ActiveSessionsCount float64 `bson:"activeSessionsCount"`
	/* Other stats to consider:
		"sessionsCollectionJobCount" : 3202,
		"lastSessionsCollectionJobDurationMillis" : 19,
		"lastSessionsCollectionJobTimestamp" : ISODate("2021-08-12T23:35:45.586Z"),
		"lastSessionsCollectionJobEntriesRefreshed" : 1,
		"lastSessionsCollectionJobEntriesEnded" : 0,
		"lastSessionsCollectionJobCursorsClosed" : 0,
		"transactionReaperJobCount" : 3202,
		"lastTransactionReaperJobDurationMillis" : 0,
		"lastTransactionReaperJobTimestamp" : ISODate("2021-08-12T23:35:47.079Z"),
		"lastTransactionReaperJobEntriesCleanedUp" : 0,
		"sessionCatalogSize" : 0
	*/
}

// Export exports the data to prometheus.
func (sessionCacheStats *SessionCacheStats) Export(ch chan<- prometheus.Metric) {
	activeSessionsCount.Set(sessionCacheStats.ActiveSessionsCount)
	activeSessionsCount.Collect(ch)
}

// Describe describes the metrics for prometheus
func (sessionCacheStats *SessionCacheStats) Describe(ch chan<- *prometheus.Desc) {
	activeSessionsCount.Describe(ch)
}