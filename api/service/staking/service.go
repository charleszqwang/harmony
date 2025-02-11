package staking

import (
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	protobuf "github.com/golang/protobuf/proto"
	"github.com/harmony-one/bls/ffi/go/bls"
	"github.com/harmony-one/harmony/internal/params"

	"github.com/harmony-one/harmony/accounts"
	"github.com/harmony-one/harmony/accounts/abi"
	proto "github.com/harmony-one/harmony/api/client/service/proto"
	proto_common "github.com/harmony-one/harmony/api/proto"
	"github.com/harmony-one/harmony/api/proto/message"
	msg_pb "github.com/harmony-one/harmony/api/proto/message"
	"github.com/harmony-one/harmony/common/denominations"
	"github.com/harmony-one/harmony/contracts"
	"github.com/harmony-one/harmony/core"
	"github.com/harmony-one/harmony/core/types"
	common2 "github.com/harmony-one/harmony/internal/common"
	"github.com/harmony-one/harmony/internal/genesis"
	hmykey "github.com/harmony-one/harmony/internal/keystore"
	"github.com/harmony-one/harmony/internal/utils"
	"github.com/harmony-one/harmony/p2p"
	"github.com/harmony-one/harmony/p2p/host"
)

const (
	// WaitTime is the delay time for resending staking transaction if the previous transaction did not get approved.
	WaitTime = 5 * time.Second
	// StakingContractAddress is the staking deployed contract address
	StakingContractAddress = "TODO(minhdoan): Create a PR to generate staking contract address"
	// StakingAmount is the amount of stake to put
	StakingAmount = 10
)

// State is the state of staking service.
type State byte

// Service is the staking service.
// Service requires private key here which is not a right design.
// In stead in the right design, the end-user who runs mining needs to provide signed tx to this service.
type Service struct {
	host          p2p.Host
	stopChan      chan struct{}
	stoppedChan   chan struct{}
	account       accounts.Account
	blsPublicKey  *bls.PublicKey
	stakingAmount int64
	state         State
	beaconChain   *core.BlockChain
	messageChan   chan *msg_pb.Message
}

// New returns staking service.
func New(host p2p.Host, account accounts.Account, beaconChain *core.BlockChain, blsPublicKey *bls.PublicKey) *Service {
	return &Service{
		host:          host,
		stopChan:      make(chan struct{}),
		stoppedChan:   make(chan struct{}),
		blsPublicKey:  blsPublicKey,
		stakingAmount: StakingAmount,
		beaconChain:   beaconChain,
	}
}

// StartService starts staking service.
func (s *Service) StartService() {
	utils.Logger().Info().Msg("Start Staking Service")
	s.Run()
}

// Run runs staking.
func (s *Service) Run() {
	tick := time.NewTicker(WaitTime)
	go func() {
		defer close(s.stoppedChan)
		// Do service first time and after that doing it every 5 minutes.
		// The reason we have to do it in every x minutes because of beacon chain syncing.
		time.Sleep(WaitTime)
		s.DoService()
		for {
			select {
			case <-tick.C:
				if s.IsStaked() {
					return
				}
				//s.DoService()
				return
			case <-s.stopChan:
				return
			}
		}
	}()
}

// IsStaked checks if the txn gets accepted and approved in the beacon chain.
func (s *Service) IsStaked() bool {
	return false
}

// DoService does staking.
func (s *Service) DoService() {
	utils.Logger().Info().Msg("Trying to send a staking transaction.")

	// TODO: no need to sync beacon chain to stake
	//if s.beaconChain == nil {
	//	utils.Logger().Info().Msg("Can not send a staking transaction because of nil beacon chain.")
	//	return
	//}

	if msg := s.createStakingMessage(); msg == nil {
		utils.Logger().Error().Msg("Can not create staking transaction")
	} else if err := s.host.SendMessageToGroups([]p2p.GroupID{p2p.GroupIDBeacon}, host.ConstructP2pMessage(byte(17), msg)); err != nil {
		utils.Logger().Warn().Err(err).Msg("cannot send staking message")
	} else {
		utils.Logger().Info().Msg("Sent staking transaction to the network.")
	}
}

