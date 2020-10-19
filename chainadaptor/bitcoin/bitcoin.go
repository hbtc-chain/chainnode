package bitcoin

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"

	"github.com/hbtc-chain/chainnode/cache"
	"github.com/hbtc-chain/chainnode/chainadaptor"
	"github.com/hbtc-chain/chainnode/chainadaptor/fallback"
	"github.com/hbtc-chain/chainnode/config"
	"github.com/hbtc-chain/chainnode/proto"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/shopspring/decimal"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	confirms     = 1
	btcDecimals  = 8
	btcFeeBlocks = 3

	ChainName = "btc"
	Symbol    = "btc"
)

type ChainAdaptor struct {
	fallback.ChainAdaptor
	client *btcClient
}

func NewChainAdaptor(conf *config.Config) (chainadaptor.ChainAdaptor, error) {
	client, err := newBtcClient(conf)
	if err != nil {
		return nil, err
	}
	return newChainAdaptorWithClient(client), nil
}

func NewLocalChainAdaptor(network config.NetWorkType) chainadaptor.ChainAdaptor {
	return newChainAdaptorWithClient(newLocalBtcClient(network))
}

func newChainAdaptorWithClient(client *btcClient) *ChainAdaptor {
	return &ChainAdaptor{
		client: client,
	}
}

// ConvertAddress convert pubkey to a actual address
// TODO(keep), currently convert btc pubkey to btc address, may convert bhex pubkey to btc address
func (a *ChainAdaptor) ConvertAddress(req *proto.ConvertAddressRequest) (*proto.ConvertAddressReply, error) {
	addressPubKey, err := btcutil.NewAddressPubKey(req.PublicKey, a.client.GetNetwork())
	if err != nil {
		return &proto.ConvertAddressReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, err
	}

	return &proto.ConvertAddressReply{
		Code:    proto.ReturnCode_SUCCESS,
		Address: addressPubKey.EncodeAddress(),
	}, nil
}

// ValidAddress check whether an address is valid
func (a *ChainAdaptor) ValidAddress(req *proto.ValidAddressRequest) (*proto.ValidAddressReply, error) {
	address, err := btcutil.DecodeAddress(req.Address, a.client.GetNetwork())
	if err != nil {
		return &proto.ValidAddressReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, err
	}

	if !address.IsForNet(a.client.GetNetwork()) {
		err := errors.New("address is not valid for this network")
		return &proto.ValidAddressReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, err
	}
	return &proto.ValidAddressReply{
		Code:             proto.ReturnCode_SUCCESS,
		Valid:            true,
		CanWithdrawal:    true,
		CanonicalAddress: address.String(),
	}, nil
}

func (a *ChainAdaptor) QueryGasPrice(*proto.QueryGasPriceRequest) (*proto.QueryGasPriceReply, error) {
	reply, err := a.client.EstimateSmartFee(btcFeeBlocks)
	if err != nil {
		log.Info("QueryGasPrice", "err", err)
		return &proto.QueryGasPriceReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, err
	}

	price := btcToSatoshi(reply.Feerate)
	return &proto.QueryGasPriceReply{
		Code:     proto.ReturnCode_SUCCESS,
		GasPrice: price.String(),
	}, nil
}

func (a *ChainAdaptor) QueryUtxo(req *proto.QueryUtxoRequest) (*proto.QueryUtxoReply, error) {
	utxo := req.Vin
	txhash, err := chainhash.NewHashFromStr(utxo.Hash)
	if err != nil {
		log.Info("QueryUtxo NewHashFromStr", "err", err)

		return &proto.QueryUtxoReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, err
	}

	reply, err := a.client.GetTxOut(txhash, utxo.Index, true)
	if err != nil {
		log.Info("QueryUtxo GetTxOut", "err", err)

		return &proto.QueryUtxoReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, err
	}

	if reply == nil {
		log.Info("QueryUtxo GetTxOut", "err", "hash not found")
		err = errors.New("hash not found")
		return &proto.QueryUtxoReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, err
	}

	if btcToSatoshi(reply.Value).Int64() != utxo.Amount {
		log.Info("QueryUtxo GetTxOut", "err", "amount mismatch")

		err = errors.New("amount mismatch")
		return &proto.QueryUtxoReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, err
	}

	tx, err := a.client.GetRawTransactionVerbose(txhash)
	if err != nil {
		log.Info("QueryUtxo GetRawTransactionVerbose", "err", err)

		return &proto.QueryUtxoReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, err
	}

	if tx.Vout[utxo.Index].ScriptPubKey.Addresses[0] != utxo.Address {
		log.Info("QueryUtxo GetTxOut", "err", "address mismatch")

		err := errors.New("address mismatch")
		return &proto.QueryUtxoReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, err
	}

	return &proto.QueryUtxoReply{
		Code:    proto.ReturnCode_SUCCESS,
		Unspent: true,
	}, nil

}

