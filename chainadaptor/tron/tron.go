package tron

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"go.uber.org/atomic"

	"github.com/btcsuite/btcd/btcec"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	pb "github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/hbtc-chain/chainnode/cache"
	"github.com/hbtc-chain/chainnode/chainadaptor"
	"github.com/hbtc-chain/chainnode/chainadaptor/fallback"
	"github.com/hbtc-chain/chainnode/config"
	"github.com/hbtc-chain/chainnode/proto"
	"github.com/hbtc-chain/gotron-sdk/pkg/address"
	"github.com/hbtc-chain/gotron-sdk/pkg/proto/api"
	"github.com/hbtc-chain/gotron-sdk/pkg/proto/core"
)

const TrxDecimals = 6

const (
	ChainName               = "trx"
	TronSymbol              = "trx"
	MaxTimeUntillExpiration = 24*60*60*1000 - 120000 //23hour58min, MaxTimeUntillExpiration is 24 hours in Tron
)
const (
	trc20TransferTopicLen        = 3
	trc20TransferTopic           = "ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"
	trc20TransferAddrLen         = 32
	trc20TransferMethodSignature = "a9059cbb"
	defultGasLimit               = 1000000 //use 1 trx as limit.
)

type ChainAdaptor struct {
	fallback.ChainAdaptor
	client *tronClient
}

func NewChainAdaptor(conf *config.Config) (chainadaptor.ChainAdaptor, error) {
	client, err := newTronClient(conf)
	if err != nil {
		return nil, err
	}
	return &ChainAdaptor{
		client: client,
	}, nil
}

func NewLocalChainAdaptor(network config.NetWorkType) chainadaptor.ChainAdaptor {
	return newChainAdaptor(newLocalTronClient(network))
}

func newChainAdaptor(client *tronClient) chainadaptor.ChainAdaptor {
	return &ChainAdaptor{
		client: client,
	}
}

// ConvertAddress convert BlueHelix chain's pubkey to a TRON address, keygen will generate compressed pubkey, tron only support uncompressed key like eth.
func (a *ChainAdaptor) ConvertAddress(req *proto.ConvertAddressRequest) (*proto.ConvertAddressReply, error) {
	log.Info("ConvertAddress", "req", req)
	btcecPubKey, err := btcec.ParsePubKey(req.PublicKey, btcec.S256())
	if err != nil {
		log.Error("btcec.ParsePubKey failed", "err", err)
		return &proto.ConvertAddressReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, err
	}
	addr := address.PubkeyToAddress(*btcecPubKey.ToECDSA()).String()

	log.Debug("ConvertAddress result", "pub", hex.EncodeToString(req.PublicKey), "address", addr)
	return &proto.ConvertAddressReply{
		Code:    proto.ReturnCode_SUCCESS,
		Address: addr,
	}, nil
}

/**/
func (a *ChainAdaptor) ValidAddress(req *proto.ValidAddressRequest) (*proto.ValidAddressReply, error) {
	log.Info("ValidAddress", "req", req)

	ok := strings.HasPrefix(req.Address, "T")
	grpcClient := a.client.grpcClient
	//a TRC10 address
	if !ok {
		if !a.client.local {
			txi, err := grpcClient.GetAssetIssueByID(req.Address)
			if err != nil {
				log.Error("invalid TRC10 issuer", "err", err)
				return &proto.ValidAddressReply{
					Code:  proto.ReturnCode_ERROR,
					Msg:   err.Error(),
					Valid: false,
				}, err
			}

			if txi.Id != req.Address {
				err := fmt.Errorf("unmatched TRC10 issuer:%v", req.Address)
				log.Error(err.Error())
				return &proto.ValidAddressReply{
					Code:  proto.ReturnCode_ERROR,
					Msg:   err.Error(),
					Valid: false,
				}, err
			}
		}

		//a valid TRC10 contract address
		return &proto.ValidAddressReply{
			Code:             proto.ReturnCode_SUCCESS,
			Valid:            true,
			CanWithdrawal:    false,
			CanonicalAddress: req.Address,
		}, nil

	}

	//TRC20 address or normal address
	addr, err := address.Base58ToAddress(req.Address)
	if err != nil {
		log.Error("Base58ToAddress failed", "err", err)
		return &proto.ValidAddressReply{
			Code:  proto.ReturnCode_ERROR,
			Msg:   err.Error(),
			Valid: false,
		}, err
	}

	if !addr.IsValid() {
		err := fmt.Errorf("%v not a valid address", addr)
		log.Error(err.Error())
		return &proto.ValidAddressReply{
			Code:  proto.ReturnCode_ERROR,
			Valid: false,
			Msg:   err.Error(),
		}, err
	}

	isTrc20 := false
	if !a.client.local {
		abi, err := grpcClient.GetContractABI(req.Address)
		if err != nil {
			return &proto.ValidAddressReply{
				Code:  proto.ReturnCode_ERROR,
				Valid: false,
				Msg:   err.Error(),
			}, err
		}

		if abi != nil {
			isTrc20 = true
		}
	}

	return &proto.ValidAddressReply{
		Code:             proto.ReturnCode_SUCCESS,
		Valid:            true,
		CanWithdrawal:    !isTrc20,
		CanonicalAddress: req.Address,
	}, nil
}

