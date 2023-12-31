package issuer

import (
	"time"

	"github.com/pkg/errors"
	"github.com/rarimo/issuer/resources"
)

var (
	ErrUnexpectedStatusCode = errors.New("unexpected status code")
)

const (
	issueEndpoint = "/credentials"
)

type ClaimType string

func (c ClaimType) String() string {
	return string(c)
}

const (
	ClaimTypeNaturalPerson     ClaimType = "NaturalPerson"
	ClaimTypeIdentityProviders ClaimType = "IdentityProviders"

	EmptyStringField  = "none"
	EmptyIntegerField = 0
)

type IdentityProviderName string

func (ipn IdentityProviderName) String() string {
	return string(ipn)
}

const (
	UnstoppableDomainsProviderName IdentityProviderName = "UnstoppableDomains"
	CivicProviderName              IdentityProviderName = "Civic"
	GitcoinProviderName            IdentityProviderName = "GitcoinPassport"
	WorldCoinProviderName          IdentityProviderName = "Worldcoin"
	KlerosProviderName             IdentityProviderName = "Kleros"
)

type IsNaturalPersonCredentialSubject struct {
	IsNatural string `json:"is_natural"`
}

type IdentityProvidersCredentialSubject struct {
	IdentityID       string               `json:"id"`
	Provider         IdentityProviderName `json:"provider"`
	IsNatural        int64                `json:"isNatural"`
	Address          string               `json:"address"`
	ProviderMetadata string               `json:"providerMetadata"`
}

type IdentityProviderMetadata struct {
	GitcoinPassportData      GitcoinPassportData `json:"gitcoinPassportData,omitempty"`
	WorldCoinData            WorldCoinData       `json:"worldcoinData,omitempty"`
	UnstoppableDomain        string              `json:"unstoppableDomain,omitempty"`
	CivicGatekeeperNetworkID int64               `json:"civicGatekeeperNetworkId,omitempty"`
}

type GitcoinPassportData struct {
	Score          string `json:"score"`
	AdditionalData string `json:"additionalData"`
}

type WorldCoinData struct {
	Score          string `json:"score"`
	AdditionalData string `json:"additionalData"`
}

func NewEmptyIdentityProvidersCredentialSubject() *IdentityProvidersCredentialSubject {
	return &IdentityProvidersCredentialSubject{
		IdentityID: EmptyStringField,
		Provider:   EmptyStringField,
		IsNatural:  EmptyIntegerField,
		Address:    EmptyStringField,
	}
}

type IssueClaimResponse struct {
	Data IssueClaimResponseData `json:"data"`
}

type IssueClaimResponseData struct {
	resources.Key
}

// CreateCredentialRequest defines model for CreateCredentialRequest.
type CreateCredentialRequest struct {
	CredentialSchema  string                              `json:"credentialSchema"`
	CredentialSubject *IdentityProvidersCredentialSubject `json:"credentialSubject"`
	Expiration        *time.Time                          `json:"expiration,omitempty"`
	MtProof           *bool                               `json:"mtProof,omitempty"`
	SignatureProof    *bool                               `json:"signatureProof,omitempty"`
	Type              string                              `json:"type"`
}

type UUIDResponse struct {
	Id string `json:"id"`
}

func (r UUIDResponse) IssueClaimResponse() IssueClaimResponse {
	return IssueClaimResponse{
		Data: IssueClaimResponseData{
			resources.Key{
				ID:   r.Id,
				Type: resources.CLAIM_ID,
			},
		},
	}
}
