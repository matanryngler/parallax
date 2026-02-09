/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// CronJobCyclesStarted tracks the total number of ListCronJob cycles that have started
	CronJobCyclesStarted = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "parallax_cronjob_cycles_started_total",
			Help: "Total number of ListCronJob cycles started",
		},
		[]string{"listcronjob", "namespace"},
	)

	// CronJobCyclesSkipped tracks the total number of skipped cycles
	CronJobCyclesSkipped = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "parallax_cronjob_cycles_skipped_total",
			Help: "Total number of skipped ListCronJob cycles",
		},
		[]string{"listcronjob", "namespace", "reason"},
	)

	// CronJobCycleDuration tracks the duration of the last completed cycle in seconds
	CronJobCycleDuration = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "parallax_cronjob_cycle_duration_seconds",
			Help: "Duration of the last completed ListCronJob cycle in seconds",
		},
		[]string{"listcronjob", "namespace"},
	)

	// CronJobActivePods tracks the current number of active pods processing items for a ListCronJob
	CronJobActivePods = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "parallax_cronjob_active_pods",
			Help: "Current number of active pods processing items for a ListCronJob",
		},
		[]string{"listcronjob", "namespace"},
	)
)

func init() {
	// Register custom metrics with the controller-runtime metrics registry
	metrics.Registry.MustRegister(
		CronJobCyclesStarted,
		CronJobCyclesSkipped,
		CronJobCycleDuration,
		CronJobActivePods,
	)
}
