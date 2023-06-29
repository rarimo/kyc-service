package core

import (
	"gitlab.com/rarimo/identity/kyc-service/internal/service/core/identity_providers/worldcoin"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"gitlab.com/rarimo/identity/kyc-service/internal/config"
	"gitlab.com/rarimo/identity/kyc-service/internal/data"
	"gitlab.com/rarimo/identity/kyc-service/internal/data/pg"
	"gitlab.com/rarimo/identity/kyc-service/internal/service/api/requests"
	providers "gitlab.com/rarimo/identity/kyc-service/internal/service/core/identity_providers"
	unstopdom "gitlab.com/rarimo/identity/kyc-service/internal/service/core/identity_providers/unstoppable_domains"
	"gitlab.com/rarimo/identity/kyc-service/internal/service/core/issuer"
)

type KYCService interface {
	NewVerifyRequest(*requests.VerifyRequest) (*data.User, error)
}

type kycService struct {
	db                data.MasterQ
	issuer            issuer.Issuer
	identityProviders map[providers.IdentityProviderName]providers.IdentityProvider
}

func NewKYCService(cfg config.Config) KYCService {
	return &kycService{
		db:     pg.NewMasterQ(cfg.DB()),
		issuer: issuer.New(cfg.Log(), cfg.Issuer()),
		identityProviders: map[providers.IdentityProviderName]providers.IdentityProvider{
			providers.UnstoppableDomainsIdentityProvider: unstopdom.New(
				cfg.Log().WithField("provider", providers.UnstoppableDomainsIdentityProvider),
				cfg.UnstoppableDomains(),
			),
			providers.WorldCoinIdentityProvider: worldcoin.NewIdentityProvider(
				cfg.Log().WithField("provider", providers.WorldCoinIdentityProvider),
				cfg.WorldcoinSettings(),
			),
		},
	}
}

func (k *kycService) NewVerifyRequest(req *requests.VerifyRequest) (*data.User, error) {
	prevUser, err := k.db.UsersQ().WhereIdentityID(data.NewIdentityID(req.IdentityID)).Get()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get user from db with the same identityID")
	}
	if prevUser != nil {
		return nil, ErrUserAlreadyVerifiedByIdentityID
	}

	newUser := data.User{
		ID:         uuid.New(),
		Status:     data.UserStatusInitialized,
		CreatedAt:  time.Now(),
		IdentityID: data.NewIdentityID(req.IdentityID),
	}

	err = k.identityProviders[req.IdentityProviderName].Verify(&newUser, req.ProviderData)
	if err != nil {
		return nil, errors.Wrap(err, "failed to verify user")
	}

	prevUser, err = k.db.UsersQ().WhereEthAddress(newUser.EthAddress).Get()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get user from db with the same ethAddress")
	}
	if prevUser != nil {
		return nil, ErrUserAlreadyVerifiedByEthAddress
	}

	if newUser.Status == data.UserStatusVerified {
		_, err = k.issuer.IssueClaim(
			newUser.IdentityID.ID,
			issuer.ClaimTypeNaturalPerson,
			issuer.IsNaturalPersonCredentialSubject{
				IsNatural: "1",
			})
		if err != nil {
			return nil, errors.Wrap(err, "failed to issue claim")
		}
	}

	err = k.db.UsersQ().Insert(&newUser)
	if err != nil {
		return nil, errors.Wrap(err, "failed to insert new user into db")
	}

	return &newUser, nil
}