// QueryTransaction query tx info from chain
func (a *ChainAdaptor) QueryUtxoTransaction(req *proto.QueryTransactionRequest) (*proto.QueryUtxoTransactionReply, error) {
	key := strings.Join([]string{req.Symbol, req.TxHash}, ":")
	txCache := cache.GetTxCache()
	if r, exist := txCache.Get(key); exist {
		return r.(*proto.QueryUtxoTransactionReply), nil
	}

	txhash, err := chainhash.NewHashFromStr(req.TxHash)
	if err != nil {
		return &proto.QueryUtxoTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, err
	}
	reply, err := a.queryTransaction(txhash)
	if err == nil && reply.TxStatus == proto.TxStatus_Success {
		txCache.Add(key, reply)
	}

	return reply, err
}

// QueryTransactionFromSignedData query tx info from chain
func (a *ChainAdaptor) QueryUtxoTransactionFromSignedData(req *proto.QueryTransactionFromSignedDataRequest) (*proto.QueryUtxoTransactionReply, error) {
	res, err := a.decodeTx(req.SignedTxData, req.Vins, true)
	if err != nil {
		log.Info("QueryTransactionFromSignedData decodeTx", "err", err)

		return &proto.QueryUtxoTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, err
	}
	return &proto.QueryUtxoTransactionReply{
		Code:       proto.ReturnCode_SUCCESS,
		TxHash:     res.Hash,
		TxStatus:   proto.TxStatus_Other,
		Vins:       res.Vins,
		Vouts:      res.Vouts,
		CostFee:    res.CostFee.String(),
		SignHashes: res.SignHashes,
	}, nil
}

func (a *ChainAdaptor) QueryUtxoTransactionFromData(req *proto.QueryTransactionFromDataRequest) (*proto.QueryUtxoTransactionReply, error) {
	res, err := a.decodeTx(req.RawData, req.Vins, false)
	if err != nil {
		log.Info("QueryTransactionFromData decodeTx", "err", err)

		return &proto.QueryUtxoTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, err
	}
	return &proto.QueryUtxoTransactionReply{
		Code:       proto.ReturnCode_SUCCESS,
		SignHashes: res.SignHashes,
		TxStatus:   proto.TxStatus_Other,
		Vins:       res.Vins,
		Vouts:      res.Vouts,
		CostFee:    res.CostFee.String(),
	}, nil
}

// CreateTransaction make a transaction without signature
func (a *ChainAdaptor) CreateUtxoTransaction(req *proto.CreateUtxoTransactionRequest) (*proto.CreateUtxoTransactionReply, error) {
	vinNum := len(req.Vins)
	var totalAmountIn, totalAmountOut int64

	if vinNum == 0 {
		err := fmt.Errorf("no Vin in req:%v", req)
		return &proto.CreateUtxoTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, err
	}

	// check the Fee
	fee, ok := big.NewInt(0).SetString(req.Fee, 0)
	if !ok {
		err := errors.New("CreateTransaction, fail to get fee")
		return &proto.CreateUtxoTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, err
	}
	for _, in := range req.Vins {
		totalAmountIn += in.Amount
	}

	for _, out := range req.Vouts {
		totalAmountOut += out.Amount
	}

	if totalAmountIn != totalAmountOut+fee.Int64() {
		err := errors.New("CreateTransaction, total amount in != total amount out + fee")
		return &proto.CreateUtxoTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, err
	}

	rawTx, err := a.createRawTx(req.Vins, req.Vouts)
	if err != nil {
		return &proto.CreateUtxoTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, err
	}

	buf := bytes.NewBuffer(make([]byte, 0, rawTx.SerializeSize()))
	err = rawTx.Serialize(buf)
	if err != nil {
		return &proto.CreateUtxoTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, err
	}

	// build the pkScript and Generate signhash for each Vin,
	signHashes, err := a.calcSignHashes(req.Vins, req.Vouts)
	if err != nil {
		return &proto.CreateUtxoTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, err
	}
	log.Info("CreateTransaction", "usigned tx", hex.EncodeToString(buf.Bytes()))

	return &proto.CreateUtxoTransactionReply{
		Code:       proto.ReturnCode_SUCCESS,
		TxData:     buf.Bytes(),
		SignHashes: signHashes,
	}, nil
}

