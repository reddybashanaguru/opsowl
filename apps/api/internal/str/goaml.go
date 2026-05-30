package str

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"
	"time"
)

// goAML 4.0 XML structures — FIU-IND India submission format.
// Schema reference: UNODC goAML 4.0 / FIU-IND Technical Specification v4.0 (2023).
// All element names exactly match the FIU-IND goAML portal schema.

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
	CurrencyLocal   string `xml:"currency_code_local"` // INR

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
	IDType      string `xml:"id_type"`               // PAN, PASSPORT, DRIVING_LICENSE
	Nationality string `xml:"nationality1,omitempty"` // IN
	Email       string `xml:"email,omitempty"`
	Phone       string `xml:"phone,omitempty"`
	Occupation  string `xml:"occupation,omitempty"`
}

type goAMLLocation struct {
	AddressType string `xml:"address_type,omitempty"` // H=HQ, B=Branch
	Address     string `xml:"address,omitempty"`
	City        string `xml:"city,omitempty"`
	CountryCode string `xml:"country_code,omitempty"` // IN
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
	RentityID      string `json:"rentity_id"`     // FIU-IND registered entity ID — REQUIRED
	RentityBranch  string `json:"rentity_branch"` // branch identifier
	SWIFTCode      string `json:"swift_code"`
	SubmissionCode string `json:"submission_code"` // E/S/C — default E

	// Reporting officer details
	OfficerTitle      string `json:"officer_title"`
	OfficerFirstName  string `json:"officer_first_name"` // REQUIRED
	OfficerLastName   string `json:"officer_last_name"`  // REQUIRED
	OfficerGender     string `json:"officer_gender"`
	OfficerBirthdate  string `json:"officer_birthdate"`
	OfficerIDNumber   string `json:"officer_id_number"` // PAN of officer — REQUIRED
	OfficerIDType     string `json:"officer_id_type"`   // PAN / DRIVING_LICENSE / PASSPORT
	OfficerPhone      string `json:"officer_phone"`
	OfficerEmail      string `json:"officer_email"`
	OfficerOccupation string `json:"officer_occupation"`

	// Entity address
	EntityAddress string `json:"entity_address"`
	EntityCity    string `json:"entity_city"`
	EntityState   string `json:"entity_state"`
	EntityZip     string `json:"entity_zip"`
	EntityCountry string `json:"entity_country"` // IN
}

type goAMLSubjectJSON struct {
	// FINnet-compatible
	Name         string `json:"name"`
	Relationship string `json:"relationship"`
	AccountRef   string `json:"account_ref"`
	PAN          string `json:"pan"`
	AadhaarRef   string `json:"aadhaar_ref"`
	RiskRating   string `json:"risk_rating"`

	// goAML-specific — split name
	FirstName   string `json:"first_name"` // REQUIRED (or fallback to Name)
	LastName    string `json:"last_name"`
	Gender      string `json:"gender"`
	Title       string `json:"title"`
	Birthdate   string `json:"birthdate"`
	IDType      string `json:"id_type"`   // PAN, PASSPORT, etc.
	IDNumber    string `json:"id_number"` // the identifier value
	Nationality string `json:"nationality"`

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
	var errs []GoAMLValidationError

	var entity goAMLEntityJSON
	_ = json.Unmarshal(f.FilingEntity, &entity)

	var subj goAMLSubjectJSON
	_ = json.Unmarshal(f.Subject, &subj)

	var txs []goAMLTxJSON
	_ = json.Unmarshal(f.Transactions, &txs)

	// Entity / reporting officer — required
	if entity.RentityID == "" {
		errs = append(errs, GoAMLValidationError{"filing_entity.rentity_id", "Reporting entity ID (FIU-IND registration number) is required"})
	}
	if entity.OfficerFirstName == "" {
		errs = append(errs, GoAMLValidationError{"filing_entity.officer_first_name", "Reporting officer first name is required"})
	}
	if entity.OfficerLastName == "" {
		errs = append(errs, GoAMLValidationError{"filing_entity.officer_last_name", "Reporting officer last name is required"})
	}
	if entity.OfficerIDNumber == "" {
		errs = append(errs, GoAMLValidationError{"filing_entity.officer_id_number", "Reporting officer ID number (PAN) is required"})
	}
	if entity.OfficerIDType == "" {
		errs = append(errs, GoAMLValidationError{"filing_entity.officer_id_type", "Reporting officer ID type is required (e.g. PAN)"})
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
		subjID = subj.PAN
	}
	if subjID == "" {
		errs = append(errs, GoAMLValidationError{"subject.id_number", "Subject identifier (PAN or other ID) is required"})
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

// ── Builder ───────────────────────────────────────────────────────────────────

// BuildGoAMLXML produces a valid goAML 4.0 XML document for FIU-IND submission.
// Returns a validation error list if the filing is incomplete; XML is only
// returned when errs is empty.
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
		officerIDType = "PAN"
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
		subjIDNumber = subj.PAN
		subjIDType = "PAN"
	}
	if subjIDType == "" {
		subjIDType = "PAN"
	}

	// Indicators from suspicion basis
	indicators := mapGoAMLIndicators(f.SuspicionBasis)

	// Build transactions
	goTxs := make([]goAMLTransaction, 0, len(txs))
	for i, tx := range txs {
		cur := tx.Currency
		if cur == "" {
			cur = "INR"
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
			Nationality: defaultStr(subj.Nationality, "IN"),
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

	fIURef := f.FIUReference
	action := "Referred to FIU-IND under PMLA Section 12"

	report := GoAMLReport{
		RentityID:       entity.RentityID,
		RentityBranch:   defaultStr(entity.RentityBranch, entity.BranchCode),
		SubmissionCode:  submissionCode,
		ReportCode:      f.ReportType,
		EntityReference: fmt.Sprintf("OWL-%s", f.ID.String()[:8]),
		FIURefNumber:    fIURef,
		SubmissionDate:  time.Now().UTC().Format("2006-01-02"),
		CurrencyLocal:   "INR",

		ReportingPerson: goAMLPerson{
			Gender:      defaultStr(entity.OfficerGender, "M"),
			Title:       defaultStr(entity.OfficerTitle, "Mr"),
			FirstName:   entity.OfficerFirstName,
			LastName:    entity.OfficerLastName,
			Birthdate:   entity.OfficerBirthdate,
			IDNumber:    entity.OfficerIDNumber,
			IDType:      officerIDType,
			Nationality: "IN",
			Email:       defaultStr(entity.OfficerEmail, entity.ContactEmail),
			Phone:       entity.OfficerPhone,
			Occupation:  defaultStr(entity.OfficerOccupation, "Compliance Officer"),
		},

		Location: goAMLLocation{
			AddressType: "H",
			Address:     entity.EntityAddress,
			City:        entity.EntityCity,
			CountryCode: defaultStr(entity.EntityCountry, "IN"),
			State:       entity.EntityState,
			Zip:         entity.EntityZip,
		},

		Reason:     f.Narrative,
		Action:     action,
		Indicators: goAMLIndicators{Indicators: indicators},
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
