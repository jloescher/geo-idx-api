package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quantyralabs/idx-api/internal/debug"
)

// DefaultLeaderLockKey is the PostgreSQL advisory lock id for cluster-wide scheduler leadership.
const DefaultLeaderLockKey int64 = 913374211

// LeaderSession holds a dedicated pool connection with an acquired advisory lock.
// The lock is session-scoped; release via Release before returning the connection to the pool.
type LeaderSession struct {
	conn      *pgxpool.Conn
	key       int64
	stopPing  context.CancelFunc
	pingDone  sync.WaitGroup
}

// TryAcquireLeader attempts pg_try_advisory_lock on a dedicated connection.
func TryAcquireLeader(ctx context.Context, pool *pgxpool.Pool, key int64) (*LeaderSession, bool, error) {
	if pool == nil {
		return nil, false, fmt.Errorf("nil pool")
	}
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, false, err
	}
	var ok bool
	err = conn.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, key).Scan(&ok)
	if err != nil {
		conn.Release()
		return nil, false, err
	}
	if !ok {
		conn.Release()
		// #region agent log
		debug.Log("A", "leader.go:TryAcquireLeader", "lock not acquired", map[string]any{"key": key})
		// #endregion
		return nil, false, nil
	}
	// #region agent log
	var holderPID int32
	_ = conn.QueryRow(ctx, `
		SELECT pid FROM pg_locks
		WHERE locktype = 'advisory' AND classid = 0 AND objid = $1::bigint AND granted
		LIMIT 1
	`, key).Scan(&holderPID)
	debug.Log("A", "leader.go:TryAcquireLeader", "lock acquired", map[string]any{"key": key, "holderPID": holderPID})
	// #endregion
	sess := &LeaderSession{conn: conn, key: key}
	sess.startKeepalive(ctx)
	return sess, true, nil
}

func (l *LeaderSession) startKeepalive(parent context.Context) {
	pingCtx, cancel := context.WithCancel(parent)
	l.stopPing = cancel
	l.pingDone.Add(1)
	go func() {
		defer l.pingDone.Done()
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-pingCtx.Done():
				return
			case <-ticker.C:
				if l.conn == nil {
					return
				}
				_, _ = l.conn.Exec(pingCtx, `SELECT 1`)
			}
		}
	}()
}

// Release unlocks the advisory lock and returns the connection to the pool.
func (l *LeaderSession) Release(ctx context.Context) {
	if l == nil || l.conn == nil {
		return
	}
	if l.stopPing != nil {
		l.stopPing()
		l.pingDone.Wait()
		l.stopPing = nil
	}
	_, _ = l.conn.Exec(ctx, `SELECT pg_advisory_unlock($1)`, l.key)
	l.conn.Release()
	l.conn = nil
}

// WaitForLeader polls until the advisory lock is acquired or ctx is cancelled.
func WaitForLeader(ctx context.Context, pool *pgxpool.Pool, key int64, poll time.Duration) (*LeaderSession, error) {
	if poll <= 0 {
		poll = 15 * time.Second
	}
	for {
		leader, ok, err := TryAcquireLeader(ctx, pool, key)
		if err != nil {
			return nil, err
		}
		if ok {
			return leader, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(poll):
		}
	}
}
