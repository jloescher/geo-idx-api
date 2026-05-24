package queue

import "encoding/json"

// Payload is the JSON envelope stored in jobs.payload (small; never embed MLS pages).
type Payload struct {
	Type string          `json:"type"`
	Args json.RawMessage `json:"args,omitempty"`
}

// Job types — revenue impact: explicit types enable memory-bounded workers per MLS pipeline stage.
const (
	TypeNoop                       = "noop"
	TypeBridgeFetchPage            = "bridge.fetch_page"
	TypeBridgePersistChunk         = "bridge.persist_chunk"
	TypeBridgePersistFinalize      = "bridge.persist_finalize"
	TypeSparkFetchPage             = "spark.fetch_page"
	TypeSparkPersistChunk          = "spark.persist_chunk"
	TypeSparkPersistFinalize       = "spark.persist_finalize"
	TypeMLSReplicationKickoff      = "mls.replication_kickoff"
	TypeMLSProxyCachePurge         = "mls.proxy_cache_purge"
	TypeMLSPurgeClosed             = "mls.purge_closed_listings"
	TypeMLSPurgeReplicaPages       = "mls.purge_replica_pages"
	TypeGISProbeSources            = "gis.probe_sources"
	TypeGISMonthlyParcelRefresh    = "gis.monthly_parcel_refresh"
	TypeGISAnnualBoundariesRefresh = "gis.annual_boundaries_refresh"
	TypeGISInitialSync             = "gis.initial_sync"
	TypeGISParcelSyncPage          = "gis.parcel_sync_page"
	TypeCryptoRefreshPricing       = "crypto.refresh_pricing"
	TypeBatchComplete              = "queue.batch_complete"
)

func MarshalPayload(typ string, args any) ([]byte, error) {
	var raw json.RawMessage
	if args != nil {
		b, err := json.Marshal(args)
		if err != nil {
			return nil, err
		}
		raw = b
	}
	return json.Marshal(Payload{Type: typ, Args: raw})
}

func UnmarshalPayload(data []byte) (Payload, error) {
	var p Payload
	err := json.Unmarshal(data, &p)
	return p, err
}