func (a *ChainAdaptor) QueryBalance(req *proto.QueryBalanceRequest) (*proto.QueryBalanceReply, error) {
	log.Info("QueryBalance", "req", req)
	key := strings.Join([]string{req.Symbol, req.Address, strconv.FormatUint(req.BlockHeight, 10)}, ":")
	balanceCache := cache.GetBalanceCache()

	grpcClient := a.client.grpcClient
	if req.BlockHeight != 0 {
		if r, exist := balanceCache.Get(key); exist {
			return &proto.QueryBalanceReply{
				Code:    proto.ReturnCode_SUCCESS,
				Balance: r.(*big.Int).String(),
			}, nil
		}
	}

	var result *big.Int
	if req.ContractAddress != "" {
		//TRC20, verify symbol
		symbol, err := grpcClient.TRC20GetSymbol(req.ContractAddress)
		if err != nil {
			return &proto.QueryBalanceReply{
				Code: proto.ReturnCode_ERROR,
				Msg:  err.Error(),
			}, err
		}

		if symbol != req.Symbol {
			err = fmt.Errorf("contract's symbol %v != symbol:%v", symbol, req.Symbol)
			return &proto.QueryBalanceReply{
				Code: proto.ReturnCode_ERROR,
				Msg:  err.Error(),
			}, err
		}

		result, err = grpcClient.TRC20ContractBalance(req.Address, req.ContractAddress)
		if err != nil {
			return &proto.QueryBalanceReply{
				Code: proto.ReturnCode_ERROR,
				Msg:  err.Error(),
			}, err
		}
	} else {
		acc, err := grpcClient.GetAccount(req.Address)
		if err != nil {
			return &proto.QueryBalanceReply{
				Code: proto.ReturnCode_ERROR,
				Msg:  err.Error(),
			}, nil
		}

		if req.Symbol == TronSymbol {
			//TRX
			result = big.NewInt(acc.Balance)
		} else {
			//TRC10
			if r, exist := acc.AssetV2[req.Symbol]; !exist {
				result = big.NewInt(0)
			} else {
				result = big.NewInt(r)
			}
		}
	}

	balanceCache.Add(key, result)
	return &proto.QueryBalanceReply{
		Code:    proto.ReturnCode_SUCCESS,
		Balance: result.String(),
	}, nil
}

func (a *ChainAdaptor) QueryNonce(req *proto.QueryNonceRequest) (*proto.QueryNonceReply, error) {
	log.Info("QueryNonce", "req", req)
	return &proto.QueryNonceReply{
		Code:  proto.ReturnCode_SUCCESS,
		Nonce: 0,
	}, nil
}

func (a *ChainAdaptor) QueryGasPrice(req *proto.QueryGasPriceRequest) (*proto.QueryGasPriceReply, error) {
	log.Info("QueryGasPrice", "req", req)
	return &proto.QueryGasPriceReply{
		Code:     proto.ReturnCode_SUCCESS,
		GasPrice: "1",
	}, nil
}

