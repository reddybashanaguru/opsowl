package str

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"
	"time"
)

// goAML 4.0 XML structures — UNODC global standard.
// Schema reference: UNODC goAML 4.0 Technical Specification.
// Used by 60+ FIUs worldwide (FIU-IND, UAEFIU, FIU Kenya, FIU Mauritius, etc.).
// Element names match the UNODC goAML 4.0 schema exactly.

// GoAMLReport is the root report element.
type GoAMLReport struct {
	XMLName xml.Name `xml:"report"`

	RentityID       string `xml:"rentity_id"`
	RentityBranch   string `xml:"rentity_branch,omitempty"`
	SubmissionCode  string `xml:"submission_code"`  // E=new, S=supplementary, C=correction
	ReportCode      string `xml:"report_code"`      // STR, CTR, SAR
	EntityReference string `xml:"entity_reference"` // our internal reference
	FIURefNumber    string `xml:"fiu_ref_number,omitempty"`
	SubmissionDate  string `xml:"submission_date"`
	CurrencyLocal   string `xml:"currency_code_local"` // ISO 4217, e.g. INR, AED, KES

	ReportingPerson goAMLPerson    `xml:"reporting_person"`
	Location        goAMLLocation  `xml:"location"`
	Reason          string         `xml:"reason"`
	Action          string         `xml:"action,omitempty"`
	Indicators      goAMLIndicators `xml:"report_indicators"`
	Transactions    []goAMLTransaction `xml:"transaction"`
}

type goAMLPerson struct {
	Gender      string `xml:"gender,omitempty"`      // M, F, U
	Title       string `xml:"title,omitempty"`       // Mr, Ms, Dr
	FirstName   string `xml:"first_name"`
	LastName    string `xml:"last_name"`
	Birthdate   string `xml:"birthdate,omitempty"`
	IDNumber    string `xml:"id_number"`
	IDType      string `xml:"id_type"`               // PASSPORT, NATIONAL_ID, TAX_ID, DRIVING_LICENSE, PAN, etc.
	Nationality string `xml:"nationality1,omitempty"` // ISO 3166-1 alpha-2 country code
	Email       string `xml:"email,omitempty"`
	Phone       string `xml:"phone,omitempty"`
	Occupation  string `xml:"occupation,omitempty"`
}

type goAMLLocation struct {
	AddressType string `xml:"address_type,omitempty"` // H=HQ, B=Branch
	Address     string `xml:"address,omitempty"`
	City        string `xml:"city,omitempty"`
	CountryCode string `xml:"country_code,omitempty"` // ISO 3166-1 alpha-2
	State       string `xml:"state,omitempty"`
	Zip         string `xml:"zip,omitempty"`
}

type goAMLIndicators struct {
	Indicators []string `xml:"indicator"`
}

type goAMLTransaction struct {
	Seq                 int    `xml:"seq,attr,omitempty"`
	TransactionNumber   string `xml:"transactionnumber,omitempty"`
	TransactionLocation string `xml:"transaction_location,omitempty"`
	TransactionDesc     string `xml:"transaction_description,omitempty"`
	DateTransaction     string `xml:"date_transaction"`
	Teller              string `xml:"teller,omitempty"`
	Authorized          string `xml:"authorized,omitempty"`
	TransmodeCode       string `xml:"transmode_code"`         // C=Cash, T=Transfer, W=Wire, D=Deposit, X=Other
	TransmodeComment    string `xml:"transmode_comment,omitempty"`
	AmountLocal         string `xml:"amount_local"`           // formatted as 2dp string

	FromMyClient *goAMLFromMyClient `xml:"t_from_my_client,omitempty"`
	ToMyClient   *goAMLToMyClient   `xml:"t_to_my_client,omitempty"`
}

type goAMLFromMyClient struct {
	FromPerson  *goAMLFromPerson  `xml:"from_person,omitempty"`
	FromAccount *goAMLFromAccount `xml:"from_account,omitempty"`
}

type goAMLToMyClient struct {
	ToAccount *goAMLToAccount `xml:"to_account,omitempty"`
}

