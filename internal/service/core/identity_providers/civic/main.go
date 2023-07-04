package civic

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"

	"gitlab.com/rarimo/identity/kyc-service/internal/config"
	"gitlab.com/rarimo/identity/kyc-service/internal/crypto"
	"gitlab.com/rarimo/identity/kyc-service/internal/data"
	providers "gitlab.com/rarimo/identity/kyc-service/internal/service/core/identity_providers"
	"gitlab.com/rarimo/identity/kyc-service/internal/service/core/identity_providers/civic/contracts"
)

type Civic struct {
	log                 *logan.Entry
	masterQ             data.MasterQ
	GatekeeperNetworkId *big.Int
	ContractsVerifiers  map[ChainName]*contracts.GatewayVerifier

	skipSigCheck bool
}

func NewIdentityProvider(log *logan.Entry, masterQ data.MasterQ, config *config.Civic) (*Civic, error) {
	contractsVerifiers, err := newContractsVerifiers(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create contracts verifiers")
	}

	return &Civic{
		log:                 log,
		masterQ:             masterQ,
		GatekeeperNetworkId: config.GatekeeperNetworkId,
		ContractsVerifiers:  contractsVerifiers,
		skipSigCheck:        config.SkipSigCheck,
	}, nil
}

func (c *Civic) Verify(user *data.User, verifyDataRaw []byte) error {
	var verifyData VerificationData
	if err := json.Unmarshal(verifyDataRaw, &verifyData); err != nil {
		return errors.Wrap(err, "failed to unmarshal verification data")
	}

	if err := verifyData.Validate(); err != nil {
		return errors.Wrap(providers.ErrInvalidVerificationData, err.Error())
	}

	userAddr := common.HexToAddress(verifyData.Address)

	if err := c.verifySignature(verifyData, userAddr); err != nil {
		return errors.Wrap(err, "failed to verify signature")
	}

	if err := c.verifyGatewayToken(chainNameFromString[verifyData.ChainName], userAddr); err != nil {
		return errors.Wrap(err, "failed to verify gateway token")
	}

	user.EthAddress = &userAddr
	user.Status = data.UserStatusVerified

	providerDataRaw, err := json.Marshal(ProviderData{
		Address: userAddr,
	})
	if err != nil {
		return errors.Wrap(err, "failed to marshal provider data")
	}

	user.ProviderData = providerDataRaw

	return nil
}

func (c *Civic) verifySignature(verifyData VerificationData, userAddress common.Address) error {
	if !c.skipSigCheck {
		nonce, err := c.masterQ.NonceQ().FilterByAddress(verifyData.Address).Get()
		if err != nil {
			return errors.Wrap(err, "failed to get nonce by address")
		}
		if nonce == nil {
			return providers.ErrNonceNotFound
		}

		valid, err := crypto.VerifyEIP191Signature(verifyData.Signature, nonce.Message, userAddress)
		if err != nil || !valid {
			return providers.ErrInvalidUsersSignature
		}

		if err = c.masterQ.NonceQ().FilterByAddress(verifyData.Address).Delete(); err != nil {
			return errors.Wrap(err, "failed to delete nonce")
		}
	}

	return nil
}

func (c *Civic) verifyGatewayToken(chainName ChainName, userAddress common.Address) error {
	verifier, ok := c.ContractsVerifiers[chainName]
	if !ok {
		return ErrVerifierNotFound
	}

	valid, err := verifier.VerifyToken(nil, userAddress, c.GatekeeperNetworkId)
	if err != nil {
		return errors.Wrap(err, "failed to verify gateway token on chain")
	}
	if !valid {
		return ErrInvalidGatewayToken
	}

	return nil
}

func newContractsVerifiers(config *config.Civic) (map[ChainName]*contracts.GatewayVerifier, error) {
	ethereumVerifier, err := contracts.NewGatewayVerifier(config.GatewayTokenContract, config.EthereumRpc)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create Ethereum gateway verifier")
	}

	polygonVerifier, err := contracts.NewGatewayVerifier(config.GatewayTokenContract, config.PolygonRpc)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create Polygon gateway verifier")
	}

	arbitrumVerifier, err := contracts.NewGatewayVerifier(config.GatewayTokenContract, config.ArbitrumRpc)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create Arbitrum gateway verifier")
	}

	xdcVerifier, err := contracts.NewGatewayVerifier(config.GatewayTokenContract, config.XDCRpc)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create XDC gateway verifier")
	}

	return map[ChainName]*contracts.GatewayVerifier{
		EthereumChainName: ethereumVerifier,
		PolygonChainName:  polygonVerifier,
		ArbitrumChainName: arbitrumVerifier,
		XDCChainName:      xdcVerifier,
	}, nil
}
