package csvimport

import (
	"encoding/csv"
	"io"
	"strings"
)

const maxFileBytes = 5 << 20  // 5 MB
const maxRows      = 5000

// parseCSV reads all records from r, enforcing row and returns headers + rows.
// Returns ErrTooManyRows or ErrEmptyFile on violations.
func parseCSV(r io.Reader) (headers []string, rows [][]string, err error) {
	cr := csv.NewReader(r)
	cr.TrimLeadingSpace = true
	cr.LazyQuotes = true

	headers, err = cr.Read()
	if err == io.EOF {
		return nil, nil, ErrEmptyFile
	}
	if err != nil {
		return nil, nil, err
	}
	for i, h := range headers {
		headers[i] = strings.TrimSpace(h)
	}

	for {
		record, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue // skip malformed rows
		}
		rows = append(rows, record)
		if len(rows) > maxRows {
			return nil, nil, ErrTooManyRows
		}
	}

	if len(rows) == 0 {
		return nil, nil, ErrEmptyFile
	}
	return headers, rows, nil
}

// autoDetectMapping tries to match CSV headers to known case fields using
// simple substring matching (case-insensitive).
func autoDetectMapping(headers []string) []ColumnMapping {
	rules := map[string][]string{
		"title":             {"title", "subject", "alert", "name", "case name", "case title"},
		"description":       {"desc", "detail", "summary", "note", "body", "content"},
		"priority":          {"priority", "severity", "urgency", "risk"},
		"tags":              {"tag", "label", "category", "type"},
		"due_date":          {"due", "deadline", "target date", "due date"},
		"assigned_to_email": {"assign", "owner", "analyst", "email"},
	}

	used := map[string]bool{}
	var result []ColumnMapping

	for _, h := range headers {
		lower := strings.ToLower(h)
		for _, field := range SupportedColumns {
			if used[field] {
				continue
			}
			for _, kw := range rules[field] {
				if strings.Contains(lower, kw) {
					result = append(result, ColumnMapping{CSVHeader: h, CaseField: field})
					used[field] = true
					break
				}
			}
		}
	}
	return result
}

// rowToMap converts a CSV record to map[header]value, padding short rows.
func rowToMap(headers []string, record []string) map[string]string {
	m := make(map[string]string, len(headers))
	for i, h := range headers {
		if i < len(record) {
			m[h] = strings.TrimSpace(record[i])
		} else {
			m[h] = ""
		}
	}
	return m
}
