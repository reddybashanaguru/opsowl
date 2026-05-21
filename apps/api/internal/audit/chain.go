package audit

import (
	"context"

	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
	"github.com/reddybashanaguru/opsowl/internal/auditchain"
)

// Re-export types from auditchain so existing code can use audit.VerifyChain / audit.VerifyResult.
type VerifyResult = auditchain.VerifyResult

func VerifyChain(ctx context.Context, q *db.Queries) (VerifyResult, error) {
	return auditchain.VerifyChain(ctx, q)
}
