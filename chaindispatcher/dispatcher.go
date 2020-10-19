package chaindispatcher

import (
	"context"
	"errors"
	"runtime/debug"
	"strings"

	"github.com/hbtc-chain/chainnode/chainadaptor"
	"github.com/hbtc-chain/chainnode/chainadaptor/bitcoin"
	"github.com/hbtc-chain/chainnode/chainadaptor/ethereum"
	"github.com/hbtc-chain/chainnode/chainadaptor/tron"
	"github.com/hbtc-chain/chainnode/config"
	"github.com/hbtc-chain/chainnode/proto"

	"github.com/ethereum/go-ethereum/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type CommonRequest interface {
	GetChain() string
}

type CommonReply = proto.SupportChainReply

type ChainType = string

type ChainDispatcher struct {
	registry map[ChainType]chainadaptor.ChainAdaptor
}

func New(conf *config.Config) (*ChainDispatcher, error) {
	dispatcher := ChainDispatcher{
		registry: make(map[ChainType]chainadaptor.ChainAdaptor),
	}

	chainAdaptorFactoryMap := map[string]func(conf *config.Config) (chainadaptor.ChainAdaptor, error){
		bitcoin.ChainName:  bitcoin.NewChainAdaptor,
		ethereum.ChainName: ethereum.NewChainAdaptor,
		tron.ChainName:     tron.NewChainAdaptor,
	}

	supportedChains := []string{bitcoin.ChainName, ethereum.ChainName, tron.ChainName}

	for _, c := range conf.Chains {
		if factory, ok := chainAdaptorFactoryMap[c]; ok {
			adaptor, err := factory(conf)
			if err != nil {
				log.Crit("failed to setup chain", "chain", c, "error", err)
			}
			dispatcher.registry[c] = adaptor
		} else {
			log.Error("unsupported chain", "chain", c, "supportedChains", supportedChains)
		}
	}
	return &dispatcher, nil
}

func NewLocal(network config.NetWorkType) *ChainDispatcher {
	dispatcher := ChainDispatcher{
		registry: make(map[ChainType]chainadaptor.ChainAdaptor),
	}

	chainAdaptorFactoryMap := map[string]func(network config.NetWorkType) chainadaptor.ChainAdaptor{
		bitcoin.ChainName:  bitcoin.NewLocalChainAdaptor,
		ethereum.ChainName: ethereum.NewLocalChainAdaptor,
		tron.ChainName:     tron.NewLocalChainAdaptor,
	}
	supportedChains := []string{bitcoin.ChainName, ethereum.ChainName, tron.ChainName}

	for _, c := range supportedChains {
		if factory, ok := chainAdaptorFactoryMap[c]; ok {
			dispatcher.registry[c] = factory(network)
		}
	}
	return &dispatcher
}

func (d *ChainDispatcher) Interceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {

	defer func() {
		if e := recover(); e != nil {
			log.Error("panic error", "msg", e)
			log.Debug(string(debug.Stack()))
			err = status.Errorf(codes.Internal, "Panic err: %v", e)
		}
	}()

	pos := strings.LastIndex(info.FullMethod, "/")
	method := info.FullMethod[pos+1:]

	chain := req.(CommonRequest).GetChain()
	log.Info(method, "chain", chain, "req", req)

	resp, err = handler(ctx, req)
	log.Debug("Finish handling", "resp", resp, "err", err)
	return
}

func (d *ChainDispatcher) preHandler(req interface{}) (resp *CommonReply) {
	chain := req.(CommonRequest).GetChain()

	if _, ok := d.registry[chain]; !ok {
		return &CommonReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  config.UnsupportedChain,
		}

	}
	return nil
}

// SupportAsset query symbol support or not
func (d *ChainDispatcher) SupportChain(_ context.Context, req *proto.SupportChainRequest) (*proto.SupportChainReply, error) {
	resp := d.preHandler(req)
	if resp != nil {
		return resp, nil
	}

	return &proto.SupportChainReply{
		Code:    proto.ReturnCode_SUCCESS,
		Support: true,
	}, nil
}

// ConvertAddress convert BlueHelix chain's pubkey to a actual chain address
func (d *ChainDispatcher) ConvertAddress(_ context.Context, req *proto.ConvertAddressRequest) (*proto.ConvertAddressReply, error) {
	resp := d.preHandler(req)
	if resp != nil {
		return &proto.ConvertAddressReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  config.UnsupportedChain,
		}, nil
	}
	return d.registry[req.Chain].ConvertAddress(req)
}

