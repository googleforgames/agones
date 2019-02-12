package framework

import (
	"sort"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// PerfResults aggregates performance test results.
// The AddSample() method is safe for concurrent use by multiple goroutines.
type PerfResults struct {
	mu      sync.Mutex
	samples []time.Duration

	firstSampleTime time.Time
	lastSampleTime  time.Time
}

// AddSample adds a single time measurement.
func (p *PerfResults) AddSample(d time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()

	n := time.Now()
	if len(p.samples) == 0 {
		p.firstSampleTime = n
	}
	p.lastSampleTime = n
	p.samples = append(p.samples, d)
}

// Report outputs performance report to log.
func (p *PerfResults) Report(name string) {
	if len(p.samples) == 0 {
		return
	}

	sort.Slice(p.samples, func(i, j int) bool {
		return p.samples[i] < p.samples[j]
	})

	var sum time.Duration
	for _, s := range p.samples {
		sum += s
	}

	avg := time.Duration(int64(sum) / int64(len(p.samples)))
	logrus.
		WithField("avg", avg).
		WithField("count", len(p.samples)).
		WithField("min", p.samples[0].Seconds()).
		WithField("max", p.samples[len(p.samples)-1].Seconds()).
		WithField("p50", p.samples[len(p.samples)*500/1001].Seconds()).
		WithField("p90", p.samples[len(p.samples)*900/1001].Seconds()).
		WithField("p95", p.samples[len(p.samples)*950/1001].Seconds()).
		WithField("p99", p.samples[len(p.samples)*990/1001].Seconds()).
		WithField("p999", p.samples[len(p.samples)*999/1001].Seconds()).
		WithField("duration", p.lastSampleTime.Sub(p.firstSampleTime).Seconds()).
		Info(name)

	// TODO - use something like Fortio ("fortio.org/fortio/stats") to
	// generate histogram for long-term storage and analysis.
}
