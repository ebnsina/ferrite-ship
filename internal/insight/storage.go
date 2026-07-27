// Package insight answers questions about a server that a number alone cannot.
//
// A storage bar at 95% tells you there is a problem and nothing about what to
// do next, which leaves the terminal as the only way forward — and avoiding
// that is the point of this product. This finds what is actually using the
// space, and what can safely be given back.
package insight

import (
	"context"
	"strconv"
	"strings"

	"github.com/ebnsina/ferrite-ship/internal/steps"
)

// Report is what is using a server's disk.
type Report struct {
	// TotalBytes and UsedBytes describe the root filesystem.
	TotalBytes int64 `json:"totalBytes"`
	UsedBytes  int64 `json:"usedBytes"`
	FreeBytes  int64 `json:"freeBytes"`

	// Directories are the largest paths, biggest first.
	Directories []Directory `json:"directories"`
	// Reclaimable are the things that can be deleted without losing anything
	// you put there.
	Reclaimable []Reclaimable `json:"reclaimable"`
}

// Directory is one path and what it holds.
type Directory struct {
	Path  string `json:"path"`
	Bytes int64  `json:"bytes"`
	// Depth is how many levels below the root it sits, so the UI can indent
	// rather than show a flat list where /var and /var/lib look unrelated.
	Depth int `json:"depth"`
}

// Reclaimable is space that can be recovered, and how.
//
// Bytes is an upper bound, not a promise. Docker reports what each thing would
// free on its own, but image layers are shared — deleting two images that sit
// on the same base frees that base once. Measured on a real server, 5.3 GB of
// "reclaimable" returned 0.9 GB. The wording in the UI says "up to" for that
// reason.
type Reclaimable struct {
	// ID is what the caller passes back to actually reclaim it.
	ID    string `json:"id"`
	Label string `json:"label"`
	// Detail says plainly what is deleted and what the cost is.
	Detail string `json:"detail"`
	Bytes  int64  `json:"bytes"`
}

// TotalReclaimable is the sum, for a headline figure.
func (r Report) TotalReclaimable() int64 {
	var total int64
	for _, item := range r.Reclaimable {
		total += item.Bytes
	}
	return total
}

// Gather inspects a server.
//
// Every command is tolerant of failure: a machine without Docker should still
// report its directories, and one where du hits a permission problem should
// still report what it could read. A partial answer is far more use than an
// error page.
func Gather(ctx context.Context, session *steps.Session) (Report, error) {
	report := Report{Directories: []Directory{}, Reclaimable: []Reclaimable{}}

	if out, err := session.Capture(ctx, `df -Pk / | tail -1`); err == nil {
		fields := strings.Fields(out)
		if len(fields) >= 4 {
			report.TotalBytes = kilobytes(fields[1])
			report.UsedBytes = kilobytes(fields[2])
			report.FreeBytes = kilobytes(fields[3])
		}
	}

	report.Directories = directories(ctx, session)
	report.Reclaimable = reclaimable(ctx, session)

	return report, nil
}

// directories finds the biggest paths two levels down.
//
// Two levels rather than one because one level says "/var is large", which
// everybody could have guessed. Two says /var/lib/docker, which is the actual
// answer most of the time. -x keeps it on the root filesystem so a mounted
// backup drive is not counted as if it were filling the disk.
func directories(ctx context.Context, session *steps.Session) []Directory {
	out, err := session.Capture(ctx,
		`du -xk --max-depth=2 / 2>/dev/null | sort -rn | head -40`)
	if err != nil {
		return []Directory{}
	}

	found := make([]Directory, 0, 40)
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		size, path, ok := strings.Cut(strings.TrimSpace(line), "\t")
		if !ok {
			continue
		}

		bytes := kilobytes(size)
		// Below about 50 MB nothing is worth acting on, and a list of forty
		// entries where thirty are noise is a list nobody reads.
		if bytes < 50<<20 || path == "/" {
			continue
		}

		found = append(found, Directory{
			Path:  path,
			Bytes: bytes,
			Depth: strings.Count(strings.TrimSuffix(path, "/"), "/"),
		})
	}

	if len(found) > 12 {
		found = found[:12]
	}
	return found
}

