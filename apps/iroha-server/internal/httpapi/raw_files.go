package httpapi

import (
	"net/http"
	"time"

	"github.com/azusachino/iroha/apps/iroha-server/internal/ids"
	"github.com/azusachino/iroha/apps/iroha-server/internal/models"
	"github.com/azusachino/iroha/apps/iroha-server/internal/rawfiles"
	"github.com/go-chi/chi/v5"
)

var allowedSourceKinds = map[string]bool{
	"apple_health_export": true,
	"gpx":                 true,
	"fit":                 true,
	"tcx":                 true,
	"strava_export":       true,
}

var allowedUploadSources = map[string]bool{
	"web":        true,
	"telegram":   true,
	"cli":        true,
	"ios_bridge": true,
}

type rawFileResponse struct {
	ID               string    `json:"id"`
	SHA256           string    `json:"sha256"`
	OriginalFilename string    `json:"original_filename"`
	ContentType      string    `json:"content_type"`
	SizeBytes        int64     `json:"size_bytes"`
	SourceKind       string    `json:"source_kind"`
	UploadedVia      string    `json:"uploaded_via"`
	CreatedAt        time.Time `json:"created_at"`
	Duplicate        bool      `json:"duplicate,omitempty"`
}

func (s *Server) handleCreateRawFile(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, s.deps.MaxUploadBytes)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	sourceKind := r.FormValue("source_kind")
	if !allowedSourceKinds[sourceKind] {
		writeError(w, http.StatusBadRequest, "invalid source_kind")
		return
	}

	uploadedVia := r.FormValue("uploaded_via")
	if uploadedVia == "" {
		uploadedVia = "web"
	}
	if !allowedUploadSources[uploadedVia] {
		writeError(w, http.StatusBadRequest, "invalid uploaded_via")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	rawFile, duplicate, err := s.deps.RawFileService.Create(rawfiles.CreateInput{
		File:             file,
		OriginalFilename: header.Filename,
		ContentType:      header.Header.Get("Content-Type"),
		SourceKind:       sourceKind,
		UploadedVia:      uploadedVia,
	})
	if err != nil {
		s.deps.Logger.Error("create raw file", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to store raw file")
		return
	}

	status := http.StatusCreated
	if duplicate {
		status = http.StatusOK
	}
	writeJSON(w, status, toRawFileResponse(rawFile, duplicate))
}

func (s *Server) handleListRawFiles(w http.ResponseWriter, _ *http.Request) {
	rawFiles, err := s.deps.RawFileService.List(50)
	if err != nil {
		s.deps.Logger.Error("list raw files", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list raw files")
		return
	}

	response := make([]rawFileResponse, 0, len(rawFiles))
	for _, rawFile := range rawFiles {
		response = append(response, toRawFileResponse(rawFile, false))
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleGetRawFile(w http.ResponseWriter, r *http.Request) {
	rawFile, found, err := s.deps.RawFileService.Get(chi.URLParam(r, "rawFileId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid raw_file id")
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "raw_file not found")
		return
	}
	writeJSON(w, http.StatusOK, toRawFileResponse(rawFile, false))
}

func toRawFileResponse(rawFile models.RawFile, duplicate bool) rawFileResponse {
	return rawFileResponse{
		ID:               ids.Encode(ids.RawFilePrefix, rawFile.ID),
		SHA256:           rawFile.SHA256,
		OriginalFilename: rawFile.OriginalFilename,
		ContentType:      rawFile.ContentType,
		SizeBytes:        rawFile.SizeBytes,
		SourceKind:       rawFile.SourceKind,
		UploadedVia:      rawFile.UploadedVia,
		CreatedAt:        rawFile.CreatedAt,
		Duplicate:        duplicate,
	}
}
