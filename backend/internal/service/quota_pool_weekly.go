package service

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/gatewayhooks"
	"github.com/shopspring/decimal"
)

// Weekly accounting uses raw account percentage points. API presentation alone
// normalizes by fixed current-cycle capacity. No billing multiplier enters this ledger.
const QuotaPoolPeriodWeekly = "weekly"
const QuotaPoolDefaultDeny = "deny"
const QuotaPoolSnapshotTTL = 2 * time.Minute

type QuotaPoolWeeklySnapshot struct {
	Percent         float64   `json:"percent"`
	ResetAt         time.Time `json:"reset_at"`
	ObservedAt      time.Time `json:"observed_at"`
	FiveHourPercent float64   `json:"five_hour_percent"`
	FiveHourResetAt time.Time `json:"five_hour_reset_at"`
	Plan            string    `json:"plan"`
	Identity        string    `json:"identity"`
}

type QuotaPoolWeeklyAttempt struct {
	ID         string    `json:"id"`
	GroupID    int64     `json:"group_id"`
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
	Weight     string    `json:"weight"`
	Known      bool      `json:"known"`
}

type QuotaPoolWeeklyReset struct {
	ID     string    `json:"id"`
	At     time.Time `json:"at"`
	Before float64   `json:"before"`
}

type QuotaPoolWeeklyState struct {
	Cycle             int64                       `json:"cycle"`
	Snapshot          *QuotaPoolWeeklySnapshot    `json:"snapshot,omitempty"`
	Accepted          string                      `json:"accepted"`
	Baseline          string                      `json:"baseline"`
	Unattributed      string                      `json:"unattributed"`
	PendingDelta      string                      `json:"pending_delta"`
	PendingObservedAt time.Time                   `json:"pending_observed_at"`
	Attempts          []QuotaPoolWeeklyAttempt    `json:"attempts,omitempty"`
	PendingReset      *QuotaPoolWeeklyReset       `json:"pending_reset,omitempty"`
	LastResetID       string                      `json:"last_reset_id,omitempty"`
	RefreshError      bool                        `json:"refresh_error"`
	UnknownGroups     []int64                     `json:"unknown_groups,omitempty"`
	Unknown           bool                        `json:"unknown"`
	Adjustments       []QuotaPoolWeeklyAdjustment `json:"adjustments,omitempty"`
}
type QuotaPoolWeeklyAdjustment struct {
	Actor    string    `json:"actor"`
	At       time.Time `json:"at"`
	GroupID  int64     `json:"group_id"`
	Previous string    `json:"previous"`
}
type QuotaPoolWeeklyMember struct {
	Binding         QuotaPoolAccount
	AccountValid    bool
	AccountIdentity string
	State           QuotaPoolWeeklyState
	// Active-cycle contributions, raw points, kept as decimals through admission.
	Contributions map[int64]string
}
type QuotaPoolWeeklyView struct {
	Pool    QuotaPool
	Members []QuotaPoolWeeklyMember
	Limits  []QuotaPoolGroupLimit
}
type QuotaPoolWeeklyMutation struct {
	Snapshot     *QuotaPoolWeeklySnapshot
	Attempt      *QuotaPoolWeeklyAttempt
	Finish       *QuotaPoolWeeklyAttempt
	Reset        *QuotaPoolWeeklyReset
	RefreshError bool
}
type QuotaPoolWeeklyRepository interface {
	// Locks pool then configured bindings in a single transaction; callback may
	// inspect and update only the selected member, and returns no change on error.
	WithWeeklyAccount(context.Context, int64, int64, func(*QuotaPoolWeeklyView, *QuotaPoolWeeklyMember) error) error
	ListWeeklyAccountIDs(context.Context) ([]int64, error)
	ListWeeklyGroupAccountIDs(context.Context, int64) ([]int64, error)
	ResetWeeklyGroup(context.Context, int64, int64, string) error
}