type goAMLFromPerson struct {
	Gender      string `xml:"gender,omitempty"`
	Title       string `xml:"title,omitempty"`
	FirstName   string `xml:"first_name"`
	LastName    string `xml:"last_name"`
	Birthdate   string `xml:"birthdate,omitempty"`
	IDNumber    string `xml:"id_number,omitempty"`
	IDType      string `xml:"id_type,omitempty"`
	Nationality string `xml:"nationality1,omitempty"`
}

type goAMLFromAccount struct {
	InstitutionName string  `xml:"institution_name,omitempty"`
	SWIFT           string  `xml:"swift,omitempty"`
	Account         string  `xml:"account,omitempty"`
	CurrencyCode    string  `xml:"currency_code,omitempty"`
	AccountName     string  `xml:"account_name,omitempty"`
	ClientNumber    string  `xml:"client_number,omitempty"`
	Opened          string  `xml:"opened,omitempty"`
	Balance         *string `xml:"balance,omitempty"`
	StatusCode      string  `xml:"status_code,omitempty"` // A=active
}

type goAMLToAccount struct {
	InstitutionName string `xml:"institution_name,omitempty"`
	SWIFT           string `xml:"swift,omitempty"`
	Account         string `xml:"account,omitempty"`
	CurrencyCode    string `xml:"currency_code,omitempty"`
	AccountName     string `xml:"account_name,omitempty"`
}

// ── Extended JSON structs (superset of finnet.go structs) ────────────────────
// These decode the JSONB columns and carry all fields for both FINnet and goAML.

type goAMLEntityJSON struct {
	// FINnet-compatible fields
	Name             string `json:"name"`
	EntityType       string `json:"entity_type"`
	LEICode          string `json:"lei_code"`
	BranchCode       string `json:"branch_code"`
	ReportingOfficer string `json:"reporting_officer"`
	ContactEmail     string `json:"contact_email"`

	// goAML-specific
	RentityID      string `json:"rentity_id"`     // FIU registration number — REQUIRED (format varies by country)
	RentityBranch  string `json:"rentity_branch"` // branch identifier
	SWIFTCode      string `json:"swift_code"`
	SubmissionCode string `json:"submission_code"` // E=new, S=supplementary, C=correction

	// Jurisdiction / currency — configurable per client country
	FIUName       string `json:"fiu_name"`       // e.g. FIU-IND, UAEFIU, FIU Kenya (used in action text)
	LocalCurrency string `json:"local_currency"` // ISO 4217, e.g. INR, AED, KES, USD (overrides country default)

	// Reporting officer details
	OfficerTitle      string `json:"officer_title"`
	OfficerFirstName  string `json:"officer_first_name"` // REQUIRED
	OfficerLastName   string `json:"officer_last_name"`  // REQUIRED
	OfficerGender     string `json:"officer_gender"`
	OfficerBirthdate  string `json:"officer_birthdate"`
	OfficerIDNumber   string `json:"officer_id_number"` // national ID / tax ID of officer — REQUIRED
	OfficerIDType     string `json:"officer_id_type"`   // PASSPORT, NATIONAL_ID, TAX_ID, DRIVING_LICENSE, PAN, etc.
	OfficerPhone      string `json:"officer_phone"`
	OfficerEmail      string `json:"officer_email"`
	OfficerOccupation string `json:"officer_occupation"`

	// Entity address
	EntityAddress string `json:"entity_address"`
	EntityCity    string `json:"entity_city"`
	EntityState   string `json:"entity_state"`
	EntityZip     string `json:"entity_zip"`
	EntityCountry string `json:"entity_country"` // ISO 3166-1 alpha-2, e.g. IN, AE, KE
}

