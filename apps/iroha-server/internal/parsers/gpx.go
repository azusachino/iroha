package parsers

import (
	"encoding/xml"
	"fmt"
	"os"
	"time"
)

type GPXOptions struct {
	Title      string
	ExternalID string
}

type gpxDocument struct {
	Tracks []gpxTrack `xml:"trk"`
}

type gpxTrack struct {
	Name     string       `xml:"name"`
	Segments []gpxSegment `xml:"trkseg"`
}

type gpxSegment struct {
	Points []gpxPoint `xml:"trkpt"`
}

type gpxPoint struct {
	Lat       float64  `xml:"lat,attr"`
	Lon       float64  `xml:"lon,attr"`
	Elevation *float64 `xml:"ele"`
	Time      string   `xml:"time"`
}

func ParseGPXFile(path string, options GPXOptions) ([]ParsedActivity, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var doc gpxDocument
	if err := xml.NewDecoder(file).Decode(&doc); err != nil {
		return nil, err
	}

	var activities []ParsedActivity
	for trackIndex, track := range doc.Tracks {
		points := flattenGPXPoints(track)
		if len(points) == 0 {
			continue
		}

		title := track.Name
		if title == "" {
			title = options.Title
		}
		startedAt := time.Now().UTC()
		if points[0].Ts != nil {
			startedAt = *points[0].Ts
		}
		var endedAt *time.Time
		if points[len(points)-1].Ts != nil {
			value := *points[len(points)-1].Ts
			endedAt = &value
		}

		externalID := options.ExternalID
		if len(doc.Tracks) > 1 {
			externalID = fmt.Sprintf("%s#%d", externalID, trackIndex+1)
		}

		activities = append(activities, ParsedActivity{
			Provider:         "gpx",
			ExternalID:       externalID,
			SportType:        "run",
			Title:            title,
			StartedAt:        startedAt,
			EndedAt:          endedAt,
			SourceKind:       "gpx",
			SourceActivityID: externalID,
			RoutePoints:      points,
		})
	}

	return activities, nil
}

func flattenGPXPoints(track gpxTrack) []RoutePoint {
	var points []RoutePoint
	for _, segment := range track.Segments {
		for _, point := range segment.Points {
			var ts *time.Time
			if point.Time != "" {
				if parsed, err := time.Parse(time.RFC3339, point.Time); err == nil {
					value := parsed.UTC()
					ts = &value
				}
			}
			points = append(points, RoutePoint{
				Ts:         ts,
				Lat:        point.Lat,
				Lon:        point.Lon,
				ElevationM: point.Elevation,
			})
		}
	}
	return points
}
