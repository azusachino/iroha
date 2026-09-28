package httpapi

import (
	"net/http"
	"sync"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/go-chi/httprate"
)

const (
	// intakeRequestsPerCredentialPerHour covers two hourly HAE automations
	// plus manual retries with a wide margin.
	intakeRequestsPerCredentialPerHour = 120
	// intakeBytesPerCredentialPerDay caps how much disk one credential can
	// fill in a day, so a leaked token cannot exhaust the raw-file volume.
	intakeBytesPerCredentialPerDay int64 = 512 << 20
)

// limitPerIntakeCredential rate-limits authenticated intake per credential.
// It must run after requireIntakeCredential.
func (s *Server) limitPerIntakeCredential(next http.Handler) http.Handler {
	return httprate.LimitBy(intakeRequestsPerCredentialPerHour, time.Hour, func(r *http.Request) (string, error) {
		credential, _ := r.Context().Value(intakeCredentialKey{}).(models.IntakeCredential)
		return credential.ID.String(), nil
	}, httprate.WithLimitHandler(rateLimitResponse))(next)
}

// intakeQuota tracks bytes accepted per credential per UTC day. It is
// in-memory, which is exact for the single iroha-server replica.
type intakeQuota struct {
	mu   sync.Mutex
	now  func() time.Time
	day  string
	used map[string]int64
}

func newIntakeQuota(now func() time.Time) *intakeQuota {
	return &intakeQuota{now: now, used: map[string]int64{}}
}

// take reserves n bytes for credential and reports whether they fit today.
func (q *intakeQuota) take(credential string, n int64) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	if day := q.now().UTC().Format(time.DateOnly); day != q.day {
		q.day = day
		q.used = map[string]int64{}
	}
	if q.used[credential]+n > intakeBytesPerCredentialPerDay {
		return false
	}
	q.used[credential] += n
	return true
}
