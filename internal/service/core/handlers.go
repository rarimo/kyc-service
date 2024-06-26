package core

import (
	"encoding/json"
	"fmt"
	"time"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"
	core "github.com/iden3/go-iden3-core/v2"
	"github.com/pkg/errors"
	"github.com/rarimo/humanornot-svc/internal/crypto"
	"github.com/rarimo/humanornot-svc/internal/data"
	"github.com/rarimo/humanornot-svc/internal/service/api/requests"
	identityproviders "github.com/rarimo/humanornot-svc/internal/service/core/identity_providers"
	unstopdom "github.com/rarimo/humanornot-svc/internal/service/core/identity_providers/unstoppable_domains"
	"github.com/rarimo/humanornot-svc/internal/service/core/issuer"
)

func (k *humanornotSvc) NewVerifyRequest(req *requests.VerifyRequest) (*data.User, error) {
	prevUser, err := k.db.UsersQ().WhereIdentityID(data.NewIdentityID(req.IdentityID)).Get()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get user from db with the same identityID")
	}
	if prevUser != nil && prevUser.Status != data.UserStatusUnverified {
		return nil, ErrUserAlreadyVerifiedByIdentityID
	}

	newUser := data.User{
		ID:         uuid.New(),
		Status:     data.UserStatusInitialized,
		CreatedAt:  time.Now(),
		IdentityID: data.NewIdentityID(req.IdentityID),
	}

	credentialSubject, err := k.verifyUser(req, &newUser)
	if err != nil {
		return nil, errors.Wrap(err, "failed to verify user")
	}

	err = k.db.Transaction(func() error {
		if err = k.db.UsersQ().Upsert(&newUser); err != nil {
			return errors.Wrap(err, "failed to upsert user into db")
		}

		// "1" == true
		credentialSubject.IsNatural = 1
		if newUser.Status == data.UserStatusVerified {
			userDID, err := core.ParseDIDFromID(req.IdentityID)
			if err != nil {
				return errors.Wrap(err, "failed to parse DID from ID")
			}
			credentialSubject.IdentityID = userDID.String()

			sigProof := true

			credentialReq := issuer.CreateCredentialRequest{
				CredentialSchema:  k.issuer.SchemaURL(),
				CredentialSubject: credentialSubject,
				Type:              k.issuer.SchemaType(),
				SignatureProof:    &sigProof,
				MtProof:           &sigProof,
			}

			resp, err := k.issuer.IssueClaim(
				newUser.IdentityID.ID,
				issuer.ClaimTypeIdentityProviders,
				credentialReq,
			)
			if err != nil {
				return errors.Wrap(err, "failed to issue claim")
			}

			claimID, err := uuid.Parse(resp.Data.ID)
			if err != nil {
				return errors.Wrap(err, "failed to parse UUID")
			}

			newUser.ClaimID = claimID

			if err := k.db.UsersQ().Update(&newUser); err != nil {
				return errors.Wrap(err, fmt.Sprintf("failed to update user %s", newUser.ID.String()))
			}
		}

		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute db transaction")
	}

	return &newUser, nil
}

func (k *humanornotSvc) NewNonce(req *requests.NonceRequest) (*data.Nonce, error) {
	if err := k.db.NonceQ().WhereEthAddress(req.Address).Delete(); err != nil {
		return nil, errors.Wrap(err, "failed to delete old nonce")
	}

	newNonce, err := crypto.NewNonce()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate nonce")
	}

	nonce := &data.Nonce{
		EthAddress: req.Address,
		Nonce:      newNonce,
		ExpiresAt:  time.Now().Add(k.nonceLifetime),
	}

	err = k.db.NonceQ().Insert(nonce)
	if err != nil {
		return nil, errors.Wrap(err, "failed to insert nonce into db")
	}

	return nonce, nil
}

func (k *humanornotSvc) GetVerifyStatus(req *requests.VerifyStatusRequest) (*data.User, error) {
	user, err := k.db.UsersQ().WhereID(req.VerifyID).Get()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get user from db")
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	return user, nil
}

func (k *humanornotSvc) verifyUser(
	req *requests.VerifyRequest, newUser *data.User,
) (*issuer.IdentityProvidersCredentialSubject, error) {
	credSubject, providerHash, err := k.identityProviders[req.IdentityProviderName].Verify(newUser, req.ProviderData)
	if err != nil {
		return nil, errors.Wrap(err, "failed to verify with identity provider")
	}

	user, err := k.db.UsersQ().WhereProviderHash(providerHash).Get()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get user from db with the same providerDataHash")
	}
	if user != nil {
		return nil, ErrDuplicatedProviderData
	}

	newUser.ProviderHash = providerHash

	return credSubject, nil
}

func (k *humanornotSvc) GetProviderByIdentityId(req *requests.GetProviderByIdentityIdRequest) (identityproviders.IdentityProviderName, error) {
	user, err := k.db.UsersQ().WhereIdentityID(data.NewIdentityID(req.IdentityID)).Get()
	if err != nil {
		return "", errors.Wrap(err, "failed to get user from db with provided identityID")
	}

	if user == nil {
		return "", identityproviders.ErrUserNotFound
	}

	domainData := unstopdom.Domain{}
	if err := json.Unmarshal(user.ProviderData, &domainData); err != nil {
		return "", errors.Wrap(err, "failed to unmarshal json")
	}

	civicHash := ethcrypto.Keccak256(user.EthAddress.Bytes(), identityproviders.CivicIdentityProvider.Bytes())
	gitcoinPassportHash := ethcrypto.Keccak256(user.EthAddress.Bytes(), identityproviders.GitCoinPassportIdentityProvider.Bytes())
	klerosHash := ethcrypto.Keccak256(user.EthAddress.Bytes(), identityproviders.KlerosIdentityProvider.Bytes())
	unstoppableDomainsHash := ethcrypto.Keccak256([]byte(domainData.Domain), identityproviders.UnstoppableDomainsIdentityProvider.Bytes())
	worldCoinHash := ethcrypto.Keccak256(user.EthAddress.Bytes(), identityproviders.WorldCoinIdentityProvider.Bytes())

	switch string(user.ProviderHash) {
	case string(civicHash):
		return identityproviders.CivicIdentityProvider, nil
	case string(gitcoinPassportHash):
		return identityproviders.GitCoinPassportIdentityProvider, nil
	case string(klerosHash):
		return identityproviders.KlerosIdentityProvider, nil
	case string(unstoppableDomainsHash):
		return identityproviders.UnstoppableDomainsIdentityProvider, nil
	case string(worldCoinHash):
		return identityproviders.WorldCoinIdentityProvider, nil
	default:
		return "", identityproviders.ErrProviderNotFound
	}
}
