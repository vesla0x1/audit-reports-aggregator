package ports

import (
	shared "shared/application/ports"
)

type (
	Queue           = shared.Queue
	QueueMessage    = shared.QueueMessage
	Storage         = shared.Storage
	ObjectMetadata  = shared.ObjectMetadata
	Database        = shared.Database
	Runtime         = shared.Runtime
	Repositories    = shared.Repositories
	Logger          = shared.Logger
	Metrics         = shared.Metrics
	HTTPClient      = shared.HTTPClient
	Observability   = shared.Observability
	RuntimeRequest  = shared.RuntimeRequest
	RuntimeResponse = shared.RuntimeResponse
)