// ValidAddress check the address valid or not
func (d *ChainDispatcher) ValidAddress(_ context.Context, req *proto.ValidAddressRequest) (*proto.ValidAddressReply, error) {
	resp := d.preHandler(req)
	if resp != nil {
		return &proto.ValidAddressReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  config.UnsupportedChain,
		}, nil
	}
	return d.registry[req.Chain].ValidAddress(req)
}

func (d *ChainDispatcher) QueryBalance(_ context.Context, req *proto.QueryBalanceRequest) (*proto.QueryBalanceReply, error) {
	resp := d.preHandler(req)
	if resp != nil {
		return &proto.QueryBalanceReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  config.UnsupportedChain,
		}, nil
	}
	return d.registry[req.Chain].QueryBalance(req)
}

func (d *ChainDispatcher) QueryUtxo(_ context.Context, req *proto.QueryUtxoRequest) (*proto.QueryUtxoReply, error) {
	resp := d.preHandler(req)
	if resp != nil {
		return &proto.QueryUtxoReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  config.UnsupportedChain,
		}, nil
	}
	return d.registry[req.Chain].QueryUtxo(req)
}

func (d *ChainDispatcher) QueryNonce(_ context.Context, req *proto.QueryNonceRequest) (*proto.QueryNonceReply, error) {
	resp := d.preHandler(req)
	if resp != nil {
		return &proto.QueryNonceReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  config.UnsupportedChain,
		}, nil
	}
	return d.registry[req.Chain].QueryNonce(req)
}

func (d *ChainDispatcher) QueryGasPrice(_ context.Context, req *proto.QueryGasPriceRequest) (*proto.QueryGasPriceReply, error) {
	resp := d.preHandler(req)
	if resp != nil {
		return &proto.QueryGasPriceReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  config.UnsupportedChain,
		}, nil
	}
	return d.registry[req.Chain].QueryGasPrice(req)
}

func (d *ChainDispatcher) QueryUtxoTransaction(_ context.Context, req *proto.QueryTransactionRequest) (*proto.QueryUtxoTransactionReply, error) {
	resp := d.preHandler(req)
	if resp != nil {
		return &proto.QueryUtxoTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  config.UnsupportedChain,
		}, nil
	}
	return d.registry[req.Chain].QueryUtxoTransaction(req)
}

func (d *ChainDispatcher) QueryAccountTransaction(_ context.Context, req *proto.QueryTransactionRequest) (*proto.QueryAccountTransactionReply, error) {
	resp := d.preHandler(req)
	if resp != nil {
		return &proto.QueryAccountTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  config.UnsupportedChain,
		}, nil
	}
	return d.registry[req.Chain].QueryAccountTransaction(req)
}

func (d *ChainDispatcher) CreateUtxoTransaction(_ context.Context, req *proto.CreateUtxoTransactionRequest) (*proto.CreateUtxoTransactionReply, error) {
	resp := d.preHandler(req)
	if resp != nil {
		return &proto.CreateUtxoTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  config.UnsupportedChain,
		}, nil
	}
	return d.registry[req.Chain].CreateUtxoTransaction(req)
}

func (d *ChainDispatcher) CreateAccountTransaction(_ context.Context, req *proto.CreateAccountTransactionRequest) (*proto.CreateAccountTransactionReply, error) {
	resp := d.preHandler(req)
	if resp != nil {
		return &proto.CreateAccountTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  config.UnsupportedChain,
		}, nil
	}
	return d.registry[req.Chain].CreateAccountTransaction(req)
}

func (d *ChainDispatcher) CreateUtxoSignedTransaction(_ context.Context, req *proto.CreateUtxoSignedTransactionRequest) (*proto.CreateSignedTransactionReply, error) {
	resp := d.preHandler(req)
	if resp != nil {
		return &proto.CreateSignedTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  config.UnsupportedChain,
		}, nil
	}
	return d.registry[req.Chain].CreateUtxoSignedTransaction(req)
}

func (d *ChainDispatcher) CreateAccountSignedTransaction(_ context.Context, req *proto.CreateAccountSignedTransactionRequest) (*proto.CreateSignedTransactionReply, error) {
	resp := d.preHandler(req)
	if resp != nil {
		return &proto.CreateSignedTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  config.UnsupportedChain,
		}, nil
	}
	return d.registry[req.Chain].CreateAccountSignedTransaction(req)
}

func (d *ChainDispatcher) VerifyAccountSignedTransaction(_ context.Context, req *proto.VerifySignedTransactionRequest) (*proto.VerifySignedTransactionReply, error) {
	resp := d.preHandler(req)
	if resp != nil {
		return &proto.VerifySignedTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  config.UnsupportedChain,
		}, nil
	}
	return d.registry[req.Chain].VerifyAccountSignedTransaction(req)
}

