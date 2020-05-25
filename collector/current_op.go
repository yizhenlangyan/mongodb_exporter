package collector

import (
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	instanceFsyncLockWorker = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: "instance",
		Name:      "fsync_lock_worker",
		Help:      "The value of the fsync field corresponds to whether the fsyncLockWorker is active or not.",
	})
)

// CurrentOp keeps the data returned by the currentOp() method.
type CurrentOp struct {
	FsyncLockWorker bool   `bson:"fsyncLock"`
}

// Export exports the current operation status to be consumed by prometheus.
func (status *CurrentOp) Export(ch chan<- prometheus.Metric) {
	var floatVar = float64(0)
	if (status.FsyncLockWorker){
		floatVar = float64(1)
	}
	instanceFsyncLockWorker.Set(floatVar)
	instanceFsyncLockWorker.Collect(ch)
}

// Describe describes the current operation status for prometheus.
func (status *CurrentOp) Describe(ch chan<- *prometheus.Desc) {
	instanceFsyncLockWorker.Describe(ch)
}

// GetCurrentOp returns the current operation info.
func GetCurrentOp(session *mgo.Session, maxTimeMS int64) *CurrentOp {
	result := &CurrentOp{}
	err := session.DB("admin").Run(bson.D{{"currentOp", 1}, {"notexist", 0}}, result)
	if err != nil {
		glog.Error("Failed to get currentOp status.")
		return nil
	}
	return result
}
