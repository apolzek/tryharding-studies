package main

import (
	"math"
	"sort"
	"time"
)

// Productivity is the scoring layer.
//
// Model: bucket raw samples into fixed-size slots (MinuteBucket = 60s). For
// each bucket we compute an "activity score" from input events and network
// traffic, normalise it, and classify the bucket as one of:
//
//   - active:      strong user interaction; clearly productive time
//   - light:       low but non-zero interaction (reading, thinking)
//   - network:     no input but meaningful network I/O (video call, upload)
//   - idle:        negligible activity
//
// Productive time = active + light + network (network is treated as
// semi-productive because video calls / uploads are real work).
//
// Over a 24h window we then compute % of the target 8h that was covered.

const (
	MinuteBucket = 60 // seconds per bucket

	// Per-minute thresholds. Tunable — these defaults are calibrated for a
	// knowledge worker on a laptop.
	ThresholdActiveKeys  = 40   // keys/minute -> clearly typing
	ThresholdLightKeys   = 5    // at least poking at something
	ThresholdClicks      = 2    // clicks/minute -> light activity
	ThresholdMouseMoves  = 30   // mouse moves/minute -> light activity
	ThresholdNetBytesMin = 50_000 // ~50 KB/min of real traffic

	// 8h target out of 24h
	ProductiveTargetSec = 8 * 3600
	ReportWindowSec     = 24 * 3600
)

type Bucket struct {
	Start        int64   `json:"start"`        // unix ts, bucket start
	End          int64   `json:"end"`          // unix ts, bucket end
	KeysPressed  int64   `json:"keys_pressed"`
	Clicks       int64   `json:"clicks"`
	MouseMoves   int64   `json:"mouse_moves"`
	Scrolls      int64   `json:"scrolls"`
	RxBytes      int64   `json:"rx_bytes"`
	TxBytes      int64   `json:"tx_bytes"`
	CPUUsedPct   float64 `json:"cpu_used_pct"`
	MemUsedPct   float64 `json:"mem_used_pct"`
	Score        float64 `json:"score"`  // 0..1
	State        string  `json:"state"`  // active | light | network | idle
}

type ProductivityReport struct {
	User             string    `json:"user"`
	WindowStart      int64     `json:"window_start"`
	WindowEnd        int64     `json:"window_end"`
	TotalSamples     int       `json:"total_samples"`
	TotalBuckets     int       `json:"total_buckets"`
	ActiveSec        int64     `json:"active_sec"`
	LightSec         int64     `json:"light_sec"`
	NetworkSec       int64     `json:"network_sec"`
	IdleSec          int64     `json:"idle_sec"`
	NoSampleSec      int64     `json:"no_sample_sec"`
	ProductiveSec    int64     `json:"productive_sec"`
	ProductiveHours  float64   `json:"productive_hours"`
	TargetHours      float64   `json:"target_hours"`
	TargetReachedPct float64   `json:"target_reached_pct"`
	ProductivityScore float64  `json:"productivity_score"` // 0..100
	TotalKeys        int64     `json:"total_keys"`
	TotalClicks      int64     `json:"total_clicks"`
	TotalMouseMoves  int64     `json:"total_mouse_moves"`
	TotalRxBytes     int64     `json:"total_rx_bytes"`
	TotalTxBytes     int64     `json:"total_tx_bytes"`
	PeakHour         int       `json:"peak_hour"`         // 0..23
	FirstActivityTs  int64     `json:"first_activity_ts"`
	LastActivityTs   int64     `json:"last_activity_ts"`
	Buckets          []Bucket  `json:"buckets"`
	HourBreakdown    []float64 `json:"hour_breakdown"` // 24 values, productive seconds / 3600
}

// normalisedScore maps the interaction density of a bucket to 0..1.
func normalisedScore(keys, clicks, moves int64, netBytes int64) float64 {
	// Each component contributes proportionally to its "active" threshold.
	kScore := float64(keys) / float64(ThresholdActiveKeys)
	cScore := float64(clicks) / 10.0
	mScore := float64(moves) / 120.0
	nScore := float64(netBytes) / 2_000_000.0 // 2 MB/min -> full score
	s := 0.55*kScore + 0.2*cScore + 0.1*mScore + 0.15*nScore
	if s > 1 {
		s = 1
	}
	if s < 0 {
		s = 0
	}
	return s
}

func classify(b Bucket) string {
	// "active" requires real input (keys or clicks+moves).
	if b.KeysPressed >= ThresholdActiveKeys ||
		(b.Clicks >= ThresholdClicks && b.MouseMoves >= ThresholdMouseMoves) {
		return "active"
	}
	if b.KeysPressed >= ThresholdLightKeys || b.Clicks >= ThresholdClicks ||
		b.MouseMoves >= ThresholdMouseMoves {
		return "light"
	}
	if b.RxBytes+b.TxBytes >= ThresholdNetBytesMin {
		return "network"
	}
	return "idle"
}

