package ethereum

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/btcsuite/btcd/btcec"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/shopspring/decimal"
	"go.uber.org/atomic"

	"github.com/hbtc-chain/chainnode/cache"
	"github.com/hbtc-chain/chainnode/chainadaptor"
	"github.com/hbtc-chain/chainnode/chainadaptor/fallback"
	"github.com/hbtc-chain/chainnode/config"
	"github.com/hbtc-chain/chainnode/proto"
)

const (
	ChainName = "eth"
	Symbol    = "eth"
)

type ChainAdaptor struct {
	fallback.ChainAdaptor
	client *ethClient
}

func NewChainAdaptor(conf *config.Config) (chainadaptor.ChainAdaptor, error) {
	client, err := newEthClient(conf)
	if err != nil {
		return nil, err
	}
	return &ChainAdaptor{
		client: client,
	}, nil
}

func NewLocalChainAdaptor(network config.NetWorkType) chainadaptor.ChainAdaptor {
	return newChainAdaptor(newLocalEthClient(network))
}

func newChainAdaptor(client *ethClient) chainadaptor.ChainAdaptor {
	return &ChainAdaptor{
		client: client,
	}
}

// ConvertAddress convert BlueHelix chain's pubkey to a ETH address
func (a *ChainAdaptor) ConvertAddress(req *proto.ConvertAddressRequest) (*proto.ConvertAddressReply, error) {
	publicKey, err := btcec.ParsePubKey(req.PublicKey, btcec.S256())
	if err != nil {
		log.Error(" btcec.ParsePubKey failed", "err", err)
		return &proto.ConvertAddressReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, err
	}

	log.Info("convert req pub to address", "address", crypto.PubkeyToAddress(*publicKey.ToECDSA()).String())

	return &proto.ConvertAddressReply{
		Code:    proto.ReturnCode_SUCCESS,
		Address: crypto.PubkeyToAddress(*publicKey.ToECDSA()).String(),
	}, nil
}

// ValidAddress check address format
func (a *ChainAdaptor) ValidAddress(req *proto.ValidAddressRequest) (*proto.ValidAddressReply, error) {
	valid := common.IsHexAddress(req.Address)
	stdAddr := common.HexToAddress(req.Address)
	log.Info("valid address", "address", req.Address, "valid", valid, "standardAddreess", stdAddr.String())

	isContract := false
	if !a.client.local {
		isContract = a.client.isContractAddress(stdAddr)
	}

	return &proto.ValidAddressReply{
		Code:             proto.ReturnCode_SUCCESS,
		Valid:            valid,
		CanWithdrawal:    !isContract,
		CanonicalAddress: stdAddr.String(),
	}, nil
}

func (a *ChainAdaptor) QueryBalance(req *proto.QueryBalanceRequest) (*proto.QueryBalanceReply, error) {
	key := strings.Join([]string{req.Symbol, req.Address, strconv.FormatUint(req.BlockHeight, 10)}, ":")
	// amount, _ := big.NewInt(0).SetString(req.Amount, 10)
	balanceCache := cache.GetBalanceCache()

	if req.BlockHeight != 0 {
		if r, exist := balanceCache.Get(key); exist {
			return &proto.QueryBalanceReply{
				Code:    proto.ReturnCode_SUCCESS,
				Balance: r.(*big.Int).String(),
			}, nil
		}
	}

	var result *big.Int
	var err error

	if req.BlockHeight == 0 {
		if len(req.ContractAddress) > 0 {
			result, err = a.client.erc20BalanceOf(req.ContractAddress, req.Address, nil)
		} else {
			result, err = a.client.BalanceAt(context.TODO(), common.HexToAddress(req.Address), nil)
		}
		if err != nil {
			log.Error("get balance error", "err", err)
			return &proto.QueryBalanceReply{
				Code: proto.ReturnCode_ERROR,
				Msg:  err.Error(),
			}, err
		}
	} else {
		if len(req.ContractAddress) > 0 {
			result, err = a.client.erc20BalanceOf(req.ContractAddress, req.Address, big.NewInt(int64(req.BlockHeight)))
		} else {
			result, err = a.client.BalanceAt(context.TODO(), common.HexToAddress(req.Address), big.NewInt(int64(req.BlockHeight)))
		}
		if err != nil {
			log.Error("get balance error", "err", err)
			return &proto.QueryBalanceReply{
				Code: proto.ReturnCode_ERROR,
				Msg:  err.Error(),
			}, err
		}
	}

	// cache the result
	balanceCache.Add(key, result)
	return &proto.QueryBalanceReply{
		Code:    proto.ReturnCode_SUCCESS,
		Balance: result.String(),
	}, nil

}

