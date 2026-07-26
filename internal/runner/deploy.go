package runner

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ebnsina/ferrite-ship/internal/deploy"
	"github.com/ebnsina/ferrite-ship/internal/steps"
	"github.com/ebnsina/ferrite-ship/internal/store"
)

// StartDeploy fetches, builds and runs an application, then points the proxy
// at it.
func (r *Runner) StartDeploy(
	ctx context.Context, userID, appID, actor string,
) (store.Job, error) {
	app, err := r.store.GetApp(ctx, userID, appID)
	if err != nil {
		return store.Job{}, err
	}

	server, err := r.store.GetServer(ctx, userID, app.ServerID)
	if err != nil {
		return store.Job{}, err
	}

	env, err := r.openEnv(app)
	if err != nil {
		return store.Job{}, err
	}

	deployKey := ""
	if app.SealedDeployKey != "" {
		deployKey, err = r.sealer.Open(app.SealedDeployKey)
		if err != nil {
			return store.Job{}, err
		}
	}

	spec := deploy.App{
		ID: app.ID, Name: app.Name, Repository: app.Repository,
		Branch: app.Branch, Domain: app.Domain, Port: app.Port,
		Env: env, DeployKey: deployKey,
	}

	job, err := r.start(ctx, server, actor, false, plan{
		kind:  "app-deploy",
		title: "Deploying " + app.Name + " to " + server.Name,
		build: func(ctx context.Context, _ store.Server) []steps.Step {
			playbook := steps.Docker()
			playbook = append(playbook, deploy.BuilderSteps()...)
			playbook = append(playbook, deploy.NetworkStep())
			playbook = append(playbook, deploy.AppSteps(spec)...)
			// The proxy is configured last, and from every published
			// application rather than just this one: its config is one file,
			// so writing it from a single app would delete the other routes.
			playbook = append(playbook, deploy.IngressSteps(r.sites(ctx, app.ServerID, app))...)
			return playbook
		},
		secrets: append(secretsOf(env), deployKey),
		onFinish: func(ctx context.Context, _ store.Server, status store.JobStatus) {
			state := store.AppRunning
			var deployed *time.Time
			if status == store.JobSucceeded {
				now := time.Now().UTC()
				deployed = &now
			} else {
				state = store.AppFailed
			}
			if err := r.store.SetAppStatus(ctx, app.ID, state, "", deployed); err != nil {
				r.log.Error("could not record deploy outcome", "app", app.ID, "error", err)
			}
		},
	})
	if err != nil {
		return store.Job{}, err
	}

	if err := r.store.SetAppStatus(ctx, app.ID, store.AppDeploying, job.ID, nil); err != nil {
		return store.Job{}, err
	}
	return job, nil
}

// StartUndeploy stops an application and takes its route away.
func (r *Runner) StartUndeploy(
	ctx context.Context, userID, appID, actor string,
) (store.Job, error) {
	app, err := r.store.GetApp(ctx, userID, appID)
	if err != nil {
		return store.Job{}, err
	}

	server, err := r.store.GetServer(ctx, userID, app.ServerID)
	if err != nil {
		return store.Job{}, err
	}

	spec := deploy.App{ID: app.ID, Name: app.Name, Domain: app.Domain, Port: app.Port}

	return r.start(ctx, server, actor, false, plan{
		kind:  "app-remove",
		title: "Removing " + app.Name + " from " + server.Name,
		build: func(ctx context.Context, _ store.Server) []steps.Step {
			playbook := []steps.Step{deploy.NetworkStep()}
			playbook = append(playbook, deploy.RemoveSteps(spec)...)
			// Rebuild the routes without this application, so its domain stops
			// answering rather than pointing at a container that has gone.
			return append(playbook, deploy.IngressSteps(r.sites(ctx, app.ServerID, store.App{ID: app.ID}))...)
		},
		onFinish: func(ctx context.Context, _ store.Server, status store.JobStatus) {
			if status != store.JobSucceeded {
				if err := r.store.SetAppStatus(ctx, app.ID, store.AppFailed, "", nil); err != nil {
					r.log.Error("could not record removal failure", "app", app.ID, "error", err)
				}
				return
			}
			if err := r.store.DeleteApp(ctx, userID, app.ID); err != nil {
				r.log.Error("could not forget app", "app", app.ID, "error", err)
			}
		},
	})
}

// sites collects every published application on a server for the proxy config.
//
// changing is the application this deploy is about: it is included with its
// current values even though the stored row may lag, and excluded entirely
// when it is being removed.
func (r *Runner) sites(ctx context.Context, serverID string, changing store.App) []deploy.Site {
	published, err := r.store.PublishedApps(ctx, serverID)
	if err != nil {
		r.log.Error("could not read published apps", "server", serverID, "error", err)
		return nil
	}

	sites := make([]deploy.Site, 0, len(published))
	for _, app := range published {
		if app.ID == changing.ID {
			continue
		}
		sites = append(sites, deploy.Site{
			Domain: app.Domain, Container: deploy.Container(app.ID), Port: app.Port,
		})
	}

	if changing.Domain != "" {
		sites = append(sites, deploy.Site{
			Domain:    changing.Domain,
			Container: deploy.Container(changing.ID),
			Port:      changing.Port,
		})
	}
	return sites
}

func (r *Runner) openEnv(app store.App) (map[string]string, error) {
	if app.SealedEnv == "" {
		return map[string]string{}, nil
	}

	plain, err := r.sealer.Open(app.SealedEnv)
	if err != nil {
		return nil, err
	}

	env := map[string]string{}
	if err := json.Unmarshal([]byte(plain), &env); err != nil {
		return nil, err
	}
	return env, nil
}

// secretsOf masks every environment value in the job log. Which of them are
// sensitive is not knowable from here, and treating them all as secret costs
// nothing but a masked build log.
func secretsOf(env map[string]string) []string {
	values := make([]string, 0, len(env))
	for _, value := range env {
		values = append(values, value)
	}
	return values
}
