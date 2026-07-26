package api

import (
	"net/http"

	"github.com/ebnsina/ferrite-ship/internal/store"
)

// metricView matches the FleetMetric type the dashboard consumes.
type metricView struct {
	ID             string    `json:"id"`
	Label          string    `json:"label"`
	Value          float64   `json:"value"`
	Format         string    `json:"format"`
	DeltaRatio     float64   `json:"deltaRatio"`
	HigherIsBetter bool      `json:"higherIsBetter"`
	Series         []float64 `json:"series"`
}

func (a *API) handleMetrics(w http.ResponseWriter, r *http.Request) {
	servers, err := a.store.ListServers(r.Context(), currentUser(r).ID)
	if err != nil {
		a.writeStoreError(w, err)
		return
	}

	// Up to a fortnight of snapshots, oldest first.
	samples, err := a.store.RecentSamples(r.Context(), currentUser(r).ID, 14)
	if err != nil {
		a.writeStoreError(w, err)
		return
	}

	current := currentSample(servers)

	metrics := []metricView{
		{
			ID:             "servers",
			Label:          "Servers connected",
			Value:          float64(current.ServerCount),
			Format:         "count",
			HigherIsBetter: true,
			Series:         series(samples, func(s store.Sample) float64 { return float64(s.ServerCount) }),
		},
		{
			ID:             "online",
			Label:          "Running fine",
			Value:          float64(current.OnlineCount),
			Format:         "count",
			HigherIsBetter: true,
			Series:         series(samples, func(s store.Sample) float64 { return float64(s.OnlineCount) }),
		},
		{
			ID:             "busy",
			Label:          "How busy they are",
			Value:          current.CPUUsage,
			Format:         "percent",
			HigherIsBetter: false,
			Series:         series(samples, func(s store.Sample) float64 { return s.CPUUsage }),
		},
		{
			ID:             "storage",
			Label:          "Storage in use",
			Value:          float64(current.DiskUsed),
			Format:         "bytes",
			HigherIsBetter: false,
			Series:         series(samples, func(s store.Sample) float64 { return float64(s.DiskUsed) }),
		},
	}

	// Deltas come from the recorded history. With fewer than two snapshots
	// there is no trend to report, and inventing one would be worse than
	// showing nothing.
	for i := range metrics {
		metrics[i].DeltaRatio = delta(metrics[i].Series)
	}

	writeJSON(w, http.StatusOK, metrics)
}

func currentSample(servers []store.Server) store.Sample {
	sample := store.Sample{ServerCount: len(servers)}

	var reachable int
	for _, srv := range servers {
		if srv.Status == store.StatusOnline {
			sample.OnlineCount++
		}
		if srv.Status != store.StatusOffline {
			sample.CPUUsage += srv.Facts.CPUUsage
			reachable++
		}
		sample.MemoryUsed += srv.Facts.MemoryUsedBytes
		sample.MemoryTotal += srv.Facts.MemoryTotalBytes
		sample.DiskUsed += srv.Facts.DiskUsedBytes
		sample.DiskTotal += srv.Facts.DiskTotalBytes
	}
	if reachable > 0 {
		sample.CPUUsage /= float64(reachable)
	}
	return sample
}

func series(samples []store.Sample, pick func(store.Sample) float64) []float64 {
	// A single point is not a line; the UI hides sparklines shorter than two.
	if len(samples) < 2 {
		return []float64{}
	}
	values := make([]float64, 0, len(samples))
	for _, sample := range samples {
		values = append(values, pick(sample))
	}
	return values
}

func delta(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	first, last := values[0], values[len(values)-1]
	if first == 0 {
		return 0
	}
	return (last - first) / first
}