func (a *ChainAdaptor) QueryNonce(req *proto.QueryNonceRequest) (*proto.QueryNonceReply, error) {
	var bockHeight *big.Int
	nonce, err := a.client.NonceAt(context.TODO(), common.HexToAddress(req.Address), bockHeight)
	if err != nil {
		log.Error("get nonce failed", "err", err)
		return &proto.QueryNonceReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, nil
	}

	return &proto.QueryNonceReply{
		Code:  proto.ReturnCode_SUCCESS,
		Nonce: nonce,
	}, nil
}

func (a *ChainAdaptor) QueryGasPrice(_ *proto.QueryGasPriceRequest) (*proto.QueryGasPriceReply, error) {
	price, err := a.client.SuggestGasPrice(context.TODO())
	if err != nil {
		log.Error("get gas price failed", "err", err)
		return &proto.QueryGasPriceReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, nil
	}

	return &proto.QueryGasPriceReply{
		Code:     proto.ReturnCode_SUCCESS,
		GasPrice: price.String(),
	}, nil

}

// QueryTransaction query tx info from chain
func (a *ChainAdaptor) QueryAccountTransaction(req *proto.QueryTransactionRequest) (*proto.QueryAccountTransactionReply, error) {
	key := strings.Join([]string{req.Symbol, req.TxHash}, ":")
	txCache := cache.GetTxCache()
	if r, exist := txCache.Get(key); exist {
		return r.(*proto.QueryAccountTransactionReply), nil
	}

	tx, pending, err := a.client.TransactionByHash(context.TODO(), common.HexToHash(req.TxHash))
	if err != nil {
		if err == ethereum.NotFound {
			return &proto.QueryAccountTransactionReply{
				Code:     proto.ReturnCode_SUCCESS,
				TxStatus: proto.TxStatus_NotFound,
			}, nil
		}
		log.Error("get transaction error", "err", err)
		return &proto.QueryAccountTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, nil
	}

	if pending {
		log.Info("get transaction ", "pending", pending)
		return &proto.QueryAccountTransactionReply{
			Code:     proto.ReturnCode_SUCCESS,
			TxStatus: proto.TxStatus_Pending,
		}, nil
	}

	receipt, err := a.client.TransactionReceipt(context.TODO(), common.HexToHash(req.TxHash))
	if err != nil {
		log.Error("get transaction receipt error", "err", err)
		return &proto.QueryAccountTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, nil
	}

	if receipt == nil {
		log.Error("receipt is nil")
		return &proto.QueryAccountTransactionReply{
			Code:     proto.ReturnCode_SUCCESS,
			TxStatus: proto.TxStatus_Pending,
		}, nil
	}

	log.Info("get transaction", "block_height", receipt.BlockNumber, "block_hash", receipt.BlockHash)

	if receipt.Status == types.ReceiptStatusFailed {
		log.Info("receipt status", "status", receipt.Status)
		return &proto.QueryAccountTransactionReply{
			Code:     proto.ReturnCode_SUCCESS,
			TxStatus: proto.TxStatus_Failed,
		}, nil
	}

	txBlockNumber := receipt.BlockNumber
	if txBlockNumber == nil {
		return &proto.QueryAccountTransactionReply{
			Code:     proto.ReturnCode_SUCCESS,
			TxStatus: proto.TxStatus_Pending,
		}, nil
	}
	blockNumber := a.blockNumber()
	if blockNumber == nil {
		log.Error("get transaction error: invalid latest block number")
		return &proto.QueryAccountTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  "invalid latest block number",
		}, nil
	}
	log.Info("get transaction for confirmations", "block", blockNumber.String(),
		"txBlockNumber", txBlockNumber.String(),
		"confirmations", a.client.confirmations)
	if big.NewInt(0).Sub(blockNumber, txBlockNumber).Int64() < int64(a.client.confirmations) {
		log.Info("get transaction ", "pending", pending)
		return &proto.QueryAccountTransactionReply{
			Code:     proto.ReturnCode_SUCCESS,
			TxStatus: proto.TxStatus_Pending,
		}, nil
	}

	signer, err := a.makeSigner()
	if err != nil {
		return nil, err
	}

	reply, err := a.queryTransaction(req.Chain != req.Symbol, tx, receipt, receipt.BlockNumber.Uint64(), signer)
	if err != nil {
		return nil, err
	}
	if reply.TxStatus == proto.TxStatus_Success {
		txCache.Add(key, reply)
	}

	return reply, err
}