func (a *ChainAdaptor) QueryAccountTransaction(req *proto.QueryTransactionRequest) (*proto.QueryAccountTransactionReply, error) {
	log.Info("QueryTransaction", "req", req)
	grpcClient := a.client.grpcClient

	tx, err := grpcClient.GetTransactionByID(req.TxHash)
	if err != nil {
		return &proto.QueryAccountTransactionReply{
			Code:     proto.ReturnCode_SUCCESS,
			Msg:      err.Error(),
			TxStatus: proto.TxStatus_NotFound,
		}, nil
	}

	r := tx.RawData.Contract
	if len(r) != 1 {
		err = fmt.Errorf("QueryAccountTransaction, unsupport tx %v", req.TxHash)
		return &proto.QueryAccountTransactionReply{
			Code:     proto.ReturnCode_ERROR,
			Msg:      err.Error(),
			TxStatus: proto.TxStatus_Other,
		}, nil
	}

	txi, err := grpcClient.GetTransactionInfoByID(req.TxHash)
	if err != nil {
		return &proto.QueryAccountTransactionReply{
			Code:     proto.ReturnCode_ERROR,
			Msg:      err.Error(),
			TxStatus: proto.TxStatus_NotFound,
		}, nil
	}

	var depositList []depositInfo
	switch r[0].Type {
	case core.Transaction_Contract_TransferContract:
		depositList, err = decodeTransferContract(r[0], req.TxHash)
		if err != nil {
			return &proto.QueryAccountTransactionReply{
				Code: proto.ReturnCode_ERROR,
				Msg:  err.Error(),
			}, nil
		}

	case core.Transaction_Contract_TransferAssetContract:
		depositList, err = decodeTransferAssetContract(r[0], req.TxHash)
		if err != nil {
			return &proto.QueryAccountTransactionReply{
				Code: proto.ReturnCode_ERROR,
				Msg:  err.Error(),
			}, nil
		}

	case core.Transaction_Contract_TriggerSmartContract:
		depositList, err = decodeTriggerSmartContract(r[0], txi, req.TxHash)
		if err != nil {
			return &proto.QueryAccountTransactionReply{
				Code: proto.ReturnCode_ERROR,
				Msg:  err.Error(),
			}, nil
		}
	default:
		err = fmt.Errorf("QueryTransaction, unsupport contract type %v, tx hash %v ", r[0].Type, req.TxHash)
		log.Info(err.Error())
		return &proto.QueryAccountTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, nil
	}

	//Note: decodeTriggerSmartContract supports multi TRC20 transfer in single hash,  but assume we will initiate single TRC20 transfer
	// in single hash, QueryAccountTransaction is supposed to query self-initiated transaction
	if len(depositList) > 1 {
		err = fmt.Errorf("QueryTransaction, more than 1 deposit list %v, tx hash %v ", len(depositList), req.TxHash)
		log.Info(err.Error())
		return &proto.QueryAccountTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, nil
	}

	var txStatus proto.TxStatus
	switch txi.Result {
	case core.TransactionInfo_SUCESS:
		txStatus = proto.TxStatus_Success

	case core.TransactionInfo_FAILED:
		txStatus = proto.TxStatus_Failed
	}

	if len(depositList) == 0 {
		return &proto.QueryAccountTransactionReply{
			Code:        proto.ReturnCode_SUCCESS,
			TxHash:      req.TxHash,
			TxStatus:    txStatus,
			Memo:        "",
			Nonce:       0,
			BlockHeight: uint64(txi.BlockNumber),
			BlockTime:   uint64(txi.BlockTimeStamp),
			GasPrice:    "1",
			GasLimit:    big.NewInt(tx.RawData.GetFeeLimit()).String(),
			CostFee:     big.NewInt(txi.GetFee()).String(),
		}, nil
	} else {
		return &proto.QueryAccountTransactionReply{
			Code:            proto.ReturnCode_SUCCESS,
			TxHash:          req.TxHash,
			TxStatus:        txStatus,
			From:            depositList[0].fromAddr,
			To:              depositList[0].toAddr,
			Amount:          depositList[0].amount,
			Memo:            "",
			Nonce:           0,
			BlockHeight:     uint64(txi.BlockNumber),
			BlockTime:       uint64(txi.BlockTimeStamp),
			GasPrice:        "1",
			GasLimit:        big.NewInt(tx.RawData.GetFeeLimit()).String(),
			CostFee:         big.NewInt(txi.GetFee()).String(),
			ContractAddress: depositList[0].contractAddr,
		}, nil
	}
}