type goAMLSubjectJSON struct {
	// FINnet-compatible (kept for backward compat with existing data)
	Name         string `json:"name"`
	Relationship string `json:"relationship"`
	AccountRef   string `json:"account_ref"`
	PAN          string `json:"pan"`         // India-specific; maps to id_number when IDType not set
	AadhaarRef   string `json:"aadhaar_ref"` // India-specific auxiliary field
	RiskRating   string `json:"risk_rating"`

	// goAML-specific — split name
	FirstName   string `json:"first_name"` // REQUIRED (or fallback to Name split)
	LastName    string `json:"last_name"`
	Gender      string `json:"gender"`
	Title       string `json:"title"`
	Birthdate   string `json:"birthdate"`
	IDType      string `json:"id_type"`   // PASSPORT, NATIONAL_ID, TAX_ID, DRIVING_LICENSE, PAN, etc.
	IDNumber    string `json:"id_number"` // the identifier value — REQUIRED
	Nationality string `json:"nationality"` // ISO 3166-1 alpha-2

	// Account details
	AccountNumber  string  `json:"account_number"`
	AccountName    string  `json:"account_name"`
	ClientNumber   string  `json:"client_number"`
	AccountOpened  string  `json:"account_opened"`
	AccountBalance float64 `json:"account_balance"`
	AccountStatus  string  `json:"account_status"` // A=active
}

type goAMLTxJSON struct {
	// FINnet-compatible
	Date        string  `json:"date"`
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Reference   string  `json:"reference"`

	// goAML-specific
	TransactionNumber   string  `json:"transaction_number"`
	TransactionLocation string  `json:"transaction_location"`
	Teller              string  `json:"teller"`
	Authorized          string  `json:"authorized"`
	TransmodeCode       string  `json:"transmode_code"` // C, T, W, D, X
	TransmodeComment    string  `json:"transmode_comment"`

	// From account
	FromAccountNumber string  `json:"from_account_number"`
	FromAccountName   string  `json:"from_account_name"`
	FromClientNumber  string  `json:"from_client_number"`
	FromInstitution   string  `json:"from_institution"`
	FromSWIFT         string  `json:"from_swift"`
	FromOpened        string  `json:"from_opened"`
	FromBalance       float64 `json:"from_balance"`

	// To account
	ToAccountNumber string `json:"to_account_number"`
	ToAccountName   string `json:"to_account_name"`
	ToInstitution   string `json:"to_institution"`
	ToSWIFT         string `json:"to_swift"`
}

// ── Validation ────────────────────────────────────────────────────────────────

// GoAMLValidationError describes a single missing/invalid field.
type GoAMLValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidateGoAMLReadiness returns all validation errors that would prevent
// a valid goAML 4.0 XML from being generated. Empty slice = ready to export.
func ValidateGoAMLReadiness(f *FilingResponse) []GoAMLValidationError {
	errs := make([]GoAMLValidationError, 0)

	var entity goAMLEntityJSON
	_ = json.Unmarshal(f.FilingEntity, &entity)

	var subj goAMLSubjectJSON
	_ = json.Unmarshal(f.Subject, &subj)

	var txs []goAMLTxJSON
	_ = json.Unmarshal(f.Transactions, &txs)

	// Entity / reporting officer — required
	if entity.RentityID == "" {
		errs = append(errs, GoAMLValidationError{"filing_entity.rentity_id", "Reporting entity FIU registration ID is required"})
	}
	if entity.OfficerFirstName == "" {
		errs = append(errs, GoAMLValidationError{"filing_entity.officer_first_name", "Reporting officer first name is required"})
	}
	if entity.OfficerLastName == "" {
		errs = append(errs, GoAMLValidationError{"filing_entity.officer_last_name", "Reporting officer last name is required"})
	}
	if entity.OfficerIDNumber == "" {
		errs = append(errs, GoAMLValidationError{"filing_entity.officer_id_number", "Reporting officer national ID / tax ID number is required"})
	}
	if entity.OfficerIDType == "" {
		errs = append(errs, GoAMLValidationError{"filing_entity.officer_id_type", "Reporting officer ID type is required (e.g. PASSPORT, NATIONAL_ID, TAX_ID)"})
	}

	// Subject — required
	subjName := subj.FirstName
	if subjName == "" {
		subjName = subj.Name
	}
	if subjName == "" {
		errs = append(errs, GoAMLValidationError{"subject.first_name", "Subject first name (or full name) is required"})
	}
	subjID := subj.IDNumber
	if subjID == "" {
		subjID = subj.PAN // backward compat: PAN maps to id_number for India clients
	}
	if subjID == "" {
		errs = append(errs, GoAMLValidationError{"subject.id_number", "Subject identifier (national ID, passport, tax ID, etc.) is required"})
	}

	// Narrative
	if strings.TrimSpace(f.Narrative) == "" {
		errs = append(errs, GoAMLValidationError{"narrative", "Narrative (reason) is required for goAML submission"})
	}

	// Transactions
	if len(txs) == 0 {
		errs = append(errs, GoAMLValidationError{"transactions", "At least one transaction is required"})
	}
	for i, tx := range txs {
		prefix := fmt.Sprintf("transactions[%d]", i)
		if tx.Date == "" {
			errs = append(errs, GoAMLValidationError{prefix + ".date", "Transaction date is required"})
		}
		if tx.Amount <= 0 {
			errs = append(errs, GoAMLValidationError{prefix + ".amount", "Transaction amount must be greater than zero"})
		}
		if tx.TransmodeCode == "" {
			errs = append(errs, GoAMLValidationError{prefix + ".transmode_code", "Transaction mode code is required (C/T/W/D/X)"})
		}
	}

	return errs
}