// CreateSignedTransaction make a transaction without signature
func (a *ChainAdaptor) CreateUtxoSignedTransaction(req *proto.CreateUtxoSignedTransactionRequest) (*proto.CreateSignedTransactionReply, error) {
	r := bytes.NewReader(req.TxData)
	var msgTx wire.MsgTx
	err := msgTx.Deserialize(r)
	if err != nil {
		log.Error("CreateSignedTransaction msgTx.Deserialize", "err", err)

		return &proto.CreateSignedTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, err
	}

	if len(req.Signatures) != len(msgTx.TxIn) {
		log.Error("CreateSignedTransaction invalid params", "err", "Signature number mismatch Txin number")
		err = errors.New("Signature number != Txin number")
		return &proto.CreateSignedTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, err
	}

	if len(req.PublicKeys) != len(msgTx.TxIn) {
		log.Error("CreateSignedTransaction invalid params", "err", "Pubkey number mismatch Txin number")
		err = errors.New("Pubkey number != Txin number")
		return &proto.CreateSignedTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, err
	}

	// assemble signatures
	for i, in := range msgTx.TxIn {
		btcecPub, err2 := btcec.ParsePubKey(req.PublicKeys[i], btcec.S256())
		if err2 != nil {
			log.Error("CreateSignedTransaction ParsePubKey", "err", err2)
			return &proto.CreateSignedTransactionReply{
				Code: proto.ReturnCode_ERROR,
				Msg:  err2.Error(),
			}, err2
		}

		var pkData []byte
		if btcec.IsCompressedPubKey(req.PublicKeys[i]) {
			pkData = btcecPub.SerializeCompressed()
		} else {
			pkData = btcecPub.SerializeUncompressed()
		}

		// verify transaction
		preTx, err2 := a.client.GetRawTransactionVerbose(&in.PreviousOutPoint.Hash)
		if err2 != nil {
			log.Error("CreateSignedTransaction GetRawTransactionVerbose", "err", err2)

			return &proto.CreateSignedTransactionReply{
				Code: proto.ReturnCode_ERROR,
				Msg:  err2.Error(),
			}, err2
		}

		log.Info("CreateSignedTransaction ", "from address", preTx.Vout[in.PreviousOutPoint.Index].ScriptPubKey.Addresses[0])

		fromAddress, err2 := btcutil.DecodeAddress(preTx.Vout[in.PreviousOutPoint.Index].ScriptPubKey.Addresses[0], a.client.GetNetwork())
		if err2 != nil {
			log.Error("CreateSignedTransaction DecodeAddress", "err", err2)

			return &proto.CreateSignedTransactionReply{
				Code: proto.ReturnCode_ERROR,
				Msg:  err2.Error(),
			}, err2
		}

		fromPkScript, err2 := txscript.PayToAddrScript(fromAddress)
		if err2 != nil {
			log.Error("CreateSignedTransaction PayToAddrScript", "err", err2)

			return &proto.CreateSignedTransactionReply{
				Code: proto.ReturnCode_ERROR,
				Msg:  err2.Error(),
			}, err2
		}

		// creat sigscript and verify
		if len(req.Signatures[i]) < 64 {
			err2 = errors.New("Invalid signature length")
			return &proto.CreateSignedTransactionReply{
				Code: proto.ReturnCode_ERROR,
				Msg:  err2.Error(),
			}, err2
		}
		r := new(big.Int).SetBytes(req.Signatures[i][0:32])
		s := new(big.Int).SetBytes(req.Signatures[i][32:64])

		btcecSig := &btcec.Signature{
			R: r,
			S: s,
		}
		sig := append(btcecSig.Serialize(), byte(txscript.SigHashAll))
		sigScript, err2 := txscript.NewScriptBuilder().AddData(sig).AddData(pkData).Script()
		if err2 != nil {
			log.Error("CreateSignedTransaction NewScriptBuilder", "err", err2)

			return &proto.CreateSignedTransactionReply{
				Code: proto.ReturnCode_ERROR,
				Msg:  err2.Error(),
			}, err2
		}

		msgTx.TxIn[i].SignatureScript = sigScript
		amount := btcToSatoshi(preTx.Vout[in.PreviousOutPoint.Index].Value).Int64()
		log.Info("CreateSignedTransaction ", "amount", preTx.Vout[in.PreviousOutPoint.Index].Value, "int amount", amount)

		vm, err2 := txscript.NewEngine(fromPkScript, &msgTx, i, txscript.StandardVerifyFlags, nil, nil, amount)
		if err2 != nil {
			log.Error("CreateSignedTransaction NewEngine", "err", err2)

			return &proto.CreateSignedTransactionReply{
				Code: proto.ReturnCode_ERROR,
				Msg:  err2.Error(),
			}, err2
		}
		if err3 := vm.Execute(); err3 != nil {
			log.Error("CreateSignedTransaction NewEngine Execute", "err", err3)

			return &proto.CreateSignedTransactionReply{
				Code: proto.ReturnCode_ERROR,
				Msg:  err3.Error(),
			}, err3
		}

	}

	// serialize tx
	buf := bytes.NewBuffer(make([]byte, 0, msgTx.SerializeSize()))

	err = msgTx.Serialize(buf)
	if err != nil {
		log.Error("CreateSignedTransaction tx Serialize", "err", err)

		return &proto.CreateSignedTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, err
	}

	hash := msgTx.TxHash()
	return &proto.CreateSignedTransactionReply{
		Code:         proto.ReturnCode_SUCCESS,
		SignedTxData: buf.Bytes(),
		Hash:         (&hash).CloneBytes(),
	}, nil
}

