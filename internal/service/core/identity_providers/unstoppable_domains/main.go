package unstopdom

import (
	"encoding/json"
	"net/http"

	"gitlab.com/rarimo/identity/kyc-service/internal/crypto"
	providers "gitlab.com/rarimo/identity/kyc-service/internal/service/core/identity_providers"

	"github.com/ethereum/go-ethereum/common"
	"github.com/imroc/req/v3"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"

	"gitlab.com/rarimo/identity/kyc-service/internal/config"
	"gitlab.com/rarimo/identity/kyc-service/internal/data"
)

type UnstoppableDomains struct {
	log *logan.Entry
	*req.Client
}

func NewIdentityProvider(log *logan.Entry, config *config.UnstoppableDomains) *UnstoppableDomains {
	authBaseURL := config.AuthBaseURL
	if authBaseURL == "" {
		log.Debugf("Base URL for Unstoppable domains not found, the default %s is set", defaultAuthBaseURL)
		authBaseURL = defaultAuthBaseURL
	}

	return &UnstoppableDomains{
		Client: req.C().
			SetBaseURL(authBaseURL).
			SetLogger(log),
	}
}

func (u *UnstoppableDomains) Verify(user *data.User, verifyDataRaw []byte) error {
	var verifyData VerificationData
	if err := json.Unmarshal(verifyDataRaw, &verifyData); err != nil {
		return errors.Wrap(err, "failed to unmarshal verification data")
	}

	userInfo, err := u.retrieveUserInfo(verifyData.AccessToken)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve user info")
	}

	isValidSignature, err := verifyUserInfoSignature(userInfo)
	if err != nil {
		return errors.Wrap(err, "failed to verify user's signature")
	}
	if !isValidSignature {
		return providers.ErrInvalidUsersSignature
	}

	address := common.HexToAddress(userInfo.WalletAddress)
	user.EthAddress = &address
	user.Status = data.UserStatusVerified

	domainInfoRaw, err := json.Marshal(Domain{
		Domain: userInfo.Domain,
	})
	if err != nil {
		return errors.Wrap(err, "failed to marshal provider data")
	}

	user.ProviderData = domainInfoRaw

	return nil
}

func (u *UnstoppableDomains) retrieveUserInfo(accessToken string) (*UserInfo, error) {
	var result UserInfo

	response, err := u.R().
		SetBearerAuthToken(accessToken).
		SetSuccessResult(&result).
		Get(userInfoEndpoint)
	if err != nil {
		return nil, errors.Wrap(err, "failed to send get request")
	}

	if response.StatusCode >= 299 {
		if response.StatusCode == http.StatusUnauthorized {
			return nil, providers.ErrInvalidAccessToken
		}

		return nil, errors.Wrapf(ErrUnexpectedStatusCode, response.String())
	}

	return &result, nil
}

func verifyUserInfoSignature(userInfo *UserInfo) (bool, error) {
	if ok := common.IsHexAddress(userInfo.WalletAddress); !ok {
		return false, ErrInvalidWalletAddress
	}

	result, err := crypto.VerifyEIP191Signature(
		userInfo.EIP4361Signature, userInfo.EIP4361Message, common.HexToAddress(userInfo.WalletAddress),
	)
	if err != nil {
		return false, errors.Wrap(err, "failed to verify EIP191 signature")
	}

	return result, nil
}