// ── Country → currency lookup ─────────────────────────────────────────────────
// ISO 4217 defaults by country code for goAML currency_code_local.
// Covers all known goAML-using jurisdictions. Falls back to "USD" if unknown.
var countryCurrency = map[string]string{
	"IN": "INR", "AE": "AED", "BH": "BHD", "JO": "JOD",
	"KW": "KWD", "OM": "OMR", "QA": "QAR", "SA": "SAR",
	"KE": "KES", "RW": "RWF", "TZ": "TZS", "UG": "UGX",
	"MU": "MUR", "GH": "GHS", "NG": "NGN", "ZA": "ZAR",
	"MA": "MAD", "TN": "TND", "EG": "EGP",
	"PK": "PKR", "LK": "LKR", "BD": "BDT", "MM": "MMK",
	"KH": "KHR", "PH": "PHP", "TH": "THB", "VN": "VND",
	"ID": "IDR", "MY": "MYR", "SG": "SGD",
	"US": "USD", "GB": "GBP", "EU": "EUR", "AU": "AUD",
}

func localCurrency(entity goAMLEntityJSON) string {
	if entity.LocalCurrency != "" {
		return entity.LocalCurrency
	}
	if c, ok := countryCurrency[strings.ToUpper(entity.EntityCountry)]; ok {
		return c
	}
	return "USD" // international fallback
}

func fiuActionText(entity goAMLEntityJSON) string {
	if entity.FIUName != "" {
		return "Filed with " + entity.FIUName + " pursuant to applicable AML/CFT reporting obligations"
	}
	return "Filed with Financial Intelligence Unit pursuant to applicable AML/CFT reporting obligations"
}

// ── Builder ───────────────────────────────────────────────────────────────────

