/*
 * Copyright (c) 2022, Gideon Williams gideon@gideonw.com
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package server

import (
	"github.com/dburkart/fossil/pkg/database"
	"github.com/prometheus/client_golang/prometheus"
)

type dbStatsCollector struct {
	db *database.Database

	segments   *prometheus.Desc
	topicCount *prometheus.Desc
}

func NewDBStatsCollector(db *database.Database) prometheus.Collector {
	return &dbStatsCollector{
		db: db,
		segments: prometheus.NewDesc(
			"fossil_database_segments",
			"Number of segments in the database.",
			nil, prometheus.Labels{"db_name": db.Name},
		),
		topicCount: prometheus.NewDesc(
			"fossil_database_topics",
			"Number of topics in the database.",
			nil, prometheus.Labels{"db_name": db.Name},
		),
	}
}

// Describe implements Collector.
func (c *dbStatsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.segments
	ch <- c.topicCount
}

// Collect implements Collector.
func (c *dbStatsCollector) Collect(ch chan<- prometheus.Metric) {
	stats := c.db.Stats()
	ch <- prometheus.MustNewConstMetric(c.segments, prometheus.GaugeValue, float64(stats.Segments))
	ch <- prometheus.MustNewConstMetric(c.topicCount, prometheus.GaugeValue, float64(stats.TopicCount))
}
