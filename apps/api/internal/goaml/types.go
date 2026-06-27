// Package goaml generates UNODC goAML XML Schema v4.0 reports for FIU submission.
// Used by UAE FIU (goaml.uaefiu.gov.ae), Saudi SAFIU-Connect, and 60+ global FIUs.
// Reference: https://www.unodc.org/unodc/en/money-laundering/goaml/goAML.html
package goaml

import "encoding/xml"

// Report is the root goAML XML document for STR/CTR/SAR submission.
type Report struct {
	XMLName      xml.Name      `xml:"goAMLReport"`
	XMLNs        string        `xml:"xmlns,attr"`
	ReportType   string        `xml:"report_type"`          // "STR", "CTR", "SAR"
	ReportDate   string        `xml:"report_date"`          // YYYY-MM-DD
	Currency     string        `xml:"currency_code_local"`  // "AED", "SAR", "GBP"
	ReportEntity ReportEntity  `xml:"rentity"`
	Transactions []Transaction `xml:"transaction"`
	SubjectNodes []Subject     `xml:"subject"`
}

// ReportEntity identifies the obliged entity filing the report.
type ReportEntity struct {
	EntityID          string `xml:"rentity_id"`
	EntityName        string `xml:"rentity_name"`
	EntityType        string `xml:"rentity_type"` // "bank", "exchange_house", "fintech"
	Phone             string `xml:"phone,omitempty"`
	Address           string `xml:"address,omitempty"`
	ComplianceOfficer string `xml:"compliance_officer,omitempty"`
}

// Transaction represents a financial transaction included in the report.
type Transaction struct {
	TxnNumber       string         `xml:"transactionnumber"`
	TxnDate         string         `xml:"transactiondate"`      // YYYY-MM-DD
	TxnType         string         `xml:"transaction_type"`     // "cash", "wire", "trade"
	AmountLocal     float64        `xml:"amount_local"`
	CurrencyLocal   string         `xml:"currency_code_local"`
	AmountForeign   float64        `xml:"amount_foreign,omitempty"`
	CurrencyForeign string         `xml:"currency_code_foreign,omitempty"`
	FromParty       *InvolvedParty `xml:"involved_party_from,omitempty"`
	ToParty         *InvolvedParty `xml:"involved_party_to,omitempty"`
	BranchCode      string         `xml:"branch_code,omitempty"`
	Narrative       string         `xml:"t_activity_type,omitempty"`
}

// InvolvedParty is the originator or beneficiary of a transaction.
type InvolvedParty struct {
	Role          string `xml:"role"`
	IsSubject     bool   `xml:"is_primary_subject,omitempty"`
	FirstName     string `xml:"first_name,omitempty"`
	MiddleName    string `xml:"middle_name,omitempty"`
	LastName      string `xml:"last_name,omitempty"`
	EntityName    string `xml:"entity_name,omitempty"`
	IDType        string `xml:"id_type,omitempty"`
	IDNumber      string `xml:"id_number,omitempty"`
	Nationality   string `xml:"nationality,omitempty"` // ISO 3166-1 alpha-2
	BirthDate     string `xml:"birth_date,omitempty"`  // YYYY-MM-DD
	AccountNumber string `xml:"account_number,omitempty"`
	BankCode      string `xml:"bank_code,omitempty"`
	CountryCode   string `xml:"country_code,omitempty"` // ISO 3166-1 alpha-2
}

// Subject is the person or entity the report is filed about.
type Subject struct {
	FirstName   string `xml:"first_name,omitempty"`
	LastName    string `xml:"last_name,omitempty"`
	EntityName  string `xml:"entity_name,omitempty"`
	IDType      string `xml:"id_type,omitempty"`
	IDNumber    string `xml:"id_number,omitempty"`
	Nationality string `xml:"nationality,omitempty"`
	BirthDate   string `xml:"birth_date,omitempty"`
	Address     string `xml:"address,omitempty"`
}

const (
	GoAMLNamespace = "http://www.goaml.eg/"

	ReportTypeSTR = "STR" // Suspicious Transaction Report (UAE, India, EU)
	ReportTypeCTR = "CTR" // Cash Transaction Report (UAE ≥ AED 55,000)
	ReportTypeSAR = "SAR" // Suspicious Activity Report (Saudi, UK, US)

	// UAE-specific ID types
	IDTypeEmiratesID    = "emirates_id"
	IDTypePassport      = "passport"
	IDTypeTRN           = "trn"
	IDTypeTradeLicense  = "trade_license"
	IDTypeGCCNationalID = "gcc_national_id"

	// Saudi-specific ID types
	IDTypeSaudiNationalID = "saudi_national_id"
	IDTypeIqama           = "iqama" // Resident identity card
)
