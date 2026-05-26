package str

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"
	"time"
)

// FINnet 2.0 XML structures (FIU-IND submission format, PMLA 2002).
// Field names follow the FINnet 2.0 schema published by FIU-IND.

type FINnetReport struct {
	XMLName     xml.Name        `xml:"FINnet2Report"`
	XMLNS       string          `xml:"xmlns,attr"`
	ReportType  string          `xml:"ReportType,attr"`
	ReportRefNo string          `xml:"ReportRefNo,attr"`
	FilingDate  string          `xml:"FilingDate,attr"`

	FilingEntity finnetEntity       `xml:"FilingEntity"`
	Subject      finnetSubject      `xml:"Subject"`
	Transactions finnetTransactions `xml:"Transactions"`
	Activity     finnetActivity     `xml:"SuspiciousActivity"`
	Certification finnetCertification `xml:"Certification"`
}

type finnetEntity struct {
	EntityName       string `xml:"EntityName"`
	EntityType       string `xml:"EntityType"`
	LEICode          string `xml:"LEICode,omitempty"`
	BranchCode       string `xml:"BranchCode,omitempty"`
	ReportingOfficer string `xml:"ReportingOfficer"`
	ContactEmail     string `xml:"ContactEmail,omitempty"`
}

type finnetSubject struct {
	SubjectType  string           `xml:"SubjectType"`
	Name         string           `xml:"Name"`
	Relationship string           `xml:"Relationship,omitempty"`
	Identifiers  finnetIdentifiers `xml:"Identifiers"`
	AccountRef   string           `xml:"AccountRef,omitempty"`
	RiskRating   string           `xml:"RiskRating,omitempty"`
}

type finnetIdentifiers struct {
	PAN         string `xml:"PAN,omitempty"`
	AadhaarHash string `xml:"AadhaarHash,omitempty"` // hashed, never raw Aadhaar
}

type finnetTransactions struct {
	Items []finnetTx `xml:"Transaction"`
}

type finnetTx struct {
	Date        string  `xml:"Date"`
	Amount      float64 `xml:"Amount"`
	Currency    string  `xml:"Currency"`
	Type        string  `xml:"Type"`
	Description string  `xml:"Description"`
	Reference   string  `xml:"Reference,omitempty"`
}

type finnetActivity struct {
	SuspicionType   string `xml:"SuspicionType"`  // FIN_01, FIN_02, etc.
	Narrative       string `xml:"Narrative"`
	EvidenceSummary string `xml:"EvidenceSummary"`
}

type finnetCertification struct {
	SubmittedBy string `xml:"SubmittedBy"`
	ApprovedBy  string `xml:"ApprovedBy"`
	SubmittedAt string `xml:"SubmittedAt"`
	ApprovedAt  string `xml:"ApprovedAt"`
}

// subjectJSON is a minimal superset of the subject JSONB payload.
type subjectJSON struct {
	Name         string `json:"name"`
	Relationship string `json:"relationship"`
	AccountRef   string `json:"account_ref"`
	PAN          string `json:"pan"`
	AadhaarRef   string `json:"aadhaar_ref"`
	RiskRating   string `json:"risk_rating"`
}

type entityJSON struct {
	Name             string `json:"name"`
	EntityType       string `json:"entity_type"`
	LEICode          string `json:"lei_code"`
	BranchCode       string `json:"branch_code"`
	ReportingOfficer string `json:"reporting_officer"`
	ContactEmail     string `json:"contact_email"`
}

type txJSON struct {
	Date        string  `json:"date"`
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Reference   string  `json:"reference"`
}

// BuildFINnetXML produces FINnet 2.0 compliant XML from a filing.
func BuildFINnetXML(f *FilingResponse) ([]byte, error) {
	var subj subjectJSON
	_ = json.Unmarshal(f.Subject, &subj)

	var entity entityJSON
	_ = json.Unmarshal(f.FilingEntity, &entity)

	var txs []txJSON
	_ = json.Unmarshal(f.Transactions, &txs)

	items := make([]finnetTx, 0, len(txs))
	for _, t := range txs {
		cur := t.Currency
		if cur == "" {
			cur = "INR"
		}
		items = append(items, finnetTx{
			Date:        t.Date,
			Amount:      t.Amount,
			Currency:    cur,
			Type:        t.Type,
			Description: t.Description,
			Reference:   t.Reference,
		})
	}

	submittedAt := ""
	if f.SubmittedAt != nil {
		submittedAt = f.SubmittedAt.UTC().Format(time.RFC3339)
	}
	approvedAt := ""
	if f.ApprovedAt != nil {
		approvedAt = f.ApprovedAt.UTC().Format(time.RFC3339)
	}

	// Map suspicion basis string to FIN_xx code (best-effort).
	suspicionCode := mapSuspicionCode(f.SuspicionBasis)

	entityType := entity.EntityType
	if entityType == "" {
		entityType = "Banking Company"
	}

	report := FINnetReport{
		XMLNS:       "urn:fiu-ind:finnet2:report:v2.0",
		ReportType:  f.ReportType,
		ReportRefNo: fmt.Sprintf("OWL-%s", f.ID.String()[:8]),
		FilingDate:  time.Now().UTC().Format("2006-01-02"),

		FilingEntity: finnetEntity{
			EntityName:       entity.Name,
			EntityType:       entityType,
			LEICode:          entity.LEICode,
			BranchCode:       entity.BranchCode,
			ReportingOfficer: entity.ReportingOfficer,
			ContactEmail:     entity.ContactEmail,
		},
		Subject: finnetSubject{
			SubjectType:  "Individual",
			Name:         subj.Name,
			Relationship: subj.Relationship,
			Identifiers: finnetIdentifiers{
				PAN: subj.PAN,
				// Aadhaar is never stored raw — store hash only
				AadhaarHash: hashIfPresent(subj.AadhaarRef),
			},
			AccountRef: subj.AccountRef,
			RiskRating: subj.RiskRating,
		},
		Transactions: finnetTransactions{Items: items},
		Activity: finnetActivity{
			SuspicionType:   suspicionCode,
			Narrative:       f.Narrative,
			EvidenceSummary: f.EvidenceSummary,
		},
		Certification: finnetCertification{
			SubmittedBy: f.SubmittedByName,
			ApprovedBy:  f.ApprovedByName,
			SubmittedAt: submittedAt,
			ApprovedAt:  approvedAt,
		},
	}

	out, err := xml.MarshalIndent(report, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), out...), nil
}

func mapSuspicionCode(basis string) string {
	lower := strings.ToLower(basis)
	codes := []struct{ keyword, code string }{
		{"cash", "FIN_01"},
		{"structured", "FIN_02"},
		{"unusual", "FIN_03"},
		{"round", "FIN_04"},
		{"fraud", "FIN_05"},
		{"terror", "FIN_06"},
		{"sanctions", "FIN_07"},
	}
	for _, c := range codes {
		if strings.Contains(lower, c.keyword) {
			return c.code
		}
	}
	return "FIN_99"
}

func hashIfPresent(s string) string {
	if s == "" {
		return ""
	}
	// Store only the first/last 4 digits as a masked reference — never raw Aadhaar.
	// Full Aadhaar must never be persisted (DPDP Act 2023 s.4).
	if len(s) >= 8 {
		return s[:4] + "XXXX" + s[len(s)-4:]
	}
	return "XXXX"
}
