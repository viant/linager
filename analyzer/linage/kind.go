package linage

type AccessKind string

const (
	Read  AccessKind = "READ"
	Write AccessKind = "WRITE"
	Call  AccessKind = "CALL"
	Xfer  AccessKind = "XFER" // transitive data transfer
	// Metadata represents edges derived from code metadata such as annotations or tags
	Metadata AccessKind = "METADATA"
)