// BuildGoAMLXML produces a valid goAML 4.0 XML document for submission to any
// UNODC-compatible FIU. Returns a validation error list if the filing is
// incomplete; XML is only returned when errs is empty.
func BuildGoAMLXML(f *FilingResponse) ([]byte, []GoAMLValidationError) {
	errs := ValidateGoAMLReadiness(f)
	if len(errs) > 0 {
		return nil, errs
	}

	var entity goAMLEntityJSON
	_ = json.Unmarshal(f.FilingEntity, &entity)

	var subj goAMLSubjectJSON
	_ = json.Unmarshal(f.Subject, &subj)

	var txs []goAMLTxJSON
	_ = json.Unmarshal(f.Transactions, &txs)

	submissionCode := entity.SubmissionCode
	if submissionCode == "" {
		submissionCode = "E"
	}

	officerIDType := entity.OfficerIDType
	if officerIDType == "" {
		officerIDType = "NATIONAL_ID" // generic fallback; FIU-specific forms override this
	}

	// Resolve subject name fields
	subjFirstName := subj.FirstName
	subjLastName := subj.LastName
	if subjFirstName == "" {
		// split on first space
		parts := strings.SplitN(subj.Name, " ", 2)
		subjFirstName = parts[0]
		if len(parts) > 1 {
			subjLastName = parts[1]
		}
	}

	// Resolve subject ID
	subjIDNumber := subj.IDNumber
	subjIDType := subj.IDType
	if subjIDNumber == "" && subj.PAN != "" {
		// backward compat: Indian clients stored PAN separately
		subjIDNumber = subj.PAN
		subjIDType = "PAN"
	}
	if subjIDType == "" {
		subjIDType = "NATIONAL_ID"
	}

	// Auto-detect indicators from suspicion basis text, then merge any manually
	// selected EU FATF typology codes (officer may add/override).
	indicators := mapGoAMLIndicators(f.SuspicionBasis)
	if len(f.EUFATFIndicators) > 0 {
		seen := map[string]bool{}
		for _, c := range indicators {
			seen[c] = true
		}
		for _, c := range f.EUFATFIndicators {
			if !seen[c] {
				indicators = append(indicators, c)
				seen[c] = true
			}
		}
	}

	// Resolve local currency once; applies to all transactions unless overridden
	entityCurrency := localCurrency(entity)

	// Build transactions
	goTxs := make([]goAMLTransaction, 0, len(txs))
	for i, tx := range txs {
		cur := tx.Currency
		if cur == "" {
			cur = entityCurrency
		}
		transmode := tx.TransmodeCode
		transmodeComment := tx.TransmodeComment
		if transmodeComment == "" {
			transmodeComment = transmodeLabel(transmode)
		}

		txNum := tx.TransactionNumber
		if txNum == "" {
			txNum = fmt.Sprintf("TXN-%03d", i+1)
		}

		// Build from_my_client block
		fromPerson := &goAMLFromPerson{
			Gender:      subj.Gender,
			Title:       subj.Title,
			FirstName:   subjFirstName,
			LastName:    subjLastName,
			Birthdate:   subj.Birthdate,
			IDNumber:    subjIDNumber,
			IDType:      subjIDType,
			Nationality: defaultStr(subj.Nationality, entity.EntityCountry),
		}

		fromAccount := buildFromAccount(tx, entity, subj, cur)

		goTx := goAMLTransaction{
			Seq:                 i + 1,
			TransactionNumber:   txNum,
			TransactionLocation: tx.TransactionLocation,
			TransactionDesc:     tx.Description,
			DateTransaction:     tx.Date,
			Teller:              tx.Teller,
			Authorized:          tx.Authorized,
			TransmodeCode:       transmode,
			TransmodeComment:    transmodeComment,
			AmountLocal:         fmt.Sprintf("%.2f", tx.Amount),
			FromMyClient: &goAMLFromMyClient{
				FromPerson:  fromPerson,
				FromAccount: fromAccount,
			},
		}

		// Add to_account if we have destination details
		if tx.ToAccountNumber != "" || tx.ToInstitution != "" {
			goTx.ToMyClient = &goAMLToMyClient{
				ToAccount: &goAMLToAccount{
					InstitutionName: tx.ToInstitution,
					SWIFT:           tx.ToSWIFT,
					Account:         tx.ToAccountNumber,
					CurrencyCode:    cur,
					AccountName:     tx.ToAccountName,
				},
			}
		}

		goTxs = append(goTxs, goTx)
	}

	report := GoAMLReport{
		RentityID:       entity.RentityID,
		RentityBranch:   defaultStr(entity.RentityBranch, entity.BranchCode),
		SubmissionCode:  submissionCode,
		ReportCode:      f.ReportType,
		EntityReference: fmt.Sprintf("OWL-%s", f.ID.String()[:8]),
		FIURefNumber:    f.FIUReference,
		SubmissionDate:  time.Now().UTC().Format("2006-01-02"),
		CurrencyLocal:   entityCurrency,

		ReportingPerson: goAMLPerson{
			Gender:      entity.OfficerGender,
			Title:       entity.OfficerTitle,
			FirstName:   entity.OfficerFirstName,
			LastName:    entity.OfficerLastName,
			Birthdate:   entity.OfficerBirthdate,
			IDNumber:    entity.OfficerIDNumber,
			IDType:      officerIDType,
			Nationality: entity.EntityCountry, // officer assumed same country as entity
			Email:       defaultStr(entity.OfficerEmail, entity.ContactEmail),
			Phone:       entity.OfficerPhone,
			Occupation:  defaultStr(entity.OfficerOccupation, "Compliance Officer"),
		},

		Location: goAMLLocation{
			AddressType: "H",
			Address:     entity.EntityAddress,
			City:        entity.EntityCity,
			CountryCode: entity.EntityCountry,
			State:       entity.EntityState,
			Zip:         entity.EntityZip,
		},

		Reason:       f.Narrative,
		Action:       fiuActionText(entity),
		Indicators:   goAMLIndicators{Indicators: indicators},
		Transactions: goTxs,
	}

	out, err := xml.MarshalIndent(report, "", "  ")
	if err != nil {
		return nil, []GoAMLValidationError{{"xml", "failed to marshal XML: " + err.Error()}}
	}
	return append([]byte(xml.Header), out...), nil
}

