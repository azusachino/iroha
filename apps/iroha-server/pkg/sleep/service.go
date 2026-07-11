package sleep

import (
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/azusachino/iroha/apps/iroha-server/pkg/ids"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const defaultPageLimit = 50

type Cursor struct {
	WakeDate time.Time
	ID       uuid.UUID
}

func EncodeCursor(cursor Cursor) string {
	raw := cursor.WakeDate.UTC().Format("2006-01-02") + "|" + cursor.ID.String()
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func DecodeCursor(value string) (Cursor, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return Cursor{}, errors.New("invalid cursor")
	}
	wakeDateValue, idValue, ok := strings.Cut(string(decoded), "|")
	if !ok {
		return Cursor{}, errors.New("invalid cursor")
	}
	wakeDate, err := time.Parse("2006-01-02", wakeDateValue)
	if err != nil {
		return Cursor{}, errors.New("invalid cursor")
	}
	id, err := uuid.Parse(idValue)
	if err != nil {
		return Cursor{}, errors.New("invalid cursor")
	}
	return Cursor{WakeDate: wakeDate, ID: id}, nil
}

type ListFilters struct {
	From   *time.Time
	To     *time.Time
	Limit  int
	Cursor *Cursor
}

type Page struct {
	Items      []models.SleepSession
	NextCursor *Cursor
	HasMore    bool
}

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) List(filters ListFilters) (Page, error) {
	limit := filters.Limit
	if limit <= 0 || limit > 100 {
		limit = defaultPageLimit
	}

	query := s.db.Model(&models.SleepSession{})
	if filters.From != nil {
		query = query.Where("wake_date >= ?", *filters.From)
	}
	if filters.To != nil {
		query = query.Where("wake_date <= ?", *filters.To)
	}
	if filters.Cursor != nil {
		query = query.Where("(wake_date, id) < (?, ?)", filters.Cursor.WakeDate, filters.Cursor.ID)
	}

	var rows []models.SleepSession
	if err := query.Order("wake_date desc, id desc").Limit(limit + 1).Find(&rows).Error; err != nil {
		return Page{}, err
	}
	page := Page{Items: rows}
	if len(rows) > limit {
		page.Items = rows[:limit]
		page.HasMore = true
		last := page.Items[len(page.Items)-1]
		page.NextCursor = &Cursor{WakeDate: last.WakeDate, ID: last.ID}
	}
	return page, nil
}

func (s *Service) Get(id string) (models.SleepSession, bool, error) {
	decoded, err := ids.Decode(ids.SleepPrefix, id)
	if err != nil {
		return models.SleepSession{}, false, err
	}
	var session models.SleepSession
	err = s.db.First(&session, "id = ?", decoded).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.SleepSession{}, false, nil
	}
	if err != nil {
		return models.SleepSession{}, false, err
	}
	return session, true, nil
}

func (s *Service) Segments(id string) ([]models.SleepSegment, bool, error) {
	session, found, err := s.Get(id)
	if err != nil || !found {
		return nil, found, err
	}
	var segments []models.SleepSegment
	err = s.db.Where("session_id = ?", session.ID).Order("seq asc").Find(&segments).Error
	return segments, true, err
}
