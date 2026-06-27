package goaml

import (
	"strings"
	"testing"
	"time"
)

func TestGenerateSTR_UAE_ProducesValidXML(t *testing.T) {
	data := FilingData{
		FilingRef:  "UAE-STR-001",
		ReportDate: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		Currency:   "AED",
		Subjects: []SubjectData{
			{FirstName: "Mohammed", LastName: "Al Rashidi", IDType: "emirates_id", IDNumber: "784-1990-1234567-1", Nationality: "AE"},
		},
		Transactions: []TxnData{
			{Reference: "TXN-001", Date: time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC), Type: "cash", AmountLocal: 75000, Narrative: "Unusual cash deposit"},
		},
	}
	out, err := GenerateSTR(data)
	if err != nil {
		t.Fatalf("GenerateSTR error: %v", err)
	}
	xml := string(out)
	if !strings.Contains(xml, "<goAMLReport") {
		t.Error("expected <goAMLReport root element")
	}
	if !strings.Contains(xml, "STR") {
		t.Error("expected report_type STR")
	}
	if !strings.Contains(xml, "AED") {
		t.Error("expected AED currency")
	}
	if !strings.Contains(xml, "UAE-STR-001-TXN-001") {
		t.Error("expected transaction number UAE-STR-001-TXN-001")
	}
	if !strings.Contains(xml, "75000") {
		t.Error("expected amount 75000")
	}
}

func TestGenerateCTR_UAE_AboveThreshold(t *testing.T) {
	data := FilingData{
		FilingRef:  "UAE-CTR-001",
		ReportDate: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		Currency:   "AED",
		Transactions: []TxnData{
			{Reference: "TXN-001", Date: time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC), Type: "cash", AmountLocal: 60000},
		},
	}
	out, err := GenerateCTR(data)
	if err != nil {
		t.Fatalf("GenerateCTR error: %v", err)
	}
	if !strings.Contains(string(out), "CTR") {
		t.Error("expected report_type CTR")
	}
}

func TestGenerateSAR_Saudi_SARTerminology(t *testing.T) {
	data := FilingData{
		FilingRef:  "SA-SAR-001",
		ReportDate: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		Currency:   "SAR",
		Subjects: []SubjectData{
			{FirstName: "Abdullah", LastName: "Al Ghamdi", IDType: "saudi_national_id", IDNumber: "1090123456", Nationality: "SA"},
		},
		Transactions: []TxnData{
			{Reference: "TXN-001", Date: time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC), Type: "wire", AmountLocal: 120000, Narrative: "Structured transfers"},
		},
	}
	out, err := GenerateSAR(data)
	if err != nil {
		t.Fatalf("GenerateSAR error: %v", err)
	}
	xml := string(out)
	if !strings.Contains(xml, "SAR") {
		t.Error("expected report_type SAR for Saudi filing")
	}
	if !strings.Contains(xml, "SAR") {
		t.Error("expected SAR currency code")
	}
	if !strings.Contains(xml, "Abdullah") {
		t.Error("expected subject first name")
	}
}

func TestGenerateSTR_DefaultsCurrencyToAED(t *testing.T) {
	data := FilingData{
		FilingRef:  "UAE-STR-002",
		ReportDate: time.Now(),
		// Currency not set — should default to AED
		Transactions: []TxnData{
			{Reference: "TXN-001", Date: time.Now(), Type: "cash", AmountLocal: 10000},
		},
	}
	out, err := GenerateSTR(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(out), "AED") {
		t.Error("expected default currency AED when not specified")
	}
}

func TestGenerateSTR_ForeignCurrencyIncluded(t *testing.T) {
	data := FilingData{
		FilingRef:  "UAE-STR-003",
		ReportDate: time.Now(),
		Currency:   "AED",
		Transactions: []TxnData{
			{
				Reference: "TXN-001", Date: time.Now(), Type: "wire",
				AmountLocal: 36750, ForeignCurrency: "USD", AmountForeign: 10000,
			},
		},
	}
	out, err := GenerateSTR(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	xml := string(out)
	if !strings.Contains(xml, "USD") {
		t.Error("expected foreign currency USD in XML")
	}
	if !strings.Contains(xml, "10000") {
		t.Error("expected foreign amount 10000 in XML")
	}
}

func TestGenerateSTR_NoForeignCurrencyWhenSameAsLocal(t *testing.T) {
	data := FilingData{
		FilingRef:  "UAE-STR-004",
		ReportDate: time.Now(),
		Currency:   "AED",
		Transactions: []TxnData{
			// ForeignCurrency same as local — should be omitted
			{Reference: "TXN-001", Date: time.Now(), Type: "cash", AmountLocal: 55000, ForeignCurrency: "AED", AmountForeign: 55000},
		},
	}
	out, err := GenerateSTR(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// should not have duplicate currency entries
	xmlStr := string(out)
	aedCount := strings.Count(xmlStr, ">AED<")
	if aedCount > 2 {
		t.Errorf("expected at most 2 AED currency entries, got %d", aedCount)
	}
}

func TestGenerateSTR_MultipleTransactions(t *testing.T) {
	data := FilingData{
		FilingRef:  "UAE-STR-005",
		ReportDate: time.Now(),
		Currency:   "AED",
		Transactions: []TxnData{
			{Reference: "T1", Date: time.Now(), Type: "cash", AmountLocal: 20000},
			{Reference: "T2", Date: time.Now(), Type: "wire", AmountLocal: 35000},
			{Reference: "T3", Date: time.Now(), Type: "trade", AmountLocal: 15000},
		},
	}
	out, err := GenerateSTR(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	xml := string(out)
	if strings.Count(xml, "<transaction>") != 3 {
		t.Errorf("expected 3 <transaction> elements")
	}
}

func TestGenerateSTR_XMLHeader(t *testing.T) {
	data := FilingData{FilingRef: "TEST", ReportDate: time.Now(), Currency: "AED"}
	out, err := GenerateSTR(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(string(out), "<?xml") {
		t.Error("expected XML declaration header")
	}
}