func buildFromAccount(tx goAMLTxJSON, entity goAMLEntityJSON, subj goAMLSubjectJSON, cur string) *goAMLFromAccount {
	acct := defaultStr(tx.FromAccountNumber, subj.AccountNumber, subj.AccountRef)
	if acct == "" && tx.FromInstitution == "" {
		return nil
	}
	inst := defaultStr(tx.FromInstitution, entity.Name)
	sw := defaultStr(tx.FromSWIFT, entity.SWIFTCode)
	name := defaultStr(tx.FromAccountName, subj.AccountName, subj.Name)
	clientNum := defaultStr(tx.FromClientNumber, subj.ClientNumber)
	opened := defaultStr(tx.FromOpened, subj.AccountOpened)
	status := defaultStr(subj.AccountStatus, "A")

	a := &goAMLFromAccount{
		InstitutionName: inst,
		SWIFT:           sw,
		Account:         acct,
		CurrencyCode:    cur,
		AccountName:     name,
		ClientNumber:    clientNum,
		Opened:          opened,
		StatusCode:      status,
	}
	if tx.FromBalance > 0 {
		s := fmt.Sprintf("%.2f", tx.FromBalance)
		a.Balance = &s
	} else if subj.AccountBalance > 0 {
		s := fmt.Sprintf("%.2f", subj.AccountBalance)
		a.Balance = &s
	}
	return a
}

func mapGoAMLIndicators(basis string) []string {
	lower := strings.ToLower(basis)
	codes := []struct{ keyword, code string }{
		{"cash", "ML01"},
		{"structured", "ML02"},
		{"unusual", "ML03"},
		{"round", "ML04"},
		{"fraud", "FRD01"},
		{"terror", "TF01"},
		{"sanctions", "PF01"},
		{"loan", "FRD02"},
		{"kyc", "ML05"},
	}
	seen := map[string]bool{}
	var out []string
	for _, c := range codes {
		if strings.Contains(lower, c.keyword) && !seen[c.code] {
			out = append(out, c.code)
			seen[c.code] = true
		}
	}
	if len(out) == 0 {
		out = []string{"ML03"} // default: unusual transaction
	}
	return out
}

func transmodeLabel(code string) string {
	switch code {
	case "C":
		return "Cash"
	case "T":
		return "Transfer"
	case "W":
		return "Wire Transfer"
	case "D":
		return "Deposit"
	case "X":
		return "Other"
	default:
		return code
	}
}

// defaultStr returns the first non-empty string from the provided values.
func defaultStr(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