func quotaDecimal(value string) decimal.Decimal {
	if value == "" {
		return decimal.Zero
	}
	d, err := decimal.NewFromString(value)
	if err != nil {
		panic("invalid persisted weekly decimal: " + value)
	}
	return d
}
func WeeklyCycleKey(cycle int64) string { return strconv.FormatInt(cycle, 10) }
func validWeeklySnapshot(s *QuotaPoolWeeklySnapshot) bool {
	return s != nil && !math.IsNaN(s.Percent) && !math.IsInf(s.Percent, 0) &&
		s.Percent >= 0 && s.Percent <= 100 && !math.IsNaN(s.FiveHourPercent) &&
		!math.IsInf(s.FiveHourPercent, 0) && s.FiveHourPercent >= 0 && s.FiveHourPercent <= 100 &&
		!s.ObservedAt.IsZero() && s.ResetAt.After(s.ObservedAt) &&
		s.FiveHourResetAt.After(s.ObservedAt) && s.Plan != "" && s.Identity != ""
}

// ApplyWeeklyMutation runs under repository row locks. A decreasing percentage
// never lowers the accepted watermark; a new weekly cycle needs positive proof.
func ApplyWeeklyMutation(m *QuotaPoolWeeklyMember, in QuotaPoolWeeklyMutation) error {
	if err := ValidateWeeklyMember(m); err != nil {
		return err
	}
	for _, attempt := range []*QuotaPoolWeeklyAttempt{in.Attempt, in.Finish} {
		if attempt != nil && attempt.Weight != "" {
			weight, err := decimal.NewFromString(attempt.Weight)
			if err != nil || weight.IsNegative() {
				return fmt.Errorf("invalid weekly attempt weight")
			}
		}
	}
	st := &m.State
	if in.RefreshError {
		st.RefreshError = true
	}
	if in.Reset != nil {
		if in.Reset.ID != "" && in.Reset.ID != st.LastResetID && (st.PendingReset == nil || !in.Reset.At.Before(st.PendingReset.At)) {
			reset := *in.Reset
			if st.Snapshot != nil {
				reset.Before = st.Snapshot.Percent
			}
			st.PendingReset = &reset
			st.RefreshError = true
		}
	}
	if in.Attempt != nil {
		if st.Cycle == 0 {
			return fmt.Errorf("weekly cycle not initialized")
		}
		for _, a := range st.Attempts {
			if a.ID == in.Attempt.ID {
				return nil
			}
		}
		if len(st.Attempts) >= 4096 {
			return &gatewayhooks.QuotaError{Status: 503, Code: "quota_pool_evidence_capacity_reached", Message: "Weekly pending evidence requires settlement"}
		}
		st.Attempts = append(st.Attempts, *in.Attempt)
	}
	if in.Finish != nil {
		for i := range st.Attempts {
			a := &st.Attempts[i]
			if a.ID == in.Finish.ID && a.FinishedAt.IsZero() {
				a.FinishedAt = in.Finish.FinishedAt
				a.Weight = in.Finish.Weight
				a.Known = in.Finish.Known
			}
		}
	}
	if in.Snapshot != nil {
		s := in.Snapshot
		if !validWeeklySnapshot(s) {
			st.RefreshError = true
			return nil
		}
		if st.Snapshot != nil && !s.ObservedAt.After(st.Snapshot.ObservedAt) {
			return settleWeekly(m)
		}
		if st.Snapshot != nil && (s.Identity != st.Snapshot.Identity || s.Plan != st.Snapshot.Plan) {
			st.RefreshError = true
			return nil
		}
		newCycle := st.Cycle == 0
		if st.Snapshot != nil && !s.ObservedAt.Before(st.Snapshot.ResetAt) &&
			s.ResetAt.After(st.Snapshot.ResetAt.Add(time.Minute)) {
			newCycle = true
		}
		if st.PendingReset != nil && s.ObservedAt.After(st.PendingReset.At) &&
			(s.Percent < st.PendingReset.Before || s.Percent == 0) {
			// PendingReset is issued only for an authenticated, successful reset of
			// codex_rate_limits including both explicit windows; post-query proves weekly restoration.
			newCycle = true
		}
		if newCycle {
			st.Cycle++
			st.Accepted = decimal.NewFromFloat(s.Percent).String()
			st.Baseline = st.Accepted
			st.Unattributed = "0"
			st.PendingDelta = "0"
			st.Unknown = false
			st.UnknownGroups = nil
			// Old-cycle requests cannot be attributed to the new cycle. Preserve them
			// in the repository's archived state and conservatively freeze this cycle
			// until another verified weekly reset. The old attempt cannot determine
			// how much of its upstream consumption belongs to each cycle.
			for _, a := range st.Attempts {
				if a.FinishedAt.IsZero() {
					st.Unknown = true
					st.UnknownGroups = append(st.UnknownGroups, a.GroupID)
				}
			}
			st.Attempts = nil
			m.Contributions = map[int64]string{}
			if st.PendingReset != nil {
				st.LastResetID = st.PendingReset.ID
				st.PendingReset = nil
			}
		} else {
			delta := decimal.NewFromFloat(s.Percent).Sub(quotaDecimal(st.Accepted))
			if delta.IsPositive() {
				st.PendingDelta = quotaDecimal(st.PendingDelta).Add(delta).String()
				st.PendingObservedAt = s.ObservedAt
				st.Accepted = decimal.NewFromFloat(s.Percent).String()
			}
		}
		if quotaDecimal(st.PendingDelta).IsPositive() {
			st.PendingObservedAt = s.ObservedAt
		}
		snapshot := *s
		st.Snapshot = &snapshot
		st.RefreshError = st.PendingReset != nil || decimal.NewFromFloat(s.Percent).LessThan(quotaDecimal(st.Accepted))
	}
	return settleWeekly(m)
}