func (a *ChainAdaptor) QueryAccountTransactionFromData(req *proto.QueryTransactionFromDataRequest) (*proto.QueryAccountTransactionReply, error) {
	log.Info("QueryAccountTransactionFromData", "req", req)
	var tx core.TransactionRaw

	err := pb.Unmarshal(req.RawData, &tx)
	if err != nil {
		return &proto.QueryAccountTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, nil
	}

	return queryTransactionLocal(&tx, req.Symbol)
}

func (a *ChainAdaptor) QueryAccountTransactionFromSignedData(req *proto.QueryTransactionFromSignedDataRequest) (*proto.QueryAccountTransactionReply, error) {
	log.Info("QueryTransactionFromSignedData", "req", req)
	var tx core.Transaction

	err := pb.Unmarshal(req.SignedTxData, &tx)
	if err != nil {
		return &proto.QueryAccountTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, nil
	}

	return queryTransactionLocal(tx.GetRawData(), req.Symbol)
}

func (a *ChainAdaptor) CreateAccountTransaction(req *proto.CreateAccountTransactionRequest) (*proto.CreateAccountTransactionReply, error) {
	log.Info("CreateTransaction", "req", req)
	grpcClient := a.client.grpcClient
	amount, ok := big.NewInt(0).SetString(req.Amount, 10)
	if !ok {
		return &proto.CreateAccountTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  "invalid amount",
		}, nil
	}

	gas, err := stringToInt64(req.GasLimit)
	if err != nil {
		return &proto.CreateAccountTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, nil
	}

	var txe *api.TransactionExtention
	if req.Symbol == TronSymbol {
		txe, err = grpcClient.Transfer(req.From, req.To, amount.Int64())
		if err != nil {
			return &proto.CreateAccountTransactionReply{
				Code: proto.ReturnCode_ERROR,
				Msg:  err.Error(),
			}, nil
		}
	} else {
		//TRC10/TRC20 both should have issuer,TRC10's issuer = "1000315", TRC20's issuer = contractAddress
		isTrc10 := false
		if req.ContractAddress == "" {
			err := fmt.Errorf(" trc10 or trc20 token %v without issuer", req.Symbol)
			return &proto.CreateAccountTransactionReply{
				Code: proto.ReturnCode_ERROR,
				Msg:  err.Error(),
			}, nil
		}

		_, err := address.Base58ToAddress(req.ContractAddress)
		if err != nil {
			isTrc10 = true
		}

		if isTrc10 {
			// for TRC10, symbol is sign in hbtc chain, contractadress is the sign in tron chain
			txe, err = grpcClient.TransferAsset(req.From, req.To, req.ContractAddress, amount.Int64())
			if err != nil {
				return &proto.CreateAccountTransactionReply{
					Code: proto.ReturnCode_ERROR,
					Msg:  err.Error(),
				}, nil
			}
		} else {
			txe, err = grpcClient.TRC20Send(req.From, req.To, req.ContractAddress, amount, gas)
			if err != nil {
				return &proto.CreateAccountTransactionReply{
					Code: proto.ReturnCode_ERROR,
					Msg:  err.Error(),
				}, nil
			}
		}
	}

	//update expiration and recalculate  hash
	txe.Transaction.RawData.Expiration = txe.Transaction.RawData.Timestamp + MaxTimeUntillExpiration
	txr, err := pb.Marshal(txe.GetTransaction().GetRawData())
	if err != nil {
		return &proto.CreateAccountTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, nil
	}

	hash := getHash(txr)
	txe.Txid = hash

	return &proto.CreateAccountTransactionReply{
		Code:     proto.ReturnCode_SUCCESS,
		TxData:   txr,
		SignHash: hash,
	}, nil
}