// QueryTransactionFromSignedData query tx info from a signed transaction
func (a *ChainAdaptor) QueryAccountTransactionFromSignedData(req *proto.QueryTransactionFromSignedDataRequest) (*proto.QueryAccountTransactionReply, error) {
	signedTx := new(types.Transaction)
	if err := rlp.DecodeBytes(req.SignedTxData, signedTx); err != nil {
		log.Error("signedTx unmarlshal failed", "err", err)
		return &proto.QueryAccountTransactionReply{
			Code: proto.ReturnCode_SUCCESS,
			Msg:  err.Error(),
		}, err
	}

	return a.queryTransaction(req.Chain != req.Symbol, signedTx, nil, 0, a.makeSignerOffline(req.Height))
}

// QueryTransactionFromData query tx info from a raw(unsigned) transaction
func (a *ChainAdaptor) QueryAccountTransactionFromData(req *proto.QueryTransactionFromDataRequest) (*proto.QueryAccountTransactionReply, error) {
	rawTx := new(types.Transaction)
	if err := rlp.DecodeBytes(req.RawData, rawTx); err != nil {
		log.Error("signedTx unmarlshal failed", "err", err)
		return &proto.QueryAccountTransactionReply{
			Code: proto.ReturnCode_SUCCESS,
			Msg:  err.Error(),
		}, err
	}

	reply, err := a.queryRawTransaction(req.Chain != req.Symbol, rawTx)
	if err != nil {
		log.Error("queryRawTransaction failed", "err", err)
		return &proto.QueryAccountTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, nil
	}
	signer := a.makeSignerOffline(req.Height)
	reply.SignHash = signer.Hash(rawTx).Bytes()
	return reply, nil
}

// CreateTransaction make a transaction without signature
func (a *ChainAdaptor) CreateAccountTransaction(req *proto.CreateAccountTransactionRequest) (*proto.CreateAccountTransactionReply, error) {
	if !common.IsHexAddress(req.From) {
		log.Info("invalid from address", "from", req.From)
		return &proto.CreateAccountTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  "invalid from address",
		}, nil
	}

	if !common.IsHexAddress(req.To) {
		log.Info("invalid to address", "to", req.To)
		return &proto.CreateAccountTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  "invalid to address",
		}, nil
	}

	if req.Amount == "" {
		log.Info("amount can not be zero", "amount", req.Amount)
		return &proto.CreateAccountTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  "zero amount",
		}, nil
	}

	if req.GasLimit == "" {
		log.Info("gas uints can not be zero", "gas_unit", req.GasLimit)
		return &proto.CreateAccountTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  "zero gasunits",
		}, nil
	}

	if req.GasPrice == "" {
		log.Info("gas price can not be zero", "gas_price", req.GasPrice)
		return &proto.CreateAccountTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  "zero gasprice",
		}, nil
	}

	// get nonce from chain
	nonce := req.Nonce
	// calc amount
	assetAmount := stringToInt(req.Amount)
	if assetAmount == nil {
		log.Error("convert asset amount failed")
		return &proto.CreateAccountTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  "convert asset amount failed",
		}, nil
	}

	// convert gas price
	gasPrice := stringToInt(req.GasPrice)
	if gasPrice == nil {
		log.Error("convert gasPrice failed")
		return &proto.CreateAccountTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  "convert gasPrice failed",
		}, nil
	}

	gasLimit := stringToInt(req.GasLimit)
	if gasLimit == nil {
		log.Error("convert gasLimit failed")
		return &proto.CreateAccountTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  "convert gasLimit failed",
		}, nil
	}

	var err error
	// make transaction
	var tx *types.Transaction
	if len(req.ContractAddress) > 0 {
		tx, err = a.client.erc20RawTransfer(req.ContractAddress, nonce, common.HexToAddress(req.To), assetAmount,
			gasLimit.Uint64(), gasPrice)
		if err != nil {
			log.Error("ERC20 tx raw transfer failed", "err", err)
			return nil, err
		}
	} else {
		tx = types.NewTransaction(nonce, common.HexToAddress(req.To), assetAmount, gasLimit.Uint64(), gasPrice, nil)
	}
	txData, err := rlp.EncodeToBytes(tx)
	if err != nil {
		log.Error("tx EncodeToBytes failed", "err", err)
		return &proto.CreateAccountTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, err
	}

	signer, err := a.makeSigner()
	if err != nil {
		return nil, err
	}

	return &proto.CreateAccountTransactionReply{
		Code:     proto.ReturnCode_SUCCESS,
		TxData:   txData,
		SignHash: signer.Hash(tx).Bytes(),
	}, nil
}