func settleWeekly(m *QuotaPoolWeeklyMember) error {
	st := &m.State
	delta := quotaDecimal(st.PendingDelta)
	if !delta.IsPositive() {
		return nil
	} // zero observations retain interval evidence
	eligible := make([]int, 0)
	total := decimal.Zero
	unknown := false
	for i, a := range st.Attempts {
		if !a.StartedAt.Before(st.PendingObservedAt) {
			continue
		}
		if a.FinishedAt.IsZero() || a.FinishedAt.After(st.PendingObservedAt) {
			return nil
		}
		if !a.Known || !quotaDecimal(a.Weight).IsPositive() {
			unknown = true
		}
		eligible = append(eligible, i)
		total = total.Add(quotaDecimal(a.Weight))
	}
	if len(eligible) == 0 || unknown || !total.IsPositive() {
		st.Unattributed = quotaDecimal(st.Unattributed).Add(delta).String()
		st.Unknown = true
		for _, i := range eligible {
			st.UnknownGroups = append(st.UnknownGroups, st.Attempts[i].GroupID)
		}
	} else {
		if m.Contributions == nil {
			m.Contributions = map[int64]string{}
		}
		remaining := delta
		for j, i := range eligible {
			a := st.Attempts[i]
			share := remaining
			if j != len(eligible)-1 {
				share = delta.Mul(quotaDecimal(a.Weight)).Div(total).Truncate(10)
			}
			remaining = remaining.Sub(share)
			m.Contributions[a.GroupID] = quotaDecimal(m.Contributions[a.GroupID]).Add(share).String()
		}
	}
	used := map[int]bool{}
	for _, i := range eligible {
		used[i] = true
	}
	remainingAttempts := st.Attempts[:0]
	for i, a := range st.Attempts {
		if !used[i] {
			remainingAttempts = append(remainingAttempts, a)
		}
	}
	st.Attempts = remainingAttempts
	st.PendingDelta = "0"
	return nil
}