func (d *ChainDispatcher) VerifyUtxoSignedTransaction(_ context.Context, req *proto.VerifySignedTransactionRequest) (*proto.VerifySignedTransactionReply, error) {
	resp := d.preHandler(req)
	if resp != nil {
		return &proto.VerifySignedTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  config.UnsupportedChain,
		}, nil
	}
	return d.registry[req.Chain].VerifyUtxoSignedTransaction(req)
}

func (d *ChainDispatcher) QueryAccountTransactionFromData(_ context.Context, req *proto.QueryTransactionFromDataRequest) (*proto.QueryAccountTransactionReply, error) {
	resp := d.preHandler(req)
	if resp != nil {
		return &proto.QueryAccountTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  config.UnsupportedChain,
		}, nil
	}
	return d.registry[req.Chain].QueryAccountTransactionFromData(req)
}

func (d *ChainDispatcher) QueryAccountTransactionFromSignedData(_ context.Context, req *proto.QueryTransactionFromSignedDataRequest) (*proto.QueryAccountTransactionReply, error) {
	resp := d.preHandler(req)
	if resp != nil {
		return &proto.QueryAccountTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  config.UnsupportedChain,
		}, nil
	}
	return d.registry[req.Chain].QueryAccountTransactionFromSignedData(req)
}

func (d *ChainDispatcher) QueryUtxoTransactionFromData(_ context.Context, req *proto.QueryTransactionFromDataRequest) (*proto.QueryUtxoTransactionReply, error) {
	resp := d.preHandler(req)
	if resp != nil {
		return &proto.QueryUtxoTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  config.UnsupportedChain,
		}, nil
	}
	return d.registry[req.Chain].QueryUtxoTransactionFromData(req)
}

func (d *ChainDispatcher) QueryUtxoTransactionFromSignedData(_ context.Context, req *proto.QueryTransactionFromSignedDataRequest) (*proto.QueryUtxoTransactionReply, error) {
	resp := d.preHandler(req)
	if resp != nil {
		return &proto.QueryUtxoTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  config.UnsupportedChain,
		}, nil
	}
	return d.registry[req.Chain].QueryUtxoTransactionFromSignedData(req)

}

func (d *ChainDispatcher) BroadcastTransaction(_ context.Context, req *proto.BroadcastTransactionRequest) (*proto.BroadcastTransactionReply, error) {
	resp := d.preHandler(req)
	if resp != nil {
		return &proto.BroadcastTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  config.UnsupportedChain,
		}, nil
	}
	return d.registry[req.Chain].BroadcastTransaction(req)
}

func (d *ChainDispatcher) QueryUtxoInsFromData(_ context.Context, req *proto.QueryUtxoInsFromDataRequest) (*proto.QueryUtxoInsReply, error) {
	resp := d.preHandler(req)
	if resp != nil {
		return &proto.QueryUtxoInsReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  config.UnsupportedChain,
		}, nil
	}
	return d.registry[req.Chain].QueryUtxoInsFromData(req)
}

func (d *ChainDispatcher) GetLatestBlockHeight(chain string) (int64, error) {
	if handler, ok := d.registry[chain]; ok {
		return handler.GetLatestBlockHeight()
	}

	return 0, errors.New(config.UnsupportedChain)
}

func (d *ChainDispatcher) GetAccountTransactionByHeight(chain string, height int64) (<-chan *proto.QueryAccountTransactionReply, <-chan error) {
	errCh := make(chan error, 20)
	replyCh := make(chan *proto.QueryAccountTransactionReply, 20)

	go func() {
		// defer close(errCh)
		defer close(replyCh)
		//defer close(errCh)
		if handler, ok := d.registry[chain]; ok {
			handler.GetAccountTransactionByHeight(height, replyCh, errCh)
		} else {
			errCh <- errors.New(config.UnsupportedChain)
		}
	}()
	return replyCh, errCh
}

func (d *ChainDispatcher) GetUtxoTransactionByHeight(chain string, height int64) (<-chan *proto.QueryUtxoTransactionReply, <-chan error) {
	errCh := make(chan error)
	replyCh := make(chan *proto.QueryUtxoTransactionReply)

	go func() {
		defer close(errCh)
		defer close(replyCh)
		if handler, ok := d.registry[chain]; ok {
			handler.GetUtxoTransactionByHeight(height, replyCh, errCh)
		} else {
			errCh <- errors.New(config.UnsupportedChain)
		}
	}()
	return replyCh, errCh
}