// CreateSignedTransaction create signed transaction
func (a *ChainAdaptor) CreateAccountSignedTransaction(req *proto.CreateAccountSignedTransactionRequest) (*proto.CreateSignedTransactionReply, error) {
	tx := new(types.Transaction)
	if err := rlp.DecodeBytes(req.TxData, tx); err != nil {
		log.Error("tx unmarlshal failed", "err", err)
		return &proto.CreateSignedTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, err
	}

	signer, err := a.makeSigner()
	if err != nil {
		return nil, err
	}

	signedTx, err := tx.WithSignature(signer, req.Signature)
	if err != nil {
		log.Error("tx WithSignature failed", "err", err)
		return &proto.CreateSignedTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, err
	}

	signedTxData, err := rlp.EncodeToBytes(signedTx)
	if err != nil {
		log.Error("signedTx EncodeToBytes failed", "err", err)
		return &proto.CreateSignedTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, err
	}

	return &proto.CreateSignedTransactionReply{
		Code:         proto.ReturnCode_SUCCESS,
		SignedTxData: signedTxData,
		Hash:         signedTx.Hash().Bytes(),
	}, nil
}

// BroadcastTransaction  broadcast tx to chain
func (a *ChainAdaptor) BroadcastTransaction(req *proto.BroadcastTransactionRequest) (*proto.BroadcastTransactionReply, error) {
	signedTx := new(types.Transaction)
	if err := rlp.DecodeBytes(req.SignedTxData, signedTx); err != nil {
		log.Error("signedTx DecodeBytes failed", "err", err)
		return &proto.BroadcastTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, err
	}
	log.Info("broadcast tx", "tx", hexutil.Encode(req.SignedTxData))

	txHash := fmt.Sprintf("0x%x", signedTx.Hash())
	if err := a.client.SendTransaction(context.TODO(), signedTx); err != nil {
		log.Error("braoadcast tx failed", "tx_hash", txHash, "err", err)
		return &proto.BroadcastTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, err
	}

	log.Info("braoadcast tx success", "tx_hash", txHash)
	return &proto.BroadcastTransactionReply{
		Code:   proto.ReturnCode_SUCCESS,
		TxHash: txHash,
	}, nil
}

func (a *ChainAdaptor) VerifyAccountSignedTransaction(req *proto.VerifySignedTransactionRequest) (*proto.VerifySignedTransactionReply, error) {
	signedTx := new(types.Transaction)
	if err := rlp.DecodeBytes(req.SignedTxData, signedTx); err != nil {
		log.Error("signedTx DecodeBytess failed", "err", err)
		return &proto.VerifySignedTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, err
	}

	signer := a.makeSignerOffline(req.Height)

	sender, err := signer.Sender(signedTx)
	if err != nil {
		log.Error("failed to get sender from signed tx", "err", err)
		return &proto.VerifySignedTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, err
	}

	var expectedSender string
	if len(req.Addresses) > 0 {
		expectedSender = req.Addresses[0]
	} else {
		expectedSender = req.Sender
	}

	return &proto.VerifySignedTransactionReply{
		Code:     proto.ReturnCode_SUCCESS,
		Verified: sender == common.HexToAddress(expectedSender),
	}, err
}

