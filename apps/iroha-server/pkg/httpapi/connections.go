package httpapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	coreimports "github.com/azusachino/iroha/apps/iroha-imports"
	"github.com/azusachino/iroha/apps/iroha-providers/parsers"
	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-runtime/jobs"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"gorm.io/gorm"
)

const healthDeliveryCadence = 24 * time.Hour

type connectionAction struct {
	Kind   string            `json:"kind"`
	Method string            `json:"method"`
	Path   string            `json:"path"`
	Body   map[string]string `json:"body,omitempty"`
}

type connectionReceipt struct {
	ID            string     `json:"id"`
	RawFileID     string     `json:"raw_file_id"`
	SourceKind    string     `json:"source_kind"`
	IngestionMode string     `json:"ingestion_mode"`
	ReceivedAt    time.Time  `json:"received_at"`
	ObservedAt    *time.Time `json:"observed_at,omitempty"`
}

type connectionImport struct {
	ID           string     `json:"id"`
	Status       string     `json:"status"`
	ParserKind   string     `json:"parser_kind"`
	ErrorMessage *string    `json:"error_message,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
}

type connectionCoverage struct {
	Category     string    `json:"category"`
	From         time.Time `json:"from"`
	To           time.Time `json:"to"`
	Completeness string    `json:"completeness"`
	RecordedAt   time.Time `json:"recorded_at"`
}

type connectionResponse struct {
	ID                string               `json:"id"`
	Provider          string               `json:"provider"`
	InstanceKey       string               `json:"instance_key"`
	DisplayName       string               `json:"display_name"`
	Availability      string               `json:"availability"`
	Collection        string               `json:"collection"`
	Operation         string               `json:"operation"`
	Freshness         string               `json:"freshness"`
	ExpectedIntervalS *int64               `json:"expected_interval_s,omitempty"`
	NextExpectedAt    *time.Time           `json:"next_expected_at,omitempty"`
	LastReceipt       *connectionReceipt   `json:"last_receipt"`
	LastImport        *connectionImport    `json:"last_import"`
	Coverage          []connectionCoverage `json:"coverage"`
	NextActions       []connectionAction   `json:"next_actions"`
}

type connectionsResponse struct {
	Connections []connectionResponse `json:"connections"`
}

type matchingDecisionRequest struct {
	Provider     string `json:"provider"`
	ExternalID   string `json:"external_id"`
	TargetItemID string `json:"target_item_id"`
	DecisionKind string `json:"decision_kind"`
}

type matchingDecisionResponse struct {
	ID           string    `json:"id"`
	Provider     string    `json:"provider"`
	ExternalID   string    `json:"external_id"`
	TargetItemID string    `json:"target_item_id"`
	DecisionKind string    `json:"decision_kind"`
	CreatedAt    time.Time `json:"created_at"`
}

func (s *Server) handleListConnections(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if s.deps.DB == nil {
		writeError(w, http.StatusServiceUnavailable, "connection service unavailable")
		return
	}
	var instances []models.SourceInstance
	if err := s.deps.DB.WithContext(r.Context()).Order("provider, instance_key").Find(&instances).Error; err != nil {
		s.deps.Logger.Error("list connections", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list connections")
		return
	}
	connections := make([]connectionResponse, 0, len(instances))
	for _, instance := range instances {
		connection, err := s.connection(r.Context(), instance)
		if err != nil {
			s.deps.Logger.Error("load connection", "source_instance_id", instance.ID, "error", err)
			writeError(w, http.StatusInternalServerError, "failed to load connection")
			return
		}
		connections = append(connections, connection)
	}
	writeJSON(w, http.StatusOK, connectionsResponse{Connections: connections})
}

func (s *Server) connection(ctx context.Context, instance models.SourceInstance) (connectionResponse, error) {
	db := s.deps.DB.WithContext(ctx)
	response := connectionResponse{
		ID: ids.Encode(ids.SourceInstancePrefix, instance.ID), Provider: instance.Provider, InstanceKey: instance.InstanceKey, DisplayName: instance.DisplayName,
		Availability: "supported", Collection: "unknown", Operation: "idle", Freshness: "unknown", Coverage: []connectionCoverage{}, NextActions: []connectionAction{},
	}
	var receipt models.SourceReceipt
	receiptResult := db.Where("source_instance_id = ?", instance.ID).Order("received_at desc, created_at desc").First(&receipt)
	if receiptResult.Error != nil && !errors.Is(receiptResult.Error, gorm.ErrRecordNotFound) {
		return connectionResponse{}, receiptResult.Error
	}
	if receiptResult.Error == nil {
		response.LastReceipt = &connectionReceipt{ID: ids.Encode(ids.ReceiptPrefix, receipt.ID), RawFileID: ids.Encode(ids.RawFilePrefix, receipt.RawFileID), SourceKind: receipt.SourceKind, IngestionMode: receipt.IngestionMode, ReceivedAt: receipt.ReceivedAt.UTC(), ObservedAt: receipt.ObservedAt}
		var importJob models.ImportJob
		importResult := db.Where("raw_file_id = ?", receipt.RawFileID).Order("created_at desc").First(&importJob)
		if importResult.Error != nil && !errors.Is(importResult.Error, gorm.ErrRecordNotFound) {
			return connectionResponse{}, importResult.Error
		}
		if importResult.Error == nil {
			response.LastImport = &connectionImport{ID: ids.Encode(ids.ImportPrefix, importJob.ID), Status: importJob.Status, ParserKind: importJob.ParserKind, ErrorMessage: importJob.ErrorMessage, CreatedAt: importJob.CreatedAt.UTC(), FinishedAt: importJob.FinishedAt}
			switch importJob.Status {
			case coreimports.StatusQueued, coreimports.StatusParsing:
				response.Operation = "importing"
			case coreimports.StatusFailed:
				response.Operation = "failed"
				response.NextActions = append(response.NextActions, connectionAction{Kind: "retry_import", Method: http.MethodPost, Path: "/api/v1/imports", Body: map[string]string{"raw_file_id": ids.Encode(ids.RawFilePrefix, receipt.RawFileID), "parser_kind": importJob.ParserKind}})
			}
		}
	}
	var assertions []models.SourceCoverageAssertion
	if err := db.Where("source_instance_id = ?", instance.ID).Order("recorded_at desc, created_at desc").Limit(50).Find(&assertions).Error; err != nil {
		return connectionResponse{}, err
	}
	for _, assertion := range assertions {
		response.Coverage = append(response.Coverage, connectionCoverage{Category: assertion.Category, From: assertion.IntervalStart.UTC(), To: assertion.IntervalEnd.UTC(), Completeness: assertion.Completeness, RecordedAt: assertion.RecordedAt.UTC()})
		switch assertion.Completeness {
		case "covered":
			response.Collection = "covered"
		case "covered_empty":
			if response.Collection == "unknown" {
				response.Collection = "covered_empty"
			}
		case "partial":
			if response.Collection == "unknown" {
				response.Collection = "partial"
			}
		}
	}
	if instance.Provider == "anilist" || instance.Provider == "bangumi" {
		response.NextActions = append(response.NextActions, connectionAction{Kind: "sync", Method: http.MethodPost, Path: "/api/v1/media/sync/" + instance.Provider})
	}
	if instance.Provider == "health_auto_export" || instance.Provider == "apple_health" {
		response.NextActions = append(response.NextActions, connectionAction{Kind: "send_bounded_payload", Method: http.MethodPost, Path: "/api/v1/intake/health"})
	}
	interval, err := connectionCadence(db, instance)
	if err != nil {
		return connectionResponse{}, err
	}
	if interval > 0 {
		seconds := int64(interval / time.Second)
		response.ExpectedIntervalS = &seconds
		lastDelivery := instance.CreatedAt
		if response.LastReceipt != nil {
			lastDelivery = response.LastReceipt.ReceivedAt
		}
		deadline := lastDelivery.Add(interval).UTC()
		response.NextExpectedAt = &deadline
		response.Freshness = connectionFreshness(s.clockNow(), deadline, response.LastReceipt != nil)
	} else if interval == 0 {
		response.Freshness = "not_scheduled"
	}
	return response, nil
}

// A negative interval means the schedule is present but cannot be interpreted.
func connectionCadence(db *gorm.DB, instance models.SourceInstance) (time.Duration, error) {
	if instance.Provider == "health_auto_export" {
		var active int64
		err := db.Model(&models.IntakeCredential{}).Where("revoked_at IS NULL AND (? || name) = ?", parsers.HealthAutoExportInstance(""), instance.InstanceKey).Count(&active).Error
		if active > 0 {
			return healthDeliveryCadence, err
		}
		return 0, err
	}
	kind := ""
	switch instance.Provider {
	case "anilist", "anilist_activity":
		kind = jobs.KindMediaSyncAniList
	case "bangumi":
		kind = jobs.KindMediaSyncBangumi
	default:
		return 0, nil
	}
	var schedule models.JobSchedule
	err := db.Where("kind = ?", kind).First(&schedule).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if !schedule.Enabled {
		return 0, nil
	}
	if schedule.ScheduleKind != jobs.ScheduleKindInterval {
		return -1, nil
	}
	interval, err := time.ParseDuration(schedule.ScheduleExpr)
	if err != nil || interval <= 0 {
		return -1, nil
	}
	return interval, nil
}

func connectionFreshness(now, deadline time.Time, delivered bool) string {
	if !now.Before(deadline) {
		return "overdue"
	}
	if !delivered {
		return "unknown"
	}
	return "within_cadence"
}

func (s *Server) handleRecordMatchingDecision(w http.ResponseWriter, r *http.Request) {
	if s.deps.ImportService == nil {
		writeError(w, http.StatusServiceUnavailable, "import service unavailable")
		return
	}
	var request matchingDecisionRequest
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	targetID, err := ids.Decode(ids.MediaPrefix, request.TargetItemID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid target_item_id")
		return
	}
	decision, err := s.deps.ImportService.RecordMediaMatchingDecision(coreimports.MediaMatchingDecisionInput{Provider: request.Provider, ExternalID: request.ExternalID, TargetItemID: targetID, DecisionKind: request.DecisionKind})
	if err != nil {
		s.deps.Logger.Error("record matching decision", "error", err)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, matchingDecisionResponse{ID: decision.ID.String(), Provider: decision.Provider, ExternalID: decision.ExternalID, TargetItemID: ids.Encode(ids.MediaPrefix, decision.TargetItemID), DecisionKind: decision.DecisionKind, CreatedAt: decision.CreatedAt.UTC()})
}
