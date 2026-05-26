package mls

// MirrorProvider identifies MLS replication source for persist mapping.
type MirrorProvider string

const (
	MirrorProviderBridge MirrorProvider = "bridge"
	MirrorProviderSpark  MirrorProvider = "spark"
)