func WeeklyAdmission(view *QuotaPoolWeeklyView, selected *QuotaPoolWeeklyMember, groupID int64, now time.Time, shortTerm bool) error {
	deny := func(status int, code, msg string) error {
		return &gatewayhooks.QuotaError{Status: status, Code: code, Message: msg}
	}
	if view == nil || selected == nil {
		return deny(503, "quota_pool_unavailable", "Weekly quota pool is unavailable")
	}
	if groupID <= 0 || !view.Pool.Enabled || view.Pool.DeletedAt != nil ||
		!selected.Binding.Enabled || selected.Binding.DeletedAt != nil {
		return deny(403, "quota_pool_forbidden", "Quota pool authorization is disabled or missing")
	}
	var limit *QuotaPoolGroupLimit
	sum := 0
	for i := range view.Limits {
		l := &view.Limits[i]
		if l.Enabled && l.DeletedAt == nil {
			sum += l.PercentBasisPoints
			if l.GroupID == groupID {
				limit = l
			}
		}
	}
	if sum > 10000 || limit == nil || limit.PercentBasisPoints <= 0 {
		return deny(403, "quota_pool_forbidden", "No positive weekly allocation for this group")
	}
	n := 0
	capacity := decimal.Zero
	used := decimal.Zero
	plan := ""
	identities := map[string]bool{}
	for _, m := range view.Members {
		if !m.Binding.Enabled || m.Binding.DeletedAt != nil {
			continue
		}
		n++
		s := m.State.Snapshot
		if m.Binding.Platform != PlatformOpenAI || !m.AccountValid || m.AccountIdentity == "" || m.State.Cycle == 0 || m.State.RefreshError ||
			!validWeeklySnapshot(s) || s.Identity != m.AccountIdentity || now.Sub(s.ObservedAt) > QuotaPoolSnapshotTTL ||
			s.ObservedAt.After(now.Add(5*time.Second)) || !s.ResetAt.After(now) {
			return deny(503, "quota_pool_statistics_unavailable", "Verified weekly quota statistics are unavailable or stale")
		}
		if identities[s.Identity] {
			return deny(503, "quota_pool_duplicate_identity", "Pool members share one upstream weekly capacity")
		}
		identities[s.Identity] = true
		if plan == "" {
			plan = s.Plan
		} else if s.Plan != plan {
			return deny(503, "quota_pool_incomparable_accounts", "Pool accounts have incompatible weekly capacity")
		}
		if weeklyStateBlocksGroup(&m.State, groupID) {
			return deny(503, "quota_pool_settlement_pending", "Weekly consumption attribution is unresolved")
		}
		memberCapacity, err := weeklyCycleCapacity(&m.State)
		if err != nil {
			return deny(503, "quota_pool_statistics_unavailable", "Verified weekly cycle capacity is unavailable")
		}
		capacity = capacity.Add(memberCapacity)
		used = used.Add(quotaDecimal(m.Contributions[groupID]))
	}
	if n == 0 {
		return deny(503, "quota_pool_empty", "Weekly quota pool has no configured members")
	}
	threshold := capacity.Mul(decimal.NewFromInt(int64(limit.PercentBasisPoints))).Div(decimal.NewFromInt(10000))
	if used.GreaterThanOrEqual(threshold) {
		return deny(429, "quota_pool_weekly_exhausted", "Group weekly quota pool allocation is exhausted")
	}
	if shortTerm {
		s := selected.State.Snapshot
		if s != nil && s.Percent >= 100 {
			return deny(429, "quota_pool_account_weekly_limited", "Account upstream weekly quota is exhausted")
		}
		if s == nil || s.FiveHourPercent >= 100 || !s.FiveHourResetAt.After(now) {
			return deny(429, "quota_pool_account_short_term_limited", "Account five-hour quota is unavailable")
		}
	}
	return nil
}
