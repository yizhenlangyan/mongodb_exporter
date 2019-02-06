package collector

import (
	"reflect"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	readOps  = map[string]bool{"Queries": true, "GetMore": true, "Commands": true}
	writeOps = map[string]bool{"Insert": true, "Remove": true, "Update": true}
)

var (
	topTimeSecondsTotal = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: Namespace,
		Name:      "top_time_seconds_total",
		Help:      "The top command provides operation time, in seconds, for each database collection",
	}, []string{"type", "database", "collection"})
	topCountTotal = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: Namespace,
		Name:      "top_count_total",
		Help:      "The top command provides operation count for each database collection",
	}, []string{"type", "database", "collection"})
	topTimeSecondsAggregateTotal = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: Namespace,
		Name:      "top_time_seconds_aggregate_total",
		Help:      "An aggregate counter for top time seconds for read/write (does not include locks)",
	}, []string{"type"})
	topCountAggregateTotal = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: Namespace,
		Name:      "top_count_aggregate_total",
		Help:      "An aggregate counter for top operations for read/write (does not include locks)",
	}, []string{"type"})
)

// TopStatsMap is a map of top stats
type TopStatsMap map[string]TopStats

// TopcountersStats topcounters stats
type TopcounterStats struct {
	Time  float64 `bson:"time"`
	Count float64 `bson:"count"`
}

// TopCollectionStats top collection stats
type TopStats struct {
	Total     TopcounterStats `bson:"total"`
	ReadLock  TopcounterStats `bson:"readLock"`
	WriteLock TopcounterStats `bson:"writeLock"`
	Queries   TopcounterStats `bson:"queries"`
	GetMore   TopcounterStats `bson:"getmore"`
	Insert    TopcounterStats `bson:"insert"`
	Update    TopcounterStats `bson:"update"`
	Remove    TopcounterStats `bson:"remove"`
	Commands  TopcounterStats `bson:"commands"`
}

// Export exports the data to prometheus.
func (topStats TopStatsMap) Export(ch chan<- prometheus.Metric) {

	totalReadSeconds := float64(0)
	totalReadOps := float64(0)
	totalWriteSeconds := float64(0)
	totalWriteOps := float64(0)
	for collectionNamespace, topStat := range topStats {

		namespace := strings.Split(collectionNamespace, ".")
		database := namespace[0]
		collection := strings.Join(namespace[1:], ".")

		topStatTypes := reflect.TypeOf(topStat)
		topStatValues := reflect.ValueOf(topStat)

		for i := 0; i < topStatValues.NumField(); i++ {

			metric_type := topStatTypes.Field(i).Name

			op_count := topStatValues.Field(i).Field(1).Float()

			op_time_microsecond := topStatValues.Field(i).Field(0).Float()
			op_time_second := float64(op_time_microsecond / 1e6)

			if _, ok := readOps[metric_type]; ok {
				totalReadSeconds += op_time_second
				totalReadOps += op_count
			}
			if _, ok := writeOps[metric_type]; ok {
				totalWriteSeconds += op_time_second
				totalWriteOps += op_count
			}

			topTimeSecondsTotal.WithLabelValues(metric_type, database, collection).Set(op_time_second)
			topCountTotal.WithLabelValues(metric_type, database, collection).Set(op_count)
		}
	}

	// Set aggregate metrics
	topTimeSecondsAggregateTotal.WithLabelValues("Read").Set(totalReadSeconds)
	topTimeSecondsAggregateTotal.WithLabelValues("Write").Set(totalWriteSeconds)
	topCountAggregateTotal.WithLabelValues("Read").Set(totalReadOps)
	topCountAggregateTotal.WithLabelValues("Write").Set(totalWriteOps)

	topTimeSecondsTotal.Collect(ch)
	topCountTotal.Collect(ch)
	topTimeSecondsAggregateTotal.Collect(ch)
	topCountAggregateTotal.Collect(ch)
}

// Describe describes the metrics for prometheus
func (tops TopStatsMap) Describe(ch chan<- *prometheus.Desc) {
	topTimeSecondsTotal.Describe(ch)
	topCountTotal.Describe(ch)
	topTimeSecondsAggregateTotal.Describe(ch)
	topCountAggregateTotal.Describe(ch)
}
