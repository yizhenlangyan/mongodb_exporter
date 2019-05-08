package collector

import (
	"sort"
	"strings"
	"sync"
	"time"

	mgo "github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	count = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: "collection",
		Name:      "total_objects",
		Help:      "The number of objects or documents in this collection",
	}, []string{"ns"})

	size = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: "collection",
		Name:      "size_bytes",
		Help:      "The total size in memory of all records in a collection",
	}, []string{"ns"})

	avgObjSize = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: "collection",
		Name:      "avg_objsize_bytes",
		Help:      "The average size of an object in the collection",
	}, []string{"ns"})

	storageSize = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: "collection",
		Name:      "storage_size_bytes",
		Help:      "The total amount of storage allocated to this collection for document storage",
	}, []string{"ns"})

	collIndexSize = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: "collection",
		Name:      "index_size_bytes",
		Help:      "The total size of all indexes",
	}, []string{"ns"})

	// Lock for using these metrics
	collectionStatsLock = sync.Mutex{}
)

type CollectionStatus struct {
	Name        string `bson:"ns"`
	Count       int    `bson:"count"`
	Size        int    `bson:"size"`
	AvgSize     int    `bson:"avgObjSize"`
	StorageSize int    `bson:"storageSize"`
	IndexSize   int    `bson:"totalIndexSize"`
}

func (collStatus *CollectionStatus) Export(ch chan<- prometheus.Metric) {
	collectionStatsLock.Lock()
	defer collectionStatsLock.Unlock()

	count.WithLabelValues(collStatus.Name).Set(float64(collStatus.Count))
	size.WithLabelValues(collStatus.Name).Set(float64(collStatus.Size))
	avgObjSize.WithLabelValues(collStatus.Name).Set(float64(collStatus.AvgSize))
	storageSize.WithLabelValues(collStatus.Name).Set(float64(collStatus.StorageSize))
	collIndexSize.WithLabelValues(collStatus.Name).Set(float64(collStatus.IndexSize))

	count.Collect(ch)
	size.Collect(ch)
	avgObjSize.Collect(ch)
	storageSize.Collect(ch)
	collIndexSize.Collect(ch)

	count.Reset()
	size.Reset()
	avgObjSize.Reset()
	storageSize.Reset()
	collIndexSize.Reset()
}

func (collStatus *CollectionStatus) Describe(ch chan<- *prometheus.Desc) {
	count.Describe(ch)
	size.Describe(ch)
	avgObjSize.Describe(ch)
	storageSize.Describe(ch)
	collIndexSize.Describe(ch)
}

func GetCollectionStatus(session *mgo.Session, db string, collection string, maxTimeMS int64) *CollectionStatus {
	var collStatus CollectionStatus
	err := session.DB(db).Run(bson.D{{"collStats", collection}, {"scale", 1}, {"maxTimeMS", maxTimeMS}}, &collStatus)
	if err != nil {
		glog.Error(err)
		return nil
	}

	return &collStatus
}

type CursorData struct {
	FirstBatch []bson.Raw `bson:"firstBatch"`
	NextBatch  []bson.Raw `bson:"nextBatch"`
	NS         string
	Id         int64
}

// This is a copy of https://github.com/globalsign/mgo/blob/master/session.go#L3959 to inject
// the maxTimeMS argument to the commands.
func GetCollectionNames(session *mgo.Session, dbname string, maxTimeMS int64) (names []string, err error) {
	db := session.DB(dbname)
	// Clone session and set it to Monotonic mode so that the server
	// used for the query may be safely obtained afterwards, if
	// necessary for iteration when a cursor is received.
	cloned := db.Session.Clone()
	cloned.SetMode(mgo.Monotonic, true)
	defer cloned.Close()

	batchSize := int(100)

	// Try with a command.
	var result struct {
		Collections []bson.Raw
		Cursor      CursorData
	}
	err = db.With(cloned).Run(bson.D{{Name: "listCollections", Value: 1}, {Name: "cursor", Value: bson.D{{Name: "batchSize", Value: batchSize}}}, {Name: "maxTimeMS", Value: maxTimeMS}}, &result)
	if err == nil {
		firstBatch := result.Collections
		if firstBatch == nil {
			firstBatch = result.Cursor.FirstBatch
		}
		var iter *mgo.Iter
		ns := strings.SplitN(result.Cursor.NS, ".", 2)
		if len(ns) < 2 {
			iter = db.With(cloned).C("").NewIter(nil, firstBatch, result.Cursor.Id, nil)
		} else {
			iter = cloned.DB(ns[0]).C(ns[1]).NewIter(nil, firstBatch, result.Cursor.Id, nil)
		}
		var coll struct{ Name string }
		for iter.Next(&coll) {
			names = append(names, coll.Name)
		}
		if err := iter.Close(); err != nil {
			return nil, err
		}
		sort.Strings(names)
		return names, err
	}
	if err != nil {
		e, ok := err.(*mgo.QueryError)
		if ok && (e.Code == 59 || e.Code == 13390 || strings.HasPrefix(e.Message, "no such cmd:")) {
			return nil, err
		}
	}

	// Command not yet supported. Query the database instead.
	nameIndex := len(db.Name) + 1
	iter := db.C("system.namespaces").Find(nil).SetMaxTime(time.Duration(maxTimeMS) * time.Millisecond).Iter()
	var coll struct{ Name string }
	for iter.Next(&coll) {
		if strings.Index(coll.Name, "$") < 0 || strings.Index(coll.Name, ".oplog.$") >= 0 {
			names = append(names, coll.Name[nameIndex:])
		}
	}
	if err := iter.Close(); err != nil {
		return nil, err
	}
	sort.Strings(names)
	return names, nil
}

func CollectCollectionStatus(session *mgo.Session, db string, ch chan<- prometheus.Metric, maxTimeMS int64) {
	collection_names, err := GetCollectionNames(session, db, maxTimeMS)
	if err != nil {
		glog.Error("Failed to get collection names for db=" + db)
		return
	}
	for _, collection_name := range collection_names {
		collStats := GetCollectionStatus(session, db, collection_name, maxTimeMS)
		if collStats != nil {
			glog.V(1).Infof("exporting Database Metrics for db=%q, table=%q", db, collection_name)
			collStats.Export(ch)
		}
	}
}