// BroadcastTransaction add signature into transaction and broadcast it to chain
func (a *ChainAdaptor) BroadcastTransaction(req *proto.BroadcastTransactionRequest) (*proto.BroadcastTransactionReply, error) {
	r := bytes.NewReader(req.SignedTxData)
	var msgTx wire.MsgTx
	err := msgTx.Deserialize(r)
	if err != nil {
		return &proto.BroadcastTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, err
	}

	txHash, err := a.client.SendRawTransaction(&msgTx)
	if err != nil {
		return &proto.BroadcastTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, err
	}

	if strings.Compare(msgTx.TxHash().String(), txHash.String()) != 0 {
		log.Error("BroadcastTransaction, txhash mismatch", "local hash", msgTx.TxHash().String(), "hash from net", txHash.String(), "signedTx", hex.EncodeToString(req.SignedTxData))
	}

	return &proto.BroadcastTransactionReply{
		Code:   proto.ReturnCode_SUCCESS,
		TxHash: txHash.String(),
	}, nil
}

func (a *ChainAdaptor) VerifyUtxoSignedTransaction(req *proto.VerifySignedTransactionRequest) (*proto.VerifySignedTransactionReply, error) {
	_, err := a.decodeTx(req.SignedTxData, req.Vins, true)
	if err != nil {
		log.Error("VerifySignedTransaction", "decodeTx err", err)
		return &proto.VerifySignedTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, err
	}

	return &proto.VerifySignedTransactionReply{
		Code:     proto.ReturnCode_SUCCESS,
		Verified: true,
	}, nil

}

func (a *ChainAdaptor) QueryUtxoInsFromData(req *proto.QueryUtxoInsFromDataRequest) (*proto.QueryUtxoInsReply, error) {
	log.Info("QueryUtxoInsFromData", "req", req)
	vins, err := decodeProtoVinsFromData(req.Data)
	if err != nil {
		return &proto.QueryUtxoInsReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, err
	}

	return &proto.QueryUtxoInsReply{
		Code: proto.ReturnCode_SUCCESS,
		Vins: vins,
	}, err
}

func (a *ChainAdaptor) GetLatestBlockHeight() (int64, error) {
	return a.client.GetBlockCount()
}

