package goaml

import (
	"encoding/xml"
	"fmt"
	"os"
	"time"
)

// FilingData is the caller-supplied input for XML generation.
// Map from OpsOwl STRFiling / Case / Transaction domain models to this struct.
type FilingData struct {
	FilingRef    string
	ReportDate   time.Time
	Currency     string // "AED" for UAE, "SAR" for Saudi — defaults to "AED"
	Transactions []TxnData
	Subjects     []SubjectData
}

type TxnData struct {
	Reference       string
	Date            time.Time
	Type            string  // "cash", "wire", "trade"
	AmountLocal     float64
	AmountForeign   float64
	ForeignCurrency string
	FromName        string
	FromIDType      string
	FromIDNumber    string
	FromAccount     string
	ToName          string
	ToIDType        string
	ToIDNumber      string
	ToAccount       string
	Narrative       string
}

type SubjectData struct {
	FirstName   string
	LastName    string
	EntityName  string
	IDType      string
	IDNumber    string
	Nationality string
	BirthDate   string
	Address     string
}

// GenerateSTR builds a goAML XML Suspicious Transaction Report.
// Used by UAE FIU (STR), Saudi SAFIU-Connect (SAR mapped to SAR report_type),
// and any UNODC-compatible FIU.
func GenerateSTR(data FilingData) ([]byte, error) {
	return marshalReport(buildReport(ReportTypeSTR, data))
}

// GenerateCTR builds a goAML XML Cash Transaction Report.
// Mandatory for UAE cash transactions ≥ AED 55,000 (CBUAE requirement).
// Saudi equivalent threshold: SAR 60,000.
func GenerateCTR(data FilingData) ([]byte, error) {
	return marshalReport(buildReport(ReportTypeCTR, data))
}

// GenerateSAR builds a goAML XML Suspicious Activity Report.
// Used for Saudi SAFIU-Connect (SAMA mandates SAR terminology).
func GenerateSAR(data FilingData) ([]byte, error) {
	return marshalReport(buildReport(ReportTypeSAR, data))
}

func buildReport(reportType string, data FilingData) Report {
	currency := data.Currency
	if currency == "" {
		currency = "AED"
	}

	r := Report{
		XMLNs:      GoAMLNamespace,
		ReportType: reportType,
		ReportDate: data.ReportDate.Format("2006-01-02"),
		Currency:   currency,
		ReportEntity: ReportEntity{
			EntityID:          os.Getenv("GOAML_ENTITY_ID"),
			EntityName:        os.Getenv("GOAML_ENTITY_NAME"),
			EntityType:        os.Getenv("GOAML_ENTITY_TYPE"),
			Phone:             os.Getenv("GOAML_ENTITY_PHONE"),
			Address:           os.Getenv("GOAML_ENTITY_ADDRESS"),
			ComplianceOfficer: os.Getenv("GOAML_COMPLIANCE_OFFICER"),
		},
	}

	for i, t := range data.Transactions {
		txn := Transaction{
			TxnNumber:     fmt.Sprintf("%s-TXN-%03d", data.FilingRef, i+1),
			TxnDate:       t.Date.Format("2006-01-02"),
			TxnType:       t.Type,
			AmountLocal:   t.AmountLocal,
			CurrencyLocal: currency,
			Narrative:     t.Narrative,
		}
		if t.AmountForeign > 0 && t.ForeignCurrency != "" && t.ForeignCurrency != currency {
			txn.AmountForeign = t.AmountForeign
			txn.CurrencyForeign = t.ForeignCurrency
		}
		if t.FromName != "" || t.FromAccount != "" {
			txn.FromParty = &InvolvedParty{
				Role:          "originator",
				EntityName:    t.FromName,
				IDType:        t.FromIDType,
				IDNumber:      t.FromIDNumber,
				AccountNumber: t.FromAccount,
			}
		}
		if t.ToName != "" || t.ToAccount != "" {
			txn.ToParty = &InvolvedParty{
				Role:          "beneficiary",
				EntityName:    t.ToName,
				IDType:        t.ToIDType,
				IDNumber:      t.ToIDNumber,
				AccountNumber: t.ToAccount,
			}
		}
		r.Transactions = append(r.Transactions, txn)
	}

	for _, s := range data.Subjects {
		r.SubjectNodes = append(r.SubjectNodes, Subject{
			FirstName:   s.FirstName,
			LastName:    s.LastName,
			EntityName:  s.EntityName,
			IDType:      s.IDType,
			IDNumber:    s.IDNumber,
			Nationality: s.Nationality,
			BirthDate:   s.BirthDate,
			Address:     s.Address,
		})
	}

	return r
}

func marshalReport(r Report) ([]byte, error) {
	out, err := xml.MarshalIndent(r, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("goaml: marshal failed: %w", err)
	}
	return append([]byte(xml.Header), out...), nil
}
