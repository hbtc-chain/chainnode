package fallback

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hbtc-chain/chainnode/config"
	"github.com/hbtc-chain/chainnode/proto"
)

type ChainAdaptor struct{}

func (d *ChainAdaptor) ConvertAddress(*proto.ConvertAddressRequest) (*proto.ConvertAddressReply, error) {
	return &proto.ConvertAddressReply{
		Code: proto.ReturnCode_ERROR,
		Msg:  config.UnsupportedOperation,
	}, nil
}

func (d *ChainAdaptor) ValidAddress(*proto.ValidAddressRequest) (*proto.ValidAddressReply, error) {
	return &proto.ValidAddressReply{
		Code: proto.ReturnCode_ERROR,
		Msg:  config.UnsupportedOperation,
	}, nil
}

func (d *ChainAdaptor) QueryBalance(*proto.QueryBalanceRequest) (*proto.QueryBalanceReply, error) {
	return &proto.QueryBalanceReply{
		Code: proto.ReturnCode_ERROR,
		Msg:  config.UnsupportedOperation,
	}, nil
}

func (d *ChainAdaptor) QueryNonce(*proto.QueryNonceRequest) (*proto.QueryNonceReply, error) {
	return &proto.QueryNonceReply{
		Code: proto.ReturnCode_ERROR,
		Msg:  config.UnsupportedOperation,
	}, nil
}

func (d *ChainAdaptor) QueryGasPrice(*proto.QueryGasPriceRequest) (*proto.QueryGasPriceReply, error) {
	return &proto.QueryGasPriceReply{
		Code: proto.ReturnCode_ERROR,
		Msg:  config.UnsupportedOperation,
	}, nil
}

func (d *ChainAdaptor) CreateUtxoTransaction(*proto.CreateUtxoTransactionRequest) (*proto.CreateUtxoTransactionReply, error) {
	return &proto.CreateUtxoTransactionReply{
		Code: proto.ReturnCode_ERROR,
		Msg:  config.UnsupportedOperation,
	}, nil
}

func (d *ChainAdaptor) CreateAccountTransaction(*proto.CreateAccountTransactionRequest) (*proto.CreateAccountTransactionReply, error) {
	return &proto.CreateAccountTransactionReply{
		Code: proto.ReturnCode_ERROR,
		Msg:  config.UnsupportedOperation,
	}, nil
}

func (d *ChainAdaptor) CreateUtxoSignedTransaction(*proto.CreateUtxoSignedTransactionRequest) (*proto.CreateSignedTransactionReply, error) {
	return &proto.CreateSignedTransactionReply{
		Code: proto.ReturnCode_ERROR,
		Msg:  config.UnsupportedOperation,
	}, nil
}

func (d *ChainAdaptor) CreateAccountSignedTransaction(*proto.CreateAccountSignedTransactionRequest) (*proto.CreateSignedTransactionReply, error) {
	return &proto.CreateSignedTransactionReply{
		Code: proto.ReturnCode_ERROR,
		Msg:  config.UnsupportedOperation,
	}, nil
}

func (d *ChainAdaptor) QueryAccountTransactionFromData(*proto.QueryTransactionFromDataRequest) (*proto.QueryAccountTransactionReply, error) {
	return &proto.QueryAccountTransactionReply{
		Code: proto.ReturnCode_ERROR,
		Msg:  config.UnsupportedOperation,
	}, nil
}

func (d *ChainAdaptor) QueryAccountTransactionFromSignedData(*proto.QueryTransactionFromSignedDataRequest) (*proto.QueryAccountTransactionReply, error) {
	return &proto.QueryAccountTransactionReply{
		Code: proto.ReturnCode_ERROR,
		Msg:  config.UnsupportedOperation,
	}, nil
}

func (d *ChainAdaptor) QueryUtxoTransactionFromData(*proto.QueryTransactionFromDataRequest) (*proto.QueryUtxoTransactionReply, error) {
	return &proto.QueryUtxoTransactionReply{
		Code: proto.ReturnCode_ERROR,
		Msg:  config.UnsupportedOperation,
	}, nil
}

func (d *ChainAdaptor) QueryUtxoTransactionFromSignedData(*proto.QueryTransactionFromSignedDataRequest) (*proto.QueryUtxoTransactionReply, error) {
	return &proto.QueryUtxoTransactionReply{
		Code: proto.ReturnCode_ERROR,
		Msg:  config.UnsupportedOperation,
	}, nil
}

func (d *ChainAdaptor) BroadcastTransaction(*proto.BroadcastTransactionRequest) (*proto.BroadcastTransactionReply, error) {
	return &proto.BroadcastTransactionReply{
		Code: proto.ReturnCode_ERROR,
		Msg:  config.UnsupportedOperation,
	}, nil
}

func (d *ChainAdaptor) QueryUtxo(*proto.QueryUtxoRequest) (*proto.QueryUtxoReply, error) {
	return &proto.QueryUtxoReply{
		Code: proto.ReturnCode_ERROR,
		Msg:  config.UnsupportedOperation,
	}, nil
}

func (d *ChainAdaptor) QueryUtxoTransaction(*proto.QueryTransactionRequest) (*proto.QueryUtxoTransactionReply, error) {
	return &proto.QueryUtxoTransactionReply{
		Code: proto.ReturnCode_ERROR,
		Msg:  config.UnsupportedOperation,
	}, nil
}

func (d *ChainAdaptor) QueryAccountTransaction(*proto.QueryTransactionRequest) (*proto.QueryAccountTransactionReply, error) {
	return &proto.QueryAccountTransactionReply{
		Code: proto.ReturnCode_ERROR,
		Msg:  config.UnsupportedOperation,
	}, nil
}

func (d *ChainAdaptor) VerifyAccountSignedTransaction(*proto.VerifySignedTransactionRequest) (*proto.VerifySignedTransactionReply, error) {
	return nil, status.Error(codes.InvalidArgument, config.UnsupportedOperation)
}

func (d *ChainAdaptor) VerifyUtxoSignedTransaction(*proto.VerifySignedTransactionRequest) (*proto.VerifySignedTransactionReply, error) {
	return nil, status.Error(codes.InvalidArgument, config.UnsupportedOperation)
}

func (d *ChainAdaptor) QueryUtxoInsFromData(*proto.QueryUtxoInsFromDataRequest) (*proto.QueryUtxoInsReply, error) {
	return &proto.QueryUtxoInsReply{
		Code: proto.ReturnCode_ERROR,
		Msg:  config.UnsupportedOperation,
	}, nil
}

func (d *ChainAdaptor) GetLatestBlockHeight() (int64, error) {
	return 0, errors.New(config.UnsupportedOperation)
}

func (d *ChainAdaptor) GetUtxoTransactionByHeight(_ int64, _ chan *proto.QueryUtxoTransactionReply, errCh chan error) {
	errCh <- errors.New(config.UnsupportedOperation)
}

func (d *ChainAdaptor) GetAccountTransactionByHeight(_ int64, _ chan *proto.QueryAccountTransactionReply, errCh chan error) {
	errCh <- errors.New(config.UnsupportedOperation)
}
