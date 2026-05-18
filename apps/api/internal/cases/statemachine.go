package cases

import (
	"errors"

	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

var ErrInvalidTransition   = errors.New("invalid state transition")
var ErrForbiddenTransition = errors.New("role not permitted for this transition")
var ErrReasonRequired      = errors.New("a reason is required for this transition")
var ErrSamePersonApproval  = errors.New("the person who submitted for review cannot also approve")

type roleLevel int

const (
	roleViewer   roleLevel = 1
	roleAnalyst  roleLevel = 2
	roleManager  roleLevel = 3
	roleAdmin    roleLevel = 4
	roleSuperAdmin roleLevel = 5
)

var roleLevels = map[db.UserRole]roleLevel{
	db.UserRoleViewer:     roleViewer,
	db.UserRoleAnalyst:    roleAnalyst,
	db.UserRoleManager:    roleManager,
	db.UserRoleAdmin:      roleAdmin,
	db.UserRoleSuperAdmin: roleSuperAdmin,
}

type transition struct {
	minRole        roleLevel
	requiresReason bool
}

var allowedTransitions = map[db.CaseStatus]map[db.CaseStatus]transition{
	db.CaseStatusDraft: {
		db.CaseStatusOpen: {minRole: roleAnalyst},
	},
	db.CaseStatusOpen: {
		db.CaseStatusInProgress: {minRole: roleAnalyst},
		db.CaseStatusEscalated:  {minRole: roleManager, requiresReason: true},
	},
	db.CaseStatusInProgress: {
		db.CaseStatusPendingReview: {minRole: roleAnalyst},
		db.CaseStatusEscalated:    {minRole: roleManager, requiresReason: true},
	},
	db.CaseStatusPendingReview: {
		db.CaseStatusClosed:      {minRole: roleManager},
		db.CaseStatusInProgress:  {minRole: roleManager, requiresReason: true},
		db.CaseStatusEscalated:   {minRole: roleManager, requiresReason: true},
	},
	db.CaseStatusEscalated: {
		db.CaseStatusInProgress: {minRole: roleAdmin},
	},
}

func validateTransition(from, to db.CaseStatus, role db.UserRole, reason string) error {
	targets, ok := allowedTransitions[from]
	if !ok {
		return ErrInvalidTransition
	}
	t, ok := targets[to]
	if !ok {
		return ErrInvalidTransition
	}
	if roleLevels[role] < t.minRole {
		return ErrForbiddenTransition
	}
	if t.requiresReason && len(reason) < 10 {
		return ErrReasonRequired
	}
	return nil
}
