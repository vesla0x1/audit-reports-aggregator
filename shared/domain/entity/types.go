package entity

import (
	"shared/domain/entity/auditprovider"
	"shared/domain/entity/auditreport"
	"shared/domain/entity/download"
	"shared/domain/entity/process"
)

type (
	Download      = download.Download
	Process       = process.Process
	AuditProvider = auditprovider.AuditProvider
	AuditReport   = auditreport.AuditReport
)