func (a *ChainAdaptor) CreateAccountSignedTransaction(req *proto.CreateAccountSignedTransactionRequest) (*proto.CreateSignedTransactionReply, error) {
	log.Info("CreateAccountSignedTransaction", "chain", req.Chain, "txData", hex.EncodeToString(req.TxData), "sig", hex.EncodeToString(req.Signature), "sig's len", len(req.Signature), "pubkey", hex.EncodeToString(req.PublicKey))
	rawData := req.TxData
	hash := getHash(rawData)

	//verify signature check R|S, omit V
	if ok := crypto.VerifySignature(req.PublicKey, hash, req.Signature[:64]); !ok {
		err := fmt.Errorf("fail to verify signature, chain:%v txdata:%v, signature:%v, pubkey:%v", req.Chain, hex.EncodeToString(req.TxData), hex.EncodeToString(req.Signature), hex.EncodeToString(req.PublicKey))
		log.Info(err.Error())
		return &proto.CreateSignedTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, err
	}

	var txRaw core.TransactionRaw
	var tx core.Transaction

	err := pb.Unmarshal(req.TxData, &txRaw)
	if err != nil {
		return &proto.CreateSignedTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, err
	}

	tx.RawData = &txRaw
	tx.Signature = append(tx.Signature, req.Signature)

	bz, err := pb.Marshal(&tx)
	if err != nil {
		return &proto.CreateSignedTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, err
	}

	return &proto.CreateSignedTransactionReply{
		Code:         proto.ReturnCode_SUCCESS,
		SignedTxData: bz,
		Hash:         hash,
	}, nil
}

