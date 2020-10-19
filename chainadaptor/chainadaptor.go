package chainadaptor

import (
	"github.com/hbtc-chain/chainnode/proto"
)

type ChainAdaptor interface {
	ConvertAddress(req *proto.ConvertAddressRequest) (*proto.ConvertAddressReply, error)
	ValidAddress(req *proto.ValidAddressRequest) (*proto.ValidAddressReply, error)
	QueryBalance(req *proto.QueryBalanceRequest) (*proto.QueryBalanceReply, error)
	QueryNonce(req *proto.QueryNonceRequest) (*proto.QueryNonceReply, error)
	QueryGasPrice(req *proto.QueryGasPriceRequest) (*proto.QueryGasPriceReply, error)
	CreateUtxoTransaction(req *proto.CreateUtxoTransactionRequest) (*proto.CreateUtxoTransactionReply, error)
	CreateAccountTransaction(req *proto.CreateAccountTransactionRequest) (*proto.CreateAccountTransactionReply, error)
	CreateUtxoSignedTransaction(req *proto.CreateUtxoSignedTransactionRequest) (*proto.CreateSignedTransactionReply, error)
	CreateAccountSignedTransaction(req *proto.CreateAccountSignedTransactionRequest) (*proto.CreateSignedTransactionReply, error)
	QueryAccountTransactionFromData(req *proto.QueryTransactionFromDataRequest) (*proto.QueryAccountTransactionReply, error)
	QueryAccountTransactionFromSignedData(req *proto.QueryTransactionFromSignedDataRequest) (*proto.QueryAccountTransactionReply, error)
	QueryUtxoTransactionFromData(req *proto.QueryTransactionFromDataRequest) (*proto.QueryUtxoTransactionReply, error)
	QueryUtxoTransactionFromSignedData(req *proto.QueryTransactionFromSignedDataRequest) (*proto.QueryUtxoTransactionReply, error)
	BroadcastTransaction(req *proto.BroadcastTransactionRequest) (*proto.BroadcastTransactionReply, error)
	QueryUtxo(req *proto.QueryUtxoRequest) (*proto.QueryUtxoReply, error)
	QueryUtxoInsFromData(req *proto.QueryUtxoInsFromDataRequest) (*proto.QueryUtxoInsReply, error)
	QueryUtxoTransaction(req *proto.QueryTransactionRequest) (*proto.QueryUtxoTransactionReply, error)
	QueryAccountTransaction(req *proto.QueryTransactionRequest) (*proto.QueryAccountTransactionReply, error)
	VerifyAccountSignedTransaction(req *proto.VerifySignedTransactionRequest) (*proto.VerifySignedTransactionReply, error)
	VerifyUtxoSignedTransaction(req *proto.VerifySignedTransactionRequest) (*proto.VerifySignedTransactionReply, error)
	GetLatestBlockHeight() (int64, error)
	GetAccountTransactionByHeight(height int64, replyCh chan *proto.QueryAccountTransactionReply, errCh chan error)
	GetUtxoTransactionByHeight(height int64, replyCh chan *proto.QueryUtxoTransactionReply, errCh chan error)
}
