package runner

import (
	"context"

	"github.com/ebnsina/ferrite-ship/internal/insight"
	"github.com/ebnsina/ferrite-ship/internal/steps"
	"github.com/ebnsina/ferrite-ship/internal/store"
)

// StartReclaim frees the space the caller asked for.
//
// A job rather than a request that blocks: pruning several gigabytes of images
// takes a while, and it belongs in the history beside everything else that
// changed the machine.
func (r *Runner) StartReclaim(
	ctx context.Context, userID, serverID string, items []string, actor string,
) (store.Job, error) {
	server, err := r.store.GetServer(ctx, userID, serverID)
	if err != nil {
		return store.Job{}, err
	}

	return r.start(ctx, server, actor, false, plan{
		kind:  "reclaim",
		title: "Freeing space on " + server.Name,
		build: func(context.Context, store.Server) []steps.Step {
			playbook := make([]steps.Step, 0, len(items)+1)

			for _, item := range items {
				commands := insight.Commands(item)
				if commands == nil {
					// Already refused at the API, and refused again here: this
					// is the layer that actually runs as root.
					continue
				}
				playbook = append(playbook, steps.Shell(steps.ShellSpec{
					ID:    "reclaim-" + item,
					Title: insight.Label(item),
					Apply: commands,
				}))
			}

			// Report the result in the log, so the person sees what they got
			// back without reloading the page they started from.
			playbook = append(playbook, steps.Shell(steps.ShellSpec{
				ID:    "reclaim-report",
				Title: "Check what is free now",
				Apply: []string{`df -h / | tail -1`},
			}))

			return playbook
		},
	})
}