func (a *ChainAdaptor) VerifyAccountSignedTransaction(req *proto.VerifySignedTransactionRequest) (*proto.VerifySignedTransactionReply, error) {
	log.Error("VerifySignedTransaction", "chain", req.Chain, "signTxData", hex.EncodeToString(req.SignedTxData), "sender", req.Sender)
	var tx core.Transaction
	err := pb.Unmarshal(req.SignedTxData, &tx)
	if err != nil {
		return &proto.VerifySignedTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, err
	}
	rawData, err := pb.Marshal(tx.GetRawData())
	if err != nil {
		return &proto.VerifySignedTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, err
	}

	hash := getHash(rawData)

	if len(tx.Signature) != 1 {
		err := fmt.Errorf("VerifySignedTransaction, len(tx.Signature) != 1")
		log.Error(err.Error())
		return &proto.VerifySignedTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, err
	}

	pubKey, err := crypto.SigToPub(hash, tx.Signature[0])
	if err != nil {
		msg := fmt.Sprintf("SigToPub error, hash:%v, signature:%v, pubKey:%v", hex.EncodeToString(hash), hex.EncodeToString(tx.Signature[0]), hex.EncodeToString(crypto.CompressPubkey(pubKey)))
		log.Error(msg)
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

	addr := address.PubkeyToAddress(*pubKey)
	return &proto.VerifySignedTransactionReply{
		Code:     proto.ReturnCode_SUCCESS,
		Verified: addr.String() == expectedSender,
	}, nil
}

func (a *ChainAdaptor) BroadcastTransaction(req *proto.BroadcastTransactionRequest) (*proto.BroadcastTransactionReply, error) {
	log.Info("BroadcastTransaction", "req", req)
	var tx core.Transaction
	err := pb.Unmarshal(req.SignedTxData, &tx)
	if err != nil {
		return &proto.BroadcastTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, nil
	}

	rawData, err := pb.Marshal(tx.GetRawData())
	hash := getHash(rawData)

	_, err = a.client.grpcClient.Broadcast(&tx)
	if err != nil {
		log.Error("broadcast tx failed", "hash", hex.EncodeToString(hash), "err", err)
		return &proto.BroadcastTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, err
	}

	log.Info("broadcast tx success", "hash", hex.EncodeToString(hash))
	return &proto.BroadcastTransactionReply{
		Code:   proto.ReturnCode_SUCCESS,
		TxHash: hex.EncodeToString(hash),
	}, nil
}

func (a *ChainAdaptor) GetLatestBlockHeight() (int64, error) {
	res, err := a.client.grpcClient.GetNowBlock()
	if err != nil {
		return 0, err
	}
	return res.GetBlockHeader().GetRawData().GetNumber(), nil
}

func (a *ChainAdaptor) GetAccountTransactionByHeight(height int64, replyCh chan *proto.QueryAccountTransactionReply, errCh chan error) {
	grpcClient := a.client.grpcClient
	block, err := grpcClient.GetBlockByNum(height)
	if err != nil {
		errCh <- err
		return
	}

	txExts := block.GetTransactions()

	var wg sync.WaitGroup
	wg.Add(len(txExts))
	sem := make(semaphore, runtime.NumCPU())
	var needStop atomic.Bool

	for _, txExt := range txExts {
		txExt := txExt
		sem.Acquire()

		go func() {
			defer sem.Release()
			defer wg.Done()

			if needStop.Load() {
				return
			}

			hash := hex.EncodeToString(txExt.GetTxid())
			tx := txExt.GetTransaction()

			r := tx.RawData.Contract
			if len(r) != 1 {
				err = fmt.Errorf("GetAccountTransactionByHeight, unsupport tx %v", hash)
				return
			}

			txi, err := grpcClient.GetTransactionInfoByID(hash)
			if err != nil {
				errCh <- err
				needStop.Store(true)
				return
			}

			var txStatus proto.TxStatus
			if txi.Result == core.TransactionInfo_SUCESS {
				txStatus = proto.TxStatus_Success
			} else {
				//	log.Info("ignore failed tx", "hash", hash, "result", txi.Result)
				return
			}

			var depositList []depositInfo
			switch r[0].Type {
			case core.Transaction_Contract_TransferContract:
				depositList, err = decodeTransferContract(r[0], hash)
				if err != nil {
					errCh <- err
					needStop.Store(true)
					return
				}

			case core.Transaction_Contract_TransferAssetContract:
				depositList, err = decodeTransferAssetContract(r[0], hash) //omit assetName check
				if err != nil {
					errCh <- err
					needStop.Store(true)
					return
				}

			case core.Transaction_Contract_TriggerSmartContract:
				depositList, err = decodeTriggerSmartContract(r[0], txi, hash)
				if err != nil {
					errCh <- err
					return
				}
			}

			if len(depositList) == 0 {
				return
			}

			for _, deposit := range depositList {
				replyCh <- &proto.QueryAccountTransactionReply{
					Code:            proto.ReturnCode_SUCCESS,
					TxHash:          hash,
					TxStatus:        txStatus,
					From:            deposit.fromAddr,
					To:              deposit.toAddr,
					Amount:          deposit.amount,
					Memo:            "",
					Nonce:           0,
					BlockHeight:     uint64(txi.BlockNumber),
					BlockTime:       uint64(txi.BlockTimeStamp),
					GasPrice:        "1",
					GasLimit:        big.NewInt(tx.RawData.GetFeeLimit()).String(),
					CostFee:         big.NewInt(txi.GetFee()).String(),
					ContractAddress: deposit.contractAddr,
				}
			}
		}()
	}
	wg.Wait()
}

type semaphore chan struct{}

func (s semaphore) Acquire() {
	s <- struct{}{}
}

func (s semaphore) Release() {
	<-s
}

func stringToInt64(amount string) (int64, error) {
	log.Info("string to Int", "amount", amount)
	intAmount, success := big.NewInt(0).SetString(amount, 0)
	if !success {
		return 0, fmt.Errorf("fail to convert string%v to int64", amount)
	}
	return intAmount.Int64(), nil
}

func getHash(bz []byte) []byte {
	h := sha256.New()
	h.Write(bz)
	hash := h.Sum(nil)
	return hash
}

type depositInfo struct {
	tokenID      string
	fromAddr     string
	toAddr       string
	amount       string
	index        int
	contractAddr string
}

func decodeTransferContract(txContract *core.Transaction_Contract, txHash string) ([]depositInfo, error) {
	var tc core.TransferContract
	if err := ptypes.UnmarshalAny(txContract.GetParameter(), &tc); err != nil {
		return nil, err
	}
	fromAddress := address.Address(tc.OwnerAddress).String()
	toAddress := address.Address(tc.ToAddress).String()

	//log.Info("decodeTransferContract", "hash", txHash, "fromAddress", fromAddress, "toAddress", toAddress, "amount", tc.Amount)
	var tronDepositInfo depositInfo
	tronDepositInfo.tokenID = TronSymbol
	tronDepositInfo.fromAddr = fromAddress
	tronDepositInfo.toAddr = toAddress
	tronDepositInfo.amount = big.NewInt(tc.Amount).String()
	tronDepositInfo.contractAddr = ""
	return []depositInfo{tronDepositInfo}, nil
}

func decodeTransferAssetContract(txContract *core.Transaction_Contract, txHash string) ([]depositInfo, error) {
	var err error
	var tc core.TransferAssetContract
	if err := ptypes.UnmarshalAny(txContract.GetParameter(), &tc); err != nil {
		log.Error("UnmarshalAny TransferAssetContract", "hash", txHash, "err", err)
		return nil, err
	}
	fromAddress := address.Address(tc.OwnerAddress).String()
	toAddress := address.Address(tc.ToAddress).String()
	assetName := string(tc.AssetName)

	//	log.Info("decodeTransferAssetContract", "hash", txHash, "symbol", assetName, "fromAddress", fromAddress, "toAddress", toAddress, "amount", tc.Amount)
	var trc10DepositInfo depositInfo
	trc10DepositInfo.fromAddr = fromAddress
	trc10DepositInfo.toAddr = toAddress
	trc10DepositInfo.amount = big.NewInt(tc.Amount).String()
	trc10DepositInfo.contractAddr = assetName
	return []depositInfo{trc10DepositInfo}, err
}

func decodeTriggerSmartContract(txContract *core.Transaction_Contract, txi *core.TransactionInfo, txHash string) ([]depositInfo, error) {
	var tsc core.TriggerSmartContract
	if err := pb.Unmarshal(txContract.GetParameter().GetValue(), &tsc); err != nil {
		log.Error("decodeTriggerSmartContractLocal", "err", err, "hash", txHash)
		return nil, err
	}

	//decode only trc20transferMethod
	trc20TransferMethodByte, _ := hex.DecodeString(trc20TransferMethodSignature)
	if ok := bytes.HasPrefix(tsc.Data, trc20TransferMethodByte); !ok {
		return nil, nil
	}

	contractAddr := address.Address(tsc.ContractAddress).String()

	var depositList []depositInfo
	// check transfer info in log
	for i, txLog := range txi.Log {
		logAddrByte := []byte{}

		// transfer log topics must be 3
		if len(txLog.Topics) != trc20TransferTopicLen {
			log.Info("decodeTriggerSmartContract", "hash's len of topics is invalid", txHash)
			continue
		}
		if hex.EncodeToString(txLog.Topics[0]) == trc20TransferTopic {
			if len(txLog.Topics[1]) != trc20TransferAddrLen || len(txLog.Topics[2]) != trc20TransferAddrLen {
				log.Debug("decodeTriggerSmartContract", "invalid transfer addr len", txHash)
				continue
			}
			//address is 20 bytes
			fromBytes := txLog.Topics[1][12:]
			toBytes := txLog.Topics[2][12:]
			logAddrByte = append([]byte{address.TronBytePrefix}, fromBytes...)
			fromAddr := address.Address(logAddrByte).String()
			logAddrByte = append([]byte{address.TronBytePrefix}, toBytes...)
			toAddr := address.Address(logAddrByte).String()
			amount := new(big.Int).SetBytes(txLog.Data)

			//	log.Info("decodeTriggerSmartContract", "hash", txHash, "from", fromAddr, "to", toAddr, "amount", amount)

			var trc20DepositInfo depositInfo
			trc20DepositInfo.amount = amount.String()
			trc20DepositInfo.fromAddr = fromAddr
			trc20DepositInfo.toAddr = toAddr
			trc20DepositInfo.index = i
			trc20DepositInfo.contractAddr = contractAddr
			depositList = append(depositList, trc20DepositInfo)

		} else {
			//	log.Debug("decodeTriggerSmartContract", "hash is not transfer method", txHash)
			continue
		}
	}

	return depositList, nil
}

//IMPORTANT, current support only 1 TRC20 transfer
func decodeTriggerSmartContractLocal(txContract *core.Transaction_Contract, txHash string) ([]depositInfo, error) {
	var tsc core.TriggerSmartContract
	if err := pb.Unmarshal(txContract.GetParameter().GetValue(), &tsc); err != nil {
		log.Error("decodeTriggerSmartContractLocal", "err", err, "hash", txHash)
		return nil, err
	}

	//decode only trc20transferMethod
	trc20TransferMethodByte, _ := hex.DecodeString(trc20TransferMethodSignature)
	if ok := bytes.HasPrefix(tsc.Data, trc20TransferMethodByte); !ok {
		return nil, nil
	}

	fromAddr := address.Address(tsc.OwnerAddress).String()
	contractAddr := address.Address(tsc.ContractAddress).String()

	start := len(trc20TransferMethodByte)
	end := start + trc20TransferAddrLen
	start = end - address.AddressLength + 1

	addressTron := make([]byte, 0)
	addressTron = append(addressTron, address.TronBytePrefix)
	addressTron = append(addressTron, tsc.Data[start:end]...)

	toAddr := address.Address(addressTron).String()
	amount := new(big.Int).SetBytes(tsc.Data[end:])

	var trc20DepositInfo depositInfo
	trc20DepositInfo.amount = amount.String()
	trc20DepositInfo.fromAddr = fromAddr
	trc20DepositInfo.contractAddr = contractAddr
	trc20DepositInfo.toAddr = toAddr
	trc20DepositInfo.index = 0
	return []depositInfo{trc20DepositInfo}, nil
}

// queryTransaction should not be called to decode locally build
func queryTransactionLocal(txRaw *core.TransactionRaw, symbol string) (*proto.QueryAccountTransactionReply, error) {
	bz, err := pb.Marshal(txRaw)
	hash := hex.EncodeToString(getHash(bz))

	r := txRaw.Contract
	if len(r) != 1 {
		err := fmt.Errorf("QueryTransactionFromSignedData, tx's len(contract): %v !=1", len(r))
		log.Error(err.Error())
		return &proto.QueryAccountTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, err
	}

	var depositList []depositInfo
	switch r[0].Type {
	case core.Transaction_Contract_TransferContract:
		depositList, err = decodeTransferContract(r[0], hash)
		if err != nil {
			return &proto.QueryAccountTransactionReply{
				Code: proto.ReturnCode_ERROR,
				Msg:  err.Error(),
			}, err
		}

	case core.Transaction_Contract_TransferAssetContract:
		depositList, err = decodeTransferAssetContract(r[0], hash)
		if err != nil {
			return &proto.QueryAccountTransactionReply{
				Code: proto.ReturnCode_ERROR,
				Msg:  err.Error(),
			}, err
		}

	case core.Transaction_Contract_TriggerSmartContract:
		depositList, err = decodeTriggerSmartContractLocal(r[0], hash)
		if err != nil {
			return &proto.QueryAccountTransactionReply{
				Code: proto.ReturnCode_ERROR,
				Msg:  err.Error(),
			}, err
		}
	default:
		err = fmt.Errorf("QueryTransaction, unsupport contract type %v, tx hash %v ", r[0].Type, hash)
		log.Info(err.Error())
		return &proto.QueryAccountTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, err
	}

	if len(depositList) > 1 {
		err = fmt.Errorf("QueryTransaction, more than 1 deposit list %v, tx hash %v ", len(depositList), hash)
		log.Info(err.Error())
		return &proto.QueryAccountTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, err
	}

	return &proto.QueryAccountTransactionReply{
		Code:            proto.ReturnCode_SUCCESS,
		TxHash:          hash,
		From:            depositList[0].fromAddr,
		To:              depositList[0].toAddr,
		Amount:          depositList[0].amount,
		Memo:            "",
		Nonce:           0,
		GasPrice:        "1",
		GasLimit:        big.NewInt(defultGasLimit).String(),
		SignHash:        getHash(bz),
		ContractAddress: depositList[0].contractAddr,
	}, nil

}
