package worker

import (
	"context"

	"github.com/riverqueue/river"
)

type PublishPostArgs struct {
	ProjectID string `json:"project_id"`
	PostID    string `json:"post_id"`
}

func (PublishPostArgs) Kind() string { return "publish_post" }

// PublishFunc is the function signature for executing a scheduled publish.
type PublishFunc func(ctx context.Context, projectID, postID string) error

type PublishPostWorker struct {
	river.WorkerDefaults[PublishPostArgs]
	Publish PublishFunc
}

func (w *PublishPostWorker) Work(ctx context.Context, job *river.Job[PublishPostArgs]) error {
	return w.Publish(ctx, job.Args.ProjectID, job.Args.PostID)
}
