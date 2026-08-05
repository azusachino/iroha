package bangumi

type collectionsResponse struct {
	Total int                `json:"total"`
	Data  []collectionRecord `json:"data"`
}

type collectionRecord struct {
	SubjectType int           `json:"subject_type"`
	Type        int           `json:"type"`
	Rate        *float64      `json:"rate"`
	Comment     string        `json:"comment"`
	Tags        []string      `json:"tags"`
	EpStatus    *float64      `json:"ep_status"`
	VolStatus   *float64      `json:"vol_status"`
	Private     bool          `json:"private"`
	UpdatedAt   string        `json:"updated_at"`
	Subject     subjectRecord `json:"subject"`
}

type subjectRecord struct {
	ID       int           `json:"id"`
	Type     int           `json:"type"`
	Name     string        `json:"name"`
	NameCN   string        `json:"name_cn"`
	Platform string        `json:"platform"`
	Date     string        `json:"date"`
	Eps      *int          `json:"eps"`
	Volumes  *int          `json:"volumes"`
	Images   subjectImages `json:"images"`
}

// subjectImages mirrors Bangumi's per-subject image size variants; large is
// the best available for a library card. Not every subject has one (private
// entries, games without art), so this may be empty.
type subjectImages struct {
	Large string `json:"large"`
}
