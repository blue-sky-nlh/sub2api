package service

import (
	"fmt"
	"github.com/shopspring/decimal"
)

func ValidateWeeklyMember(m *QuotaPoolWeeklyMember) error {
	if m == nil {
		return fmt.Errorf("weekly member is missing")
	}
	values := []string{m.State.Accepted, m.State.Baseline, m.State.Unattributed, m.State.PendingDelta}
	for _, a := range m.State.Attempts {
		values = append(values, a.Weight)
	}
	for _, points := range m.Contributions {
		values = append(values, points)
	}
	for _, value := range values {
		if value == "" {
			continue
		}
		d, err := decimal.NewFromString(value)
		if err != nil || d.IsNegative() {
			return fmt.Errorf("invalid weekly accounting decimal")
		}
	}
	if m.State.Cycle > 0 {
		if _, err := weeklyCycleCapacity(&m.State); err != nil {
			return err
		}
	}
	if m.State.Cycle < 0 {
		return fmt.Errorf("invalid weekly accounting cycle")
	}
	return nil
}
func weeklyStateBlocksGroup(st *QuotaPoolWeeklyState, group int64) bool {
	if st.Unknown {
		if len(st.UnknownGroups) == 0 {
			return true
		}
		for _, g := range st.UnknownGroups {
			if g == group {
				return true
			}
		}
	}
	if quotaDecimal(st.PendingDelta).IsPositive() {
		found := false
		for _, a := range st.Attempts {
			if a.StartedAt.Before(st.PendingObservedAt) {
				found = true
				if a.GroupID == group {
					return true
				}
			}
		}
		return !found
	}
	return false
}

// The persisted baseline freezes usable capacity for this binding's cycle.
// Ordinary observations, short-term resets and local group adjustments never
// rebase it. Older records already contain the same baseline; missing or invalid
// evidence is an error, never an assumed full account.
func weeklyCycleCapacity(st *QuotaPoolWeeklyState) (decimal.Decimal, error) {
	if st == nil || st.Cycle <= 0 || st.Baseline == "" {
		return decimal.Zero, fmt.Errorf("weekly cycle baseline is missing")
	}
	baseline, err := decimal.NewFromString(st.Baseline)
	if err != nil || baseline.IsNegative() || baseline.GreaterThan(decimal.NewFromInt(100)) {
		return decimal.Zero, fmt.Errorf("invalid weekly cycle baseline")
	}
	return decimal.NewFromInt(100).Sub(baseline), nil
}
