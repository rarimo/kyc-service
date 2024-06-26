package identityproviders

import "github.com/pkg/errors"

var (
	// Unauthorized errors
	ErrNonceNotFound         = errors.New("nonce for provided address not found")
	ErrInvalidUsersSignature = errors.New("invalid signature")
	ErrInvalidAccessToken    = errors.New("invalid access token")

	// Bad request errors
	ErrInvalidVerificationData = errors.New("verification data is invalid")

	// Not found errors
	ErrProviderNotFound = errors.New("provider not found")
	ErrUserNotFound     = errors.New("user not found")
)

type IdentityProviderName string

func (ipn IdentityProviderName) String() string {
	return string(ipn)
}

func (ipn IdentityProviderName) Bytes() []byte {
	return []byte(ipn.String())
}

const (
	UnstoppableDomainsIdentityProvider IdentityProviderName = "unstoppable_domains"
	CivicIdentityProvider              IdentityProviderName = "civic"
	GitCoinPassportIdentityProvider    IdentityProviderName = "gitcoin_passport"
	WorldCoinIdentityProvider          IdentityProviderName = "worldcoin"
	KlerosIdentityProvider             IdentityProviderName = "kleros"
)

var IdentityProviderNames = map[string]IdentityProviderName{
	UnstoppableDomainsIdentityProvider.String(): UnstoppableDomainsIdentityProvider,
	CivicIdentityProvider.String():              CivicIdentityProvider,
	GitCoinPassportIdentityProvider.String():    GitCoinPassportIdentityProvider,
	WorldCoinIdentityProvider.String():          WorldCoinIdentityProvider,
	KlerosIdentityProvider.String():             KlerosIdentityProvider,
}
