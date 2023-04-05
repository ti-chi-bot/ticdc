// Copyright 2020 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package processor

import (
	"github.com/pingcap/tiflow/cdc/processor/memquota"
	"github.com/pingcap/tiflow/cdc/processor/pipeline"
	"github.com/pingcap/tiflow/cdc/processor/sinkmanager"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	syncTableNumGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "ticdc",
			Subsystem: "processor",
			Name:      "num_of_tables",
			Help:      "number of synchronized table of processor",
		}, []string{"namespace", "changefeed"})
	processorErrorCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "ticdc",
			Subsystem: "processor",
			Name:      "exit_with_error_count",
			Help:      "counter for processor exits with error",
		}, []string{"namespace", "changefeed"})
	processorSchemaStorageGcTsGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "ticdc",
			Subsystem: "processor",
			Name:      "schema_storage_gc_ts",
			Help:      "the TS of the currently maintained oldest snapshot in SchemaStorage",
		}, []string{"namespace", "changefeed"})
	processorTickDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "ticdc",
			Subsystem: "processor",
			Name:      "processor_tick_duration",
			Help:      "Bucketed histogram of processorManager tick processor time (s).",
			Buckets:   prometheus.ExponentialBuckets(0.01 /* 10 ms */, 2, 18),
		}, []string{"namespace", "changefeed"})
	processorCloseDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "ticdc",
			Subsystem: "processor",
			Name:      "processor_close_duration",
			Help:      "Bucketed histogram of processorManager close processor time (s).",
			Buckets:   prometheus.ExponentialBuckets(0.01 /* 10 ms */, 2, 18),
		})

	tableMemoryHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "ticdc",
			Subsystem: "processor",
			Name:      "table_memory_consumption",
			Help:      "each table's memory consumption after sorter, in bytes",
			Buckets:   prometheus.ExponentialBuckets(256, 2.0, 20),
		}, []string{"namespace", "changefeed"})

	processorMemoryGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "ticdc",
			Subsystem: "processor",
			Name:      "memory_consumption",
			Help:      "processor's memory consumption estimated in bytes",
		}, []string{"namespace", "changefeed"})

	remainKVEventsGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "ticdc",
			Subsystem: "processor",
			Name:      "remain_kv_events",
			Help:      "processor's kv events that remained in sorter",
		}, []string{"namespace", "changefeed"})
)

// InitMetrics registers all metrics used in processor
func InitMetrics(registry *prometheus.Registry) {
	registry.MustRegister(syncTableNumGauge)
	registry.MustRegister(processorErrorCounter)
	registry.MustRegister(processorSchemaStorageGcTsGauge)
	registry.MustRegister(processorTickDuration)
	registry.MustRegister(processorCloseDuration)
	registry.MustRegister(tableMemoryHistogram)
	registry.MustRegister(processorMemoryGauge)
	registry.MustRegister(remainKVEventsGauge)
	pipeline.InitMetrics(registry)
	sinkmanager.InitMetrics(registry)
	memquota.InitMetrics(registry)
}
