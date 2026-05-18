package auth

import (
	"fmt"
	"strings"

	"github.com/go-ldap/ldap/v3"
)

// LDAPConfig holds the LDAP/AD connection parameters.
// All fields are optional; if URL is empty, LDAP auth is disabled.
type LDAPConfig struct {
	URL          string // e.g. ldap://dc.corp.com:389 or ldaps://dc.corp.com:636
	BaseDN       string // e.g. dc=corp,dc=com
	UserAttr     string // e.g. sAMAccountName (AD) or uid (OpenLDAP)
	BindDN       string // service account DN for directory search
	BindPassword string // service account password
}

func (c *LDAPConfig) Enabled() bool {
	return c != nil && c.URL != ""
}

// LDAPAuthenticator verifies credentials against an LDAP/AD server.
type LDAPAuthenticator struct {
	cfg *LDAPConfig
}

func NewLDAPAuthenticator(cfg *LDAPConfig) *LDAPAuthenticator {
	return &LDAPAuthenticator{cfg: cfg}
}

// Authenticate returns the user's email/UPN from LDAP on success.
// It performs a two-step bind: service-account search to find the user DN,
// then a bind with the user's own credentials to verify the password.
func (a *LDAPAuthenticator) Authenticate(username, password string) (email string, err error) {
	if password == "" {
		return "", ErrInvalidCredentials
	}

	conn, err := ldap.DialURL(a.cfg.URL)
	if err != nil {
		return "", fmt.Errorf("ldap dial: %w", err)
	}
	defer conn.Close()

	// Step 1: bind as service account to search for the user DN
	if a.cfg.BindDN != "" {
		if err := conn.Bind(a.cfg.BindDN, a.cfg.BindPassword); err != nil {
			return "", fmt.Errorf("ldap service bind: %w", err)
		}
	}

	// Step 2: search for the user entry
	filter := fmt.Sprintf("(%s=%s)", a.cfg.UserAttr, ldap.EscapeFilter(username))
	searchReq := ldap.NewSearchRequest(
		a.cfg.BaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		1, 0, false,
		filter,
		[]string{"dn", "mail", "userPrincipalName", "cn"},
		nil,
	)

	result, err := conn.Search(searchReq)
	if err != nil {
		return "", fmt.Errorf("ldap search: %w", err)
	}
	if len(result.Entries) == 0 {
		return "", ErrInvalidCredentials
	}

	userDN := result.Entries[0].DN
	email = result.Entries[0].GetAttributeValue("mail")
	if email == "" {
		email = result.Entries[0].GetAttributeValue("userPrincipalName")
	}
	// Fallback: construct email from username if not in directory
	if email == "" {
		parts := strings.Split(a.cfg.BaseDN, ",")
		domain := ""
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if strings.HasPrefix(strings.ToLower(p), "dc=") {
				if domain != "" {
					domain += "."
				}
				domain += p[3:]
			}
		}
		if domain != "" {
			email = username + "@" + domain
		} else {
			email = username
		}
	}

	// Step 3: verify the user's password by binding as them
	if err := conn.Bind(userDN, password); err != nil {
		return "", ErrInvalidCredentials
	}

	return email, nil
}