func (a *ChainAdaptor) GetUtxoTransactionByHeight(height int64, replyCh chan *proto.QueryUtxoTransactionReply, errCh chan error) {

	hash, err := a.client.GetBlockHash(height)
	if err != nil {
		errCh <- err
		return
	}
	block, err := a.client.GetBlockWithRawTransactionVerbose(hash)
	if err != nil {
		errCh <- err
		return
	}

	for _, tx := range block.Tx {
		reply, err := a.assembleUtxoTransactionReply(tx, block.Height, block.Time, func(txid string, index uint32) (int64, string, error) {
			return 0, "", nil
		})
		if err != nil {
			errCh <- err
			return
		}
		replyCh <- reply
	}
}

func (a *ChainAdaptor) queryTransaction(txhash *chainhash.Hash) (*proto.QueryUtxoTransactionReply, error) {
	tx, err := a.client.GetRawTransactionVerbose(txhash)
	if err != nil {
		if rpcErr, ok := err.(*btcjson.RPCError); ok && rpcErr.Code == btcjson.ErrRPCBlockNotFound {
			return &proto.QueryUtxoTransactionReply{
				Code:     proto.ReturnCode_SUCCESS,
				TxStatus: proto.TxStatus_NotFound,
			}, nil
		}
		log.Error("queryTransaction GetRawTransactionVerbose", "err", err)
		return &proto.QueryUtxoTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, err
	}

	if tx == nil || txhash.String() != tx.Txid {
		log.Error("queryTransaction txid mismatch")

		return &proto.QueryUtxoTransactionReply{
			Code:     proto.ReturnCode_SUCCESS,
			TxStatus: proto.TxStatus_NotFound,
		}, nil
	}

	if tx.Confirmations < confirms {
		log.Error("queryTransaction confirmes too low", "tx confirms", tx.Confirmations, "need confirms", confirms)

		return &proto.QueryUtxoTransactionReply{
			Code:     proto.ReturnCode_SUCCESS,
			TxStatus: proto.TxStatus_Pending,
		}, nil
	}

	blockHash, _ := chainhash.NewHashFromStr(tx.BlockHash)
	block, err := a.client.GetBlockVerbose(blockHash)
	if err != nil {
		log.Error("queryTransaction GetBlockVerbose", "err", err)

		return &proto.QueryUtxoTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, err
	}

	reply, err := a.assembleUtxoTransactionReply(tx, block.Height, block.Time, func(txid string, index uint32) (int64, string, error) {
		preHash, err2 := chainhash.NewHashFromStr(txid)
		if err2 != nil {
			return 0, "", err2
		}
		preTx, err2 := a.client.GetRawTransactionVerbose(preHash)
		if err2 != nil {
			return 0, "", err2
		}
		amount := btcToSatoshi(preTx.Vout[index].Value).Int64()

		return amount, preTx.Vout[index].ScriptPubKey.Addresses[0], nil
	})
	if err != nil {
		return &proto.QueryUtxoTransactionReply{
			Code: proto.ReturnCode_ERROR,
			Msg:  err.Error(),
		}, err
	}

	return reply, nil
}

func (a *ChainAdaptor) assembleUtxoTransactionReply(tx *btcjson.TxRawResult, blockHeight, blockTime int64, getPrevTxInfo func(txid string, index uint32) (int64, string, error)) (*proto.QueryUtxoTransactionReply, error) {
	var totalAmountIn, totalAmountOut int64
	ins := make([]*proto.Vin, 0, len(tx.Vin))
	outs := make([]*proto.Vout, 0, len(tx.Vout))

	for _, in := range tx.Vin {
		amount, address, err := getPrevTxInfo(in.Txid, in.Vout)
		if err != nil {
			return nil, err
		}
		totalAmountIn += amount

		t := proto.Vin{
			Hash:    in.Txid,
			Index:   in.Vout,
			Amount:  amount,
			Address: address,
		}
		ins = append(ins, &t)
	}

	for index, out := range tx.Vout {
		amount := btcToSatoshi(out.Value).Int64()
		addr := ""
		if len(out.ScriptPubKey.Addresses) > 0 {
			addr = out.ScriptPubKey.Addresses[0]
		}

		totalAmountOut += amount
		t := proto.Vout{
			Address: addr,
			Amount:  amount,
			Index:   uint32(index),
		}

		outs = append(outs, &t)
	}

	gasUsed := totalAmountIn - totalAmountOut
	reply := &proto.QueryUtxoTransactionReply{
		Code:        proto.ReturnCode_SUCCESS,
		TxHash:      tx.Txid,
		TxStatus:    proto.TxStatus_Success,
		Vins:        ins,
		Vouts:       outs,
		CostFee:     strconv.FormatInt(gasUsed, 10),
		BlockHeight: uint64(blockHeight),
		BlockTime:   uint64(blockTime),
	}
	return reply, nil
}

