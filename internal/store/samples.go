package store

import (
	"context"
	"fmt"
	"time"
)

// Sample is a point-in-time snapshot of the whole fleet.
type Sample struct {
	At          time.Time
	ServerCount int
	OnlineCount int
	CPUUsage    float64
	MemoryUsed  int64
	MemoryTotal int64
	DiskUsed    int64
	DiskTotal   int64
}

// SampleFleet computes and stores a snapshot from the current server rows.
// Called after any run that refreshes facts, which is what gives the dashboard
// real history instead of an invented trend line.
func (s *Store) SampleFleet(ctx context.Context) error {
	servers, err := s.ListServers(ctx)
	if err != nil {
		return err
	}

	sample := Sample{At: time.Now().UTC(), ServerCount: len(servers)}

	var reachable int
	for _, srv := range servers {
		if srv.Status == StatusOnline {
			sample.OnlineCount++
		}
		if srv.Status != StatusOffline {
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

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO fleet_samples
			(at, server_count, online_count, cpu_usage, memory_used, memory_total, disk_used, disk_total)
		VALUES (?,?,?,?,?,?,?,?)`,
		formatTime(sample.At), sample.ServerCount, sample.OnlineCount, sample.CPUUsage,
		sample.MemoryUsed, sample.MemoryTotal, sample.DiskUsed, sample.DiskTotal)
	if err != nil {
		return fmt.Errorf("store: insert fleet sample: %w", err)
	}
	return nil
}

// RecentSamples returns up to limit snapshots, oldest first, ready to plot.
func (s *Store) RecentSamples(ctx context.Context, limit int) ([]Sample, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT at, server_count, online_count, cpu_usage, memory_used, memory_total, disk_used, disk_total
		FROM (
			SELECT * FROM fleet_samples ORDER BY at DESC LIMIT ?
		) ORDER BY at ASC`, limit)
	if err != nil {
		return nil, fmt.Errorf("store: list fleet samples: %w", err)
	}
	defer func() { _ = rows.Close() }()

	samples := []Sample{}
	for rows.Next() {
		var (
			sample Sample
			at     string
		)
		if err := rows.Scan(&at, &sample.ServerCount, &sample.OnlineCount, &sample.CPUUsage,
			&sample.MemoryUsed, &sample.MemoryTotal, &sample.DiskUsed, &sample.DiskTotal); err != nil {
			return nil, err
		}
		sample.At = parseTime(at)
		samples = append(samples, sample)
	}
	return samples, rows.Err()
}
