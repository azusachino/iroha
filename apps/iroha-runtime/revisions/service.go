// Package revisions provides the primary-Postgres revision boundary for
// cached reads. Revisions are advanced inside the same transaction as the
// canonical mutation that made the read namespace stale.
package revisions

import (
	"sort"

	"gorm.io/gorm"
)

const (
	NamespaceBriefing         = "read_briefing"
	NamespaceActivities       = "read_activities"
	NamespaceSleep            = "read_sleep"
	NamespaceDaily            = "read_daily"
	NamespaceMedia            = "read_media"
	NamespaceMetrics          = "read_metrics"
	NamespaceReports          = "read_reports"
	NamespaceExpenses         = "read_expenses"
	NamespaceCoverage         = "read_coverage"
	NamespacePublicSummary    = "public_summary"
	NamespacePublicActivities = "public_activities"
	NamespacePublicRoutes     = "public_routes"
)

var ImportNamespaces = []string{
	NamespaceBriefing,
	NamespaceActivities,
	NamespaceSleep,
	NamespaceDaily,
	NamespaceMedia,
	NamespaceMetrics,
	NamespaceReports,
	NamespaceCoverage,
	NamespacePublicSummary,
	NamespacePublicActivities,
	NamespacePublicRoutes,
}

// Bump advances each distinct namespace once, using lexical order for every
// multi-namespace mutation. The order is part of the transaction contract and
// prevents two writers from deadlocking while acquiring overlapping scopes.
func Bump(tx *gorm.DB, namespaces ...string) error {
	ordered := orderedNamespaces(namespaces)
	for _, namespace := range ordered {
		if err := tx.Exec("select pg_advisory_xact_lock(hashtext(?))", "iroha:revision:"+namespace).Error; err != nil {
			return err
		}
		if err := tx.Exec(`
insert into tb_read_revisions (namespace, revision, updated_at)
values (?, 1, now())
on conflict (namespace) do update set
  revision = tb_read_revisions.revision + 1,
  updated_at = excluded.updated_at`, namespace).Error; err != nil {
			return err
		}
	}
	return nil
}

// Read returns the committed revision for each requested namespace. Missing
// rows are the initial revision zero and are not created by a read.
func Read(tx *gorm.DB, namespaces ...string) (map[string]int64, error) {
	ordered := orderedNamespaces(namespaces)
	result := make(map[string]int64, len(ordered))
	for _, namespace := range ordered {
		var revision int64
		if err := tx.Raw("select coalesce((select revision from tb_read_revisions where namespace = ?), 0)", namespace).Scan(&revision).Error; err != nil {
			return nil, err
		}
		result[namespace] = revision
	}
	return result, nil
}

func orderedNamespaces(namespaces []string) []string {
	ordered := make([]string, 0, len(namespaces))
	seen := make(map[string]struct{}, len(namespaces))
	for _, namespace := range namespaces {
		if namespace == "" {
			continue
		}
		if _, exists := seen[namespace]; exists {
			continue
		}
		seen[namespace] = struct{}{}
		ordered = append(ordered, namespace)
	}
	sort.Strings(ordered)
	return ordered
}