func btcToSatoshi(btcCount float64) *big.Int {
	amount := strconv.FormatFloat(btcCount, 'f', -1, 64)
	amountDm, _ := decimal.NewFromString(amount)
	tenDm := decimal.NewFromFloat(math.Pow(10, float64(btcDecimals)))
	satoshiDm, _ := big.NewInt(0).SetString(amountDm.Mul(tenDm).String(), 10)
	return satoshiDm
}

func (a *ChainAdaptor) createRawTx(ins []*proto.Vin, outs []*proto.Vout) (*wire.MsgTx, error) {
	if len(ins) == 0 || len(outs) == 0 {
		return nil, errors.New("invalid len in or out")
	}

	rawTx := wire.NewMsgTx(wire.TxVersion)
	for _, in := range ins {
		// convert string hash to a bitcoin hash
		utxoHash, err := chainhash.NewHashFromStr(in.Hash)
		if err != nil {
			return nil, err
		}

		// make a txin
		txIn := wire.NewTxIn(wire.NewOutPoint(utxoHash, in.Index), nil, nil)
		// add txIn to transaction
		rawTx.AddTxIn(txIn)
	}

	// build tx output
	for _, out := range outs {
		if strings.HasPrefix(out.Address, omniPrefix) {
			toPkScript, err := buildOmniScript(out.Address)
			if err != nil {
				return nil, err
			}
			rawTx.AddTxOut(wire.NewTxOut(out.Amount, toPkScript))
			continue
		}

		toAddress, err := btcutil.DecodeAddress(out.Address, a.client.GetNetwork())
		if err != nil {
			return nil, err
		}

		// build the pkScript
		toPkScript, err := txscript.PayToAddrScript(toAddress)
		if err != nil {
			return nil, err
		}
		// add txOut to transaction
		rawTx.AddTxOut(wire.NewTxOut(out.Amount, toPkScript))
	}

	return rawTx, nil
}

func buildOmniScript(addr string) ([]byte, error) {
	omniData, err := hex.DecodeString(addr)
	if err != nil {
		return nil, errors.New("parse omni data error")
	}
	return txscript.NewScriptBuilder().AddOp(txscript.OP_RETURN).AddData(omniData).Script()
}

type DecodeTxRes struct {
	Hash       string
	SignHashes [][]byte
	Vins       []*proto.Vin
	Vouts      []*proto.Vout
	CostFee    *big.Int
}

func (a *ChainAdaptor) decodeTx(txData []byte, vins []*proto.Vin, sign bool) (*DecodeTxRes, error) {
	var msgTx wire.MsgTx
	err := msgTx.Deserialize(bytes.NewReader(txData))
	if err != nil {
		return nil, err
	}

	offline := true
	if len(vins) == 0 {
		offline = false
	}
	if offline && len(vins) != len(msgTx.TxIn) {
		return nil, status.Error(codes.InvalidArgument, "the length of deserialized tx's in differs from vin in req")
	}

	ins, totalAmountIn, err := a.decodeVins(msgTx, offline, vins, sign)
	if err != nil {
		return nil, err
	}

	outs, totalAmountOut, err := a.decodeVouts(msgTx)
	if err != nil {
		return nil, err
	}

	// build the pkScript and Generate signhash for each Vin,
	signHashes, err := a.calcSignHashes(ins, outs)
	if err != nil {
		return nil, err
	}

	res := DecodeTxRes{
		SignHashes: signHashes,
		Vins:       ins,
		Vouts:      outs,
		CostFee:    totalAmountIn.Sub(totalAmountIn, totalAmountOut),
	}
	if sign {
		res.Hash = msgTx.TxHash().String()
	}
	return &res, nil
}

