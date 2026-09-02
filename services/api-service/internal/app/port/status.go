package port

type ComponentStatus struct {
	ID string
	OK bool
}

const (
	ComponentAPI      = "api"
	ComponentWorker   = "worker"
	ComponentPostgres = "postgres"
	ComponentQueue    = "queue"
	ComponentFiles    = "files"
)
