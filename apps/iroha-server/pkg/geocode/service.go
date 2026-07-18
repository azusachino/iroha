package geocode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	imports "github.com/azusachino/iroha/apps/iroha-imports"
	"github.com/azusachino/iroha/apps/iroha-runtime/cache"
	"github.com/azusachino/iroha/apps/iroha-runtime/jobs"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"gorm.io/gorm"
)

const (
	providerID       = "nominatim"
	geocodeUserAgent = "Iroha-Fitness-Cockpit/1.0"
	geocodeTTL       = 365 * 24 * time.Hour
	queueLease       = 10 * time.Minute
)

type Enqueuer interface {
	EnqueueTx(tx *gorm.DB, kind string, payload any) (models.Job, error)
}

type RefreshPayload struct {
	CoordinateKey string  `json:"coordinate_key"`
	Latitude      float64 `json:"latitude"`
	Longitude     float64 `json:"longitude"`
}

type Service struct {
	db       *gorm.DB
	enqueuer Enqueuer
	cache    *cache.Client
	client   *http.Client
}

func NewService(db *gorm.DB, enqueuer Enqueuer, responseCache *cache.Client) *Service {
	return &Service{db: db, enqueuer: enqueuer, cache: responseCache, client: http.DefaultClient}
}

func (s *Service) LookupCity(ctx context.Context, latitude, longitude float64) (string, bool, error) {
	var row struct {
		City string
	}
	result := s.db.WithContext(ctx).Raw(`
		select city from tb_geocode_cache
		where coordinate_key = ? and expires_at > now()
	`, CoordinateKey(latitude, longitude)).Scan(&row)
	if result.Error != nil {
		return "", false, result.Error
	}
	if result.RowsAffected == 0 {
		return "", false, nil
	}
	return row.City, true, nil
}

func (s *Service) LookupResponse(ctx context.Context, latitude, longitude float64) ([]byte, bool, error) {
	var row struct {
		Response []byte `gorm:"column:response_json"`
	}
	result := s.db.WithContext(ctx).Raw(`
		select response_json from tb_geocode_cache
		where coordinate_key = ? and expires_at > now()
	`, CoordinateKey(latitude, longitude)).Scan(&row)
	if result.Error != nil {
		return nil, false, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, false, nil
	}
	return row.Response, true, nil
}

func (s *Service) EnqueueRefresh(ctx context.Context, latitude, longitude float64) error {
	if s.enqueuer == nil {
		return errors.New("geocode job enqueuer is not configured")
	}
	key := CoordinateKey(latitude, longitude)
	now := time.Now().UTC()
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
			insert into tb_geocode_cache (coordinate_key, latitude, longitude, provider, city, response_json, fetched_at, expires_at, created_at, updated_at)
			values (?, ?, ?, ?, 'Unknown', '{}'::jsonb, ?, ?, ?, ?)
			on conflict (coordinate_key) do nothing
		`, key, latitude, longitude, providerID, time.Unix(0, 0).UTC(), time.Unix(0, 0).UTC(), now, now).Error; err != nil {
			return err
		}

		var row struct {
			RefreshQueuedAt *time.Time `gorm:"column:refresh_queued_at"`
		}
		result := tx.Raw(`select refresh_queued_at from tb_geocode_cache where coordinate_key = ? for update`, key).Scan(&row)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected > 0 && row.RefreshQueuedAt != nil && row.RefreshQueuedAt.After(now.Add(-queueLease)) {
			return nil
		}
		if err := tx.Exec(`update tb_geocode_cache set refresh_queued_at = ?, last_error = null, updated_at = ? where coordinate_key = ?`, now, now, key).Error; err != nil {
			return err
		}
		_, err := s.enqueuer.EnqueueTx(tx, jobs.KindGeocodeRefresh, RefreshPayload{CoordinateKey: key, Latitude: latitude, Longitude: longitude})
		return err
	})
}

func (s *Service) Refresh(ctx context.Context, payload RefreshPayload) error {
	if payload.CoordinateKey == "" {
		payload.CoordinateKey = CoordinateKey(payload.Latitude, payload.Longitude)
	}
	url := fmt.Sprintf("https://nominatim.openstreetmap.org/reverse?lat=%.6f&lon=%.6f&format=json&zoom=10", payload.Latitude, payload.Longitude)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return s.recordFailure(payload.CoordinateKey, err)
	}
	req.Header.Set("User-Agent", geocodeUserAgent)
	resp, err := s.client.Do(req)
	if err != nil {
		return s.recordFailure(payload.CoordinateKey, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return s.recordFailure(payload.CoordinateKey, fmt.Errorf("geocoder returned HTTP %d", resp.StatusCode))
	}

	var data struct {
		Address map[string]string `json:"address"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return s.recordFailure(payload.CoordinateKey, err)
	}
	city := firstNonEmpty(data.Address["city"], data.Address["town"], data.Address["village"], data.Address["city_district"], data.Address["county"], data.Address["state"], "Unknown")
	body, err := json.Marshal(data)
	if err != nil {
		return s.recordFailure(payload.CoordinateKey, err)
	}
	now := time.Now().UTC()
	if err := s.db.WithContext(ctx).Exec(`
		insert into tb_geocode_cache (coordinate_key, latitude, longitude, provider, city, response_json, fetched_at, expires_at, refresh_queued_at, last_error, created_at, updated_at)
		values (?, ?, ?, ?, ?, ?::jsonb, ?, ?, null, null, ?, ?)
		on conflict (coordinate_key) do update set
			latitude = excluded.latitude,
			longitude = excluded.longitude,
			provider = excluded.provider,
			city = excluded.city,
			response_json = excluded.response_json,
			fetched_at = excluded.fetched_at,
			expires_at = excluded.expires_at,
			refresh_queued_at = null,
			last_error = null,
			updated_at = excluded.updated_at
	`, payload.CoordinateKey, payload.Latitude, payload.Longitude, providerID, city, body, now, now.Add(geocodeTTL), now, now).Error; err != nil {
		return err
	}
	if s.cache != nil {
		_ = s.cache.InvalidateNamespace(ctx, "public_routes")
	}
	return nil
}

func (s *Service) recordFailure(key string, cause error) error {
	message := cause.Error()
	_ = s.db.Exec(`update tb_geocode_cache set refresh_queued_at = null, last_error = ?, updated_at = now() where coordinate_key = ?`, message, key)
	return cause
}

func CoordinateKey(latitude, longitude float64) string {
	return fmt.Sprintf("v1:%.2f:%.2f", latitude, longitude)
}

func ParseCoordinate(value string) (float64, error) {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return 0, err
	}
	return parsed, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return "Unknown"
}

var _ imports.Enqueuer = (Enqueuer)(nil)
