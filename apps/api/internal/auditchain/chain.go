// Package auditchain provides the hash-chain audit writer.
// It is a dependency-neutral package imported by both auth and middleware
// packages without creating an import cycle.
package auditchain

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sync"

	"go.uber.org/zap"

	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

// SIEMForwarder is the subset of siem.Forwarder used by ChainWriter.
type SIEMForwarder interface {
	Forward(p db.CreateAuditLogParams)
}

// ChainWriter serialises audit log inserts so each entry's hash covers the
// previous entry — tamper-evident hash chain (RBI CSF requirement).
// Any deleted or modified row breaks the chain, detectable via /api/v1/audit/verify.
type ChainWriter struct {
	mu       sync.Mutex
	queries  *db.Queries
	lastSeq  int64
	lastHash string
	logger   *zap.Logger
	siem     SIEMForwarder
}

func NewChainWriter(ctx context.Context, q *db.Queries, logger *zap.Logger) *ChainWriter {
	cw := &ChainWriter{queries: q, logger: logger}
	row, err := q.GetLastAuditEntry(ctx)
	if err == nil {
		cw.lastSeq = row.SeqNum
		cw.lastHash = row.EntryHash
		if cw.lastHash == "" {
			cw.lastHash = "genesis"
		}
	} else {
		cw.lastHash = "genesis"
	}
	return cw
}

func (cw *ChainWriter) WithSIEM(f SIEMForwarder) *ChainWriter {
	cw.siem = f
	return cw
}

// Write inserts an audit entry into the hash chain.
func (cw *ChainWriter) Write(ctx context.Context, p db.CreateAuditLogParams) error {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	seq := cw.lastSeq + 1
	prevHash := cw.lastHash

	actorStr := ""
	if p.ActorID.Valid {
		actorStr = p.ActorID.UUID.String()
	}
	metaStr := "{}"
	if len(p.Metadata) > 0 {
		metaStr = string(p.Metadata)
	}
	raw := fmt.Sprintf("%d|%s|%s|%s|%s|%s", seq, p.EventCode, actorStr, p.Severity, metaStr, prevHash)
	sum := sha256.Sum256([]byte(raw))
	entryHash := fmt.Sprintf("%x", sum)

	p.SeqNum = seq
	p.PrevHash = prevHash
	p.EntryHash = entryHash

	if err := cw.queries.CreateAuditLog(ctx, p); err != nil {
		return err
	}

	cw.lastSeq = seq
	cw.lastHash = entryHash

	if cw.siem != nil {
		cw.siem.Forward(p)
	}
	return nil
}

// VerifyResult is returned by VerifyChain.
type VerifyResult struct {
	Valid        bool   `json:"valid"`
	TotalChecked int    `json:"total_checked"`
	FirstBreakAt *int64 `json:"first_break_at,omitempty"`
	Message      string `json:"message"`
}

// VerifyChain walks the entire chain and re-computes each entry's expected hash.
func VerifyChain(ctx context.Context, q *db.Queries) (VerifyResult, error) {
	const batchSize = 500
	var fromSeq int64
	prevHash := "genesis"
	var checked int

	for {
		rows, err := q.ListAuditChain(ctx, db.ListAuditChainParams{
			FromSeq:  fromSeq,
			RowLimit: batchSize,
		})
		if err != nil {
			return VerifyResult{}, err
		}
		if len(rows) == 0 {
			break
		}

		for _, row := range rows {
			if row.EntryHash == "pre-chain" {
				prevHash = row.EntryHash
				fromSeq = row.SeqNum
				checked++
				continue
			}
			actorStr := ""
			if row.ActorID.Valid {
				actorStr = row.ActorID.UUID.String()
			}
			metaStr := "{}"
			if len(row.Metadata) > 0 {
				metaStr = string(row.Metadata)
			}
			raw := fmt.Sprintf("%d|%s|%s|%s|%s|%s", row.SeqNum, row.EventCode, actorStr, row.Severity, metaStr, prevHash)
			sum := sha256.Sum256([]byte(raw))
			expected := fmt.Sprintf("%x", sum)

			if expected != row.EntryHash {
				seq := row.SeqNum
				return VerifyResult{
					Valid:        false,
					TotalChecked: checked,
					FirstBreakAt: &seq,
					Message:      fmt.Sprintf("chain broken at seq_num=%d", seq),
				}, nil
			}
			prevHash = row.EntryHash
			fromSeq = row.SeqNum
			checked++
		}
		if len(rows) < batchSize {
			break
		}
	}

	return VerifyResult{
		Valid:        true,
		TotalChecked: checked,
		Message:      fmt.Sprintf("chain intact — %d entries verified", checked),
	}, nil
}