func (a *ChainAdaptor) GetLatestBlockHeight() (int64, error) {
	number, err := a.client.BlockByNumber(context.TODO(), nil)
	if err != nil {
		return 0, err
	}
	return number.Number().Int64(), err
}

func (a *ChainAdaptor) GetAccountTransactionByHeight(height int64, replyCh chan *proto.QueryAccountTransactionReply, errCh chan error) {
	block, err := a.client.BlockByNumber(context.TODO(), big.NewInt(height))
	if err != nil {
		errCh <- err
		return
	}

	transactions := block.Transactions()

	var wg sync.WaitGroup
	wg.Add(len(transactions))
	sem := make(semaphore, runtime.NumCPU())
	var needStop atomic.Bool

	for _, tx := range transactions {
		tx := tx
		sem.Acquire()

		go func() {
			defer sem.Release()
			defer wg.Done()

			if needStop.Load() {
				return
			}

			receipt, err := a.client.TransactionReceipt(context.TODO(), tx.Hash())
			if err != nil {
				errCh <- err
				needStop.Store(true)
				return
			}

			if receipt.Status == types.ReceiptStatusFailed {
				return
			}
			toAddress := tx.To()
			if toAddress == nil {
				return
			}
			signer := a.makeSignerOffline(height)
			sender, err := signer.Sender(tx)
			if err != nil {
				errCh <- err
				needStop.Store(true)
				return
			}
			costFee := new(big.Int).Mul(new(big.Int).SetUint64(receipt.GasUsed), tx.GasPrice())

			if tx.Value().Cmp(big.NewInt(0)) == 1 {
				replyCh <- &proto.QueryAccountTransactionReply{
					TxHash:          tx.Hash().String(),
					TxStatus:        proto.TxStatus_Success,
					From:            sender.String(),
					To:              toAddress.String(),
					Amount:          tx.Value().String(),
					Memo:            "",
					Nonce:           tx.Nonce(),
					GasLimit:        new(big.Int).SetUint64(tx.Gas()).String(),
					GasPrice:        tx.GasPrice().String(),
					CostFee:         costFee.String(),
					BlockHeight:     uint64(height),
					BlockTime:       block.Time(),
					SignHash:        signer.Hash(tx).Bytes(),
					ContractAddress: "",
				}
			}
			for _, receiptLog := range receipt.Logs {
				if receiptLog.Removed {
					continue
				}
				if len(receiptLog.Topics) != 3 {
					continue
				}
				if receiptLog.Topics[0] != common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef") {
					continue
				}

				tokenFromAddress := common.BytesToAddress(receiptLog.Topics[1].Bytes())
				tokenToAddress := common.BytesToAddress(receiptLog.Topics[2].Bytes())
				tokenAmount, ok := big.NewInt(0).SetString(fmt.Sprintf("%x", receiptLog.Data), 16)
				if !ok {
					errCh <- errors.New("failed to decode token amount from receipt log data")
					needStop.Store(true)
					return
				}
				if tokenAmount.Cmp(big.NewInt(0)) == 1 {
					replyCh <- &proto.QueryAccountTransactionReply{
						TxHash:          tx.Hash().String(),
						TxStatus:        proto.TxStatus_Success,
						From:            tokenFromAddress.String(),
						To:              tokenToAddress.String(),
						Amount:          tokenAmount.String(),
						Memo:            "",
						Nonce:           tx.Nonce(),
						GasLimit:        new(big.Int).SetUint64(tx.Gas()).String(),
						GasPrice:        tx.GasPrice().String(),
						CostFee:         costFee.String(),
						BlockHeight:     uint64(height),
						BlockTime:       block.Time(),
						SignHash:        signer.Hash(tx).Bytes(),
						ContractAddress: receiptLog.Address.String(),
					}
				}

			}

		}()
	}

	wg.Wait()
}

// stringToInt convert string amount to big.Int
func stringToInt(amount string) *big.Int {
	log.Info("string to Int", "amount", amount)
	intAmount, success := big.NewInt(0).SetString(amount, 0)
	if !success {
		return nil
	}
	return intAmount
}

