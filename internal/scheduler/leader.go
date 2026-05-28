package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/quantyralabs/idx-api/internal/debug"
)

// DefaultLeaderLockKey is the PostgreSQL advisory lock id for cluster-wide scheduler leadership.
const DefaultLeaderLockKey int64 = 913374211

// LeaderSession holds a dedicated PostgreSQL connection with a session advisory lock.
// Uses pgx.Connect (not pgxpool) so HAProxy/pooler paths cannot drop the lock between statements.
type LeaderSession struct {
	conn     *pgx.Conn
	key      int64
	stopPing context.CancelFunc
	pingDone sync.WaitGroup
}

// TryAcquireLeader opens a dedicated connection and attempts pg_try_advisory_lock.
func TryAcquireLeader(ctx context.Context, dsn string, key int64) (*LeaderSession, bool, error) {
	if dsn == "" {
		return nil, false, fmt.Errorf("empty dsn")
	}
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return nil, false, err
	}
	var ok bool
	err = conn.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, key).Scan(&ok)
	if err != nil {
		conn.Close(ctx)
		return nil, false, err
	}
	held, err := lockHeldBySession(ctx, conn, key)
	if err != nil {
		conn.Close(ctx)
		return nil, false, err
	}
	if !ok || !held {
		conn.Close(ctx)
		// #region agent log
		debug.Log("F", "leader.go:TryAcquireLeader", "lock not held after try", map[string]any{
			"key": key, "tryOk": ok, "held": held,
		})
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

func lockHeldBySession(ctx context.Context, conn *pgx.Conn, key int64) (bool, error) {
	var held bool
	err := conn.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM pg_locks
			WHERE locktype = 'advisory'
			  AND classid = 0
			  AND objid = $1::bigint
			  AND granted
			  AND pid = pg_backend_pid()
		)
	`, key).Scan(&held)
	return held, err
}

func (l *LeaderSession) startKeepalive(parent context.Context) {
	pingCtx, cancel := context.WithCancel(parent)
	l.stopPing = cancel
	l.pingDone.Add(1)
	go func() {
		defer l.pingDone.Done()
		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-pingCtx.Done():
				return
			case <-ticker.C:
				if l.conn == nil {
					return
				}
				if _, err := l.conn.Exec(pingCtx, `SELECT 1`); err != nil {
					// #region agent log
					debug.Log("D", "leader.go:keepalive", "ping failed", map[string]any{"error": err.Error()})
					// #endregion
					return
				}
			}
		}
	}()
}

// Release unlocks the advisory lock and closes the dedicated connection.
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
	_ = l.conn.Close(ctx)
	l.conn = nil
}

// WaitForLeader polls until the advisory lock is acquired or ctx is cancelled.
func WaitForLeader(ctx context.Context, dsn string, key int64, poll time.Duration) (*LeaderSession, error) {
	if poll <= 0 {
		poll = 15 * time.Second
	}
	for {
		leader, ok, err := TryAcquireLeader(ctx, dsn, key)
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