func (s *Service) getStakingInfo() *proto.StakingContractInfoResponse {
	address := s.account.Address
	state, err := s.beaconChain.State()
	if err != nil {
		utils.Logger().Error().Msg("error to get beacon chain state when getting staking info")
		return nil
	}
	balance := state.GetBalance(address)
	if balance == common.Big0 {
		utils.Logger().Error().Msg("account balance empty when getting staking info")
		return nil
	}
	nonce := state.GetNonce(address)
	if nonce == 0 {
		utils.Logger().Error().Msg("nonce zero when getting staking info")
		return nil
	}
	return &proto.StakingContractInfoResponse{
		ContractAddress: StakingContractAddress,
		Balance:         balance.Bytes(),
		Nonce:           nonce,
	}
}

func (s *Service) getFakeStakingInfo() *proto.StakingContractInfoResponse {
	balance := big.NewInt(denominations.One)
	nonce := uint64(0) // TODO: make it a incrementing field

	priKey := genesis.GenesisBeaconAccountPriKey
	contractAddress := crypto.PubkeyToAddress(priKey.PublicKey)

	stakingContractAddress := crypto.CreateAddress(contractAddress, uint64(nonce))
	return &proto.StakingContractInfoResponse{
		ContractAddress: common2.MustAddressToBech32(stakingContractAddress),
		Balance:         balance.Bytes(),
		Nonce:           nonce,
	}
}

// Constructs the staking message
func constructStakingMessage(ts types.Transactions) []byte {
	tsBytes, err := rlp.EncodeToBytes(ts)
	if err == nil {
		msg := &message.Message{
			Type: message.MessageType_NEWNODE_BEACON_STAKING,
			Request: &message.Message_Staking{
				Staking: &message.StakingRequest{
					Transaction: tsBytes,
					NodeId:      "",
				},
			},
		}
		if data, err := protobuf.Marshal(msg); err == nil {
			return data
		}
	}
	utils.Logger().Error().Err(err).Msg("Error when creating staking message")
	return nil
}

func (s *Service) createRawStakingMessage() []byte {
	// TODO(minhdoan): Enable getStakingInfo back after testing.
	stakingInfo := s.getFakeStakingInfo()
	toAddress := common2.ParseAddr(stakingInfo.ContractAddress)

	abi, err := abi.JSON(strings.NewReader(contracts.StakeLockContractABI))
	if err != nil {
		utils.Logger().Error().Err(err).Msg("Failed to generate staking contract's ABI")
	}
	// TODO: the bls address should be signed by the bls private key
	blsPubKeyBytes := s.blsPublicKey.Serialize()
	if len(blsPubKeyBytes) != 96 {
		utils.Logger().Error().Int("size", len(blsPubKeyBytes)).Msg("Wrong bls pubkey size")
		return []byte{}
	}
	blsPubKeyPart1 := [32]byte{}
	blsPubKeyPart2 := [32]byte{}
	blsPubKeyPart3 := [32]byte{}
	copy(blsPubKeyPart1[:], blsPubKeyBytes[:32])
	copy(blsPubKeyPart2[:], blsPubKeyBytes[32:64])
	copy(blsPubKeyPart3[:], blsPubKeyBytes[64:96])
	bytesData, err := abi.Pack("lock", blsPubKeyPart1, blsPubKeyPart2, blsPubKeyPart3)

	if err != nil {
		utils.Logger().Error().Err(err).Msg("Failed to generate ABI function bytes data")
	}

	tx := types.NewTransaction(
		stakingInfo.Nonce,
		toAddress,
		0,
		big.NewInt(s.stakingAmount),
		params.TxGas*10,
		nil,
		bytesData,
	)

	// This is currently not called.
	chainID := big.NewInt(1) // TODO: wire the correct chain ID after staking flow is revamped.
	if signedTx, err := hmykey.SignTx(s.account, tx, chainID); err == nil {
		ts := types.Transactions{signedTx}
		return constructStakingMessage(ts)
	}
	return nil
}

func (s *Service) createStakingMessage() []byte {
	if msg := s.createRawStakingMessage(); msg != nil {
		return proto_common.ConstructStakingMessage(msg)
	}
	return nil
}

// StopService stops staking service.
func (s *Service) StopService() {
	utils.Logger().Info().Msg("Stopping staking service.")
	s.stopChan <- struct{}{}
	<-s.stoppedChan
	utils.Logger().Info().Msg("Role conversion stopped.")
}

// NotifyService notify service
func (s *Service) NotifyService(params map[string]interface{}) {
	return
}

// SetMessageChan sets up message channel to service.
func (s *Service) SetMessageChan(messageChan chan *msg_pb.Message) {
	s.messageChan = messageChan
}

// APIs for the services.
func (s *Service) APIs() []rpc.API {
	return nil
}