// reclaimable finds space that can be given back without losing data.
//
// Deliberately conservative. `docker system prune --volumes` would free the
// most and would also delete every database this product installs, so it is
// not offered at any price. Nothing here removes something a person put on the
// machine on purpose.
func reclaimable(ctx context.Context, session *steps.Session) []Reclaimable {
	items := make([]Reclaimable, 0, 4)

	if out, err := session.Capture(ctx,
		`docker system df --format '{{.Type}}\t{{.Reclaimable}}' 2>/dev/null`); err == nil {
		for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
			kind, amount, ok := strings.Cut(line, "\t")
			if !ok {
				continue
			}

			bytes := dockerSize(amount)
			if bytes <= 0 {
				continue
			}

			switch strings.TrimSpace(kind) {
			case "Images":
				items = append(items, Reclaimable{
					ID:    "docker-images",
					Label: "Container images nothing is using",
					Detail: "Old versions left behind by updates and deployments. " +
						"Anything still running keeps its image; the next deploy re-downloads what it needs.",
					Bytes: bytes,
				})
			case "Build Cache":
				items = append(items, Reclaimable{
					ID:    "docker-build-cache",
					Label: "Leftovers from building",
					Detail: "Intermediate layers kept to make the next build faster. " +
						"Deleting them costs nothing but a slower build next time.",
					Bytes: bytes,
				})
			}
		}
	}

	if out, err := session.Capture(ctx, `du -sk /var/cache/apt/archives 2>/dev/null`); err == nil {
		if size, _, ok := strings.Cut(strings.TrimSpace(out), "\t"); ok {
			if bytes := kilobytes(size); bytes > 10<<20 {
				items = append(items, Reclaimable{
					ID:     "apt-cache",
					Label:  "Downloaded package files",
					Detail: "The installers for software already installed. They are not needed again.",
					Bytes:  bytes,
				})
			}
		}
	}

	if out, err := session.Capture(ctx, `journalctl --disk-usage 2>/dev/null`); err == nil {
		if bytes := journalSize(out); bytes > 200<<20 {
			items = append(items, Reclaimable{
				ID:    "journal",
				Label: "Old system logs",
				Detail: "Trimmed to the most recent 200 MB. " +
					"You keep recent history and lose the oldest entries.",
				Bytes: bytes - 200<<20,
			})
		}
	}

	return items
}

// Commands returns what to run to reclaim an item, or nil for an unknown id.
//
// A map rather than string building at the call site: these run as root on
// someone's server, so the set of things that can possibly run is written down
// in one place and nothing assembles a command from a request.
func Commands(id string) []string {
	switch id {
	case "docker-images":
		return []string{`docker image prune -af`}
	case "docker-build-cache":
		return []string{`docker builder prune -af`}
	case "apt-cache":
		return []string{`apt-get clean`}
	case "journal":
		return []string{`journalctl --vacuum-size=200M`}
	default:
		return nil
	}
}

// Label names an action for the job log.
func Label(id string) string {
	switch id {
	case "docker-images":
		return "Delete container images nothing is using"
	case "docker-build-cache":
		return "Delete leftovers from building"
	case "apt-cache":
		return "Delete downloaded package files"
	case "journal":
		return "Trim old system logs"
	default:
		return "Reclaim space"
	}
}

func kilobytes(field string) int64 {
	value, err := strconv.ParseInt(strings.TrimSpace(field), 10, 64)
	if err != nil {
		return 0
	}
	return value * 1024
}

// dockerSize reads "3.875GB (97%)" and similar.
func dockerSize(text string) int64 {
	text = strings.TrimSpace(text)
	if open := strings.IndexByte(text, '('); open > 0 {
		text = strings.TrimSpace(text[:open])
	}

	units := []struct {
		suffix string
		scale  float64
	}{
		{"TB", 1 << 40}, {"GB", 1 << 30}, {"MB", 1 << 20}, {"kB", 1 << 10}, {"B", 1},
	}

	for _, unit := range units {
		if !strings.HasSuffix(text, unit.suffix) {
			continue
		}
		number := strings.TrimSpace(strings.TrimSuffix(text, unit.suffix))
		value, err := strconv.ParseFloat(number, 64)
		if err != nil {
			return 0
		}
		return int64(value * unit.scale)
	}
	return 0
}

// journalSize reads "Archived and active journals take up 24.0M in the file system."
func journalSize(text string) int64 {
	fields := strings.Fields(text)
	for _, field := range fields {
		if size := suffixed(field); size > 0 {
			return size
		}
	}
	return 0
}

func suffixed(field string) int64 {
	units := map[byte]float64{'K': 1 << 10, 'M': 1 << 20, 'G': 1 << 30, 'T': 1 << 40}

	if len(field) < 2 {
		return 0
	}
	scale, ok := units[field[len(field)-1]]
	if !ok {
		return 0
	}
	value, err := strconv.ParseFloat(field[:len(field)-1], 64)
	if err != nil {
		return 0
	}
	return int64(value * scale)
}