// isContractAddress check the address is a contract address or not

func (a *ChainAdaptor) blockNumber() *big.Int {
	return a.client.blockNumber()
}

func (a *ChainAdaptor) makeSigner() (types.Signer, error) {
	height := a.blockNumber()
	if height == nil {
		err := fmt.Errorf("fail to get height in making signer")
		return nil, err
	}
	log.Info("make signer", "height", height.Uint64())
	return types.MakeSigner(a.client.chainConfig, height), nil
}

func (a *ChainAdaptor) makeSignerOffline(height int64) types.Signer {
	if height == 0 {
		height = math.MaxInt64
	}
	return types.MakeSigner(a.client.chainConfig, big.NewInt(height))
}

// queryTransaction retrieve transaction information from a signed data.
func (a *ChainAdaptor) queryTransaction(isERC20 bool, tx *types.Transaction, receipt *types.Receipt, blockNumber uint64, signer types.Signer) (*proto.QueryAccountTransactionReply, error) {
	reply, err := a.queryRawTransaction(isERC20, tx)
	if err != nil {
		return &proto.QueryAccountTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, nil
	}
	reply.SignHash = signer.Hash(tx).Bytes()
	msg, err := tx.AsMessage(signer)
	if err != nil {
		log.Error("tx as message err", "err", err)
		return &proto.QueryAccountTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, nil
	}

	gasUsed := new(big.Int)
	if receipt != nil {
		gasUsed = gasUsed.SetUint64(receipt.GasUsed).Mul(gasUsed, tx.GasPrice())
		if isERC20 {
			// Check ERC20 Transfer event log
			err := a.validateAndQueryERC20TransferReceipt(common.HexToAddress(reply.ContractAddress),
				msg.From().String(), reply.To, reply.Amount, receipt)
			if err != nil {
				return &proto.QueryAccountTransactionReply{
					Code: proto.ReturnCode_ERROR,
					Msg:  err.Error(),
				}, nil
			}
		}
	}

	log.Info("QueryTransaction", "from", msg.From().String(),
		"block_number", blockNumber,
		"gas_used", decimal.NewFromBigInt(gasUsed, 0).String())

	reply.From = msg.From().String()
	reply.CostFee = decimal.NewFromBigInt(gasUsed, 0).String()
	reply.BlockHeight = blockNumber
	reply.TxStatus = proto.TxStatus_Success
	reply.TxHash = tx.Hash().String()

	return reply, nil
}

// queryRawTransaction retrieve transaction information from a raw(unsigned) data.
func (a *ChainAdaptor) queryRawTransaction(isERC20 bool, rawTx *types.Transaction) (*proto.QueryAccountTransactionReply, error) {
	var amount *big.Int
	var to common.Address
	contractAddress := ""
	var err error
	if isERC20 {
		// erc20 transfer transaction
		to, amount, err = a.validateAndQueryERC20RawTransfer(*rawTx.To(), rawTx)
		if err != nil {
			return nil, err
		}
		contractAddress = rawTx.To().String()
	} else {
		// ether transaction
		to = *rawTx.To()
		amount = rawTx.Value()
	}
	log.Info("QueryRawTransaction",
		"is_erc20", isERC20,
		"to", to.String(),
		"nonce", rawTx.Nonce(),
		"amount", decimal.NewFromBigInt(amount, 0).String(),
		"gas_limit", decimal.NewFromBigInt(big.NewInt(int64(rawTx.Gas())), 0).String(),
		"gas_price", decimal.NewFromBigInt(rawTx.GasPrice(), 0).String())

	return &proto.QueryAccountTransactionReply{
		Code:            proto.ReturnCode_SUCCESS,
		To:              to.String(),
		Nonce:           rawTx.Nonce(),
		Amount:          decimal.NewFromBigInt(amount, 0).String(),
		GasLimit:        decimal.NewFromBigInt(big.NewInt(int64(rawTx.Gas())), 0).String(),
		GasPrice:        decimal.NewFromBigInt(rawTx.GasPrice(), 0).String(),
		ContractAddress: contractAddress,
	}, nil
}

type semaphore chan struct{}

func (s semaphore) Acquire() {
	s <- struct{}{}
}

func (s semaphore) Release() {
	<-s
}