func (a *ChainAdaptor) decodeVouts(msgTx wire.MsgTx) ([]*proto.Vout, *big.Int, error) {
	outs := make([]*proto.Vout, 0, len(msgTx.TxOut))
	totalAmountOut := big.NewInt(0)
	for _, out := range msgTx.TxOut {
		var t proto.Vout
		_, pubkeyAddrs, _, err := txscript.ExtractPkScriptAddrs(out.PkScript, a.client.GetNetwork())
		if err != nil {
			return nil, nil, err
		}
		t.Address = pubkeyAddrs[0].EncodeAddress()
		t.Amount = out.Value
		totalAmountOut.Add(totalAmountOut, big.NewInt(t.Amount))
		outs = append(outs, &t)
	}
	return outs, totalAmountOut, nil
}

func (a *ChainAdaptor) decodeVins(msgTx wire.MsgTx, offline bool, vins []*proto.Vin, sign bool) ([]*proto.Vin, *big.Int, error) {
	// verify signatures and decode
	ins := make([]*proto.Vin, 0, len(msgTx.TxIn))
	totalAmountIn := big.NewInt(0)
	for index, in := range msgTx.TxIn {
		vin, err := a.getVin(offline, vins, index, in)
		if err != nil {
			return nil, nil, err
		}

		if sign {
			err = a.verifySign(vin, msgTx, index)
			if err != nil {
				return nil, nil, err
			}
		}

		totalAmountIn.Add(totalAmountIn, big.NewInt(vin.Amount))
		ins = append(ins, vin)
	}
	return ins, totalAmountIn, nil
}

func (a *ChainAdaptor) getVin(offline bool, vins []*proto.Vin, index int, in *wire.TxIn) (*proto.Vin, error) {
	var vin *proto.Vin
	if offline {
		vin = vins[index]
	} else {
		preTx, err := a.client.GetRawTransactionVerbose(&in.PreviousOutPoint.Hash)
		if err != nil {
			return nil, err
		}
		out := preTx.Vout[in.PreviousOutPoint.Index]
		vin = &proto.Vin{
			Hash:    "",
			Index:   0,
			Amount:  btcToSatoshi(out.Value).Int64(),
			Address: out.ScriptPubKey.Addresses[0],
		}
	}
	vin.Hash = in.PreviousOutPoint.Hash.String()
	vin.Index = in.PreviousOutPoint.Index
	return vin, nil
}

func (a *ChainAdaptor) verifySign(vin *proto.Vin, msgTx wire.MsgTx, index int) error {
	fromAddress, err := btcutil.DecodeAddress(vin.Address, a.client.GetNetwork())
	if err != nil {
		return err
	}

	fromPkScript, err := txscript.PayToAddrScript(fromAddress)
	if err != nil {
		return err
	}

	vm, err := txscript.NewEngine(fromPkScript, &msgTx, index, txscript.StandardVerifyFlags, nil, nil, vin.Amount)
	if err != nil {
		return err
	}
	return vm.Execute()
}

func (a *ChainAdaptor) calcSignHashes(Vins []*proto.Vin, Vouts []*proto.Vout) ([][]byte, error) {

	rawTx, err := a.createRawTx(Vins, Vouts)
	if err != nil {
		return nil, err
	}

	signHashes := make([][]byte, len(Vins))
	for i, in := range Vins {
		from := in.Address
		fromAddr, err := btcutil.DecodeAddress(from, a.client.GetNetwork())
		if err != nil {
			log.Info("DecodeAddress err", "from", from, "err", err)
			return nil, err
		}
		fromPkScript, err := txscript.PayToAddrScript(fromAddr)
		if err != nil {
			log.Info("PayToAddrScript err", "err", err)
			return nil, err
		}

		signHash, err := txscript.CalcSignatureHash(fromPkScript, txscript.SigHashAll, rawTx, i)
		if err != nil {
			log.Info("CalcSignatureHash err", "err", err)
			return nil, err
		}
		signHashes[i] = signHash
	}
	return signHashes, nil
}

func decodeProtoVinsFromData(data []byte) ([]*proto.Vin, error) {
	r := bytes.NewReader(data)
	var msgTx wire.MsgTx
	err := msgTx.Deserialize(r)
	if err != nil {
		return nil, err
	}

	ins := make([]*proto.Vin, len(msgTx.TxIn))

	for i, in := range msgTx.TxIn {
		t := proto.Vin{}
		t.Hash = in.PreviousOutPoint.Hash.String()
		t.Index = in.PreviousOutPoint.Index
		ins[i] = &t
	}

	return ins, nil
}