// BuildReport takes a sorted list of samples and produces a productivity
// report for the window [from, to].
func BuildReport(user string, samples []Sample, from, to int64) ProductivityReport {
	// Bucket by minute.
	byBucket := map[int64]*Bucket{}
	for _, s := range samples {
		startT := int64(s.WindowStart)
		endT := int64(s.WindowEnd)
		if endT <= startT {
			endT = startT + 60
		}
		if endT < from || startT > to {
			continue
		}
		// If a sample spans multiple minute buckets, split proportionally by time.
		dur := float64(endT - startT)
		if dur <= 0 {
			dur = 1
		}
		for t := startT; t < endT; {
			bStart := (t / MinuteBucket) * MinuteBucket
			bEnd := bStart + MinuteBucket
			overlap := math.Min(float64(bEnd), float64(endT)) - math.Max(float64(bStart), float64(t))
			if overlap <= 0 {
				break
			}
			frac := overlap / dur
			b, ok := byBucket[bStart]
			if !ok {
				b = &Bucket{Start: bStart, End: bEnd}
				byBucket[bStart] = b
			}
			b.KeysPressed += int64(float64(s.KeysPressed) * frac)
			b.Clicks += int64(float64(s.Clicks) * frac)
			b.MouseMoves += int64(float64(s.MouseMoves) * frac)
			b.Scrolls += int64(float64(s.Scrolls) * frac)
			b.RxBytes += int64(float64(s.RxBytes) * frac)
			b.TxBytes += int64(float64(s.TxBytes) * frac)
			if s.CPUUsedPct > b.CPUUsedPct {
				b.CPUUsedPct = s.CPUUsedPct
			}
			if s.MemUsedPct > b.MemUsedPct {
				b.MemUsedPct = s.MemUsedPct
			}
			t = bEnd
		}
	}

	// Flatten and sort.
	buckets := make([]Bucket, 0, len(byBucket))
	for _, b := range byBucket {
		b.Score = normalisedScore(b.KeysPressed, b.Clicks, b.MouseMoves, b.RxBytes+b.TxBytes)
		b.State = classify(*b)
		buckets = append(buckets, *b)
	}
	sort.Slice(buckets, func(i, j int) bool { return buckets[i].Start < buckets[j].Start })

	rep := ProductivityReport{
		User:         user,
		WindowStart:  from,
		WindowEnd:    to,
		TotalSamples: len(samples),
		TotalBuckets: len(buckets),
		TargetHours:  float64(ProductiveTargetSec) / 3600.0,
		Buckets:      buckets,
		HourBreakdown: make([]float64, 24),
	}

	hourProductive := make([]int64, 24)

	var firstTs, lastTs int64
	for _, b := range buckets {
		dur := b.End - b.Start
		rep.TotalKeys += b.KeysPressed
		rep.TotalClicks += b.Clicks
		rep.TotalMouseMoves += b.MouseMoves
		rep.TotalRxBytes += b.RxBytes
		rep.TotalTxBytes += b.TxBytes
		switch b.State {
		case "active":
			rep.ActiveSec += dur
			rep.ProductiveSec += dur
			hourProductive[time.Unix(b.Start, 0).Hour()] += dur
			if firstTs == 0 || b.Start < firstTs {
				firstTs = b.Start
			}
			if b.End > lastTs {
				lastTs = b.End
			}
		case "light":
			rep.LightSec += dur
			rep.ProductiveSec += dur
			hourProductive[time.Unix(b.Start, 0).Hour()] += dur
			if firstTs == 0 || b.Start < firstTs {
				firstTs = b.Start
			}
			if b.End > lastTs {
				lastTs = b.End
			}
		case "network":
			rep.NetworkSec += dur
			rep.ProductiveSec += dur
			hourProductive[time.Unix(b.Start, 0).Hour()] += dur
			if firstTs == 0 || b.Start < firstTs {
				firstTs = b.Start
			}
			if b.End > lastTs {
				lastTs = b.End
			}
		default:
			rep.IdleSec += dur
		}
	}

	// No-sample seconds = window - (covered bucket time)
	coveredSec := int64(0)
	for _, b := range buckets {
		coveredSec += b.End - b.Start
	}
	rep.NoSampleSec = (to - from) - coveredSec
	if rep.NoSampleSec < 0 {
		rep.NoSampleSec = 0
	}

	rep.ProductiveHours = float64(rep.ProductiveSec) / 3600.0
	rep.TargetReachedPct = 100.0 * float64(rep.ProductiveSec) / float64(ProductiveTargetSec)
	if rep.TargetReachedPct > 200 {
		rep.TargetReachedPct = 200 // clamp to avoid UI overflow
	}
	// Productivity score: mix of target coverage and average bucket score.
	var sumScore float64
	for _, b := range buckets {
		sumScore += b.Score
	}
	avg := 0.0
	if len(buckets) > 0 {
		avg = sumScore / float64(len(buckets))
	}
	// 70% target coverage, 30% intensity
	cov := math.Min(1.0, float64(rep.ProductiveSec)/float64(ProductiveTargetSec))
	rep.ProductivityScore = 100 * (0.7*cov + 0.3*avg)

	rep.FirstActivityTs = firstTs
	rep.LastActivityTs = lastTs

	peak := 0
	var peakVal int64
	for h, v := range hourProductive {
		rep.HourBreakdown[h] = float64(v) / 3600.0
		if v > peakVal {
			peakVal = v
			peak = h
		}
	}
	rep.PeakHour = peak
	return rep
}
