package tron

import (
	"crypto/ecdsa"
	"encoding/hex"
	"github.com/hbtc-chain/gotron-sdk/pkg/address"
	"github.com/hbtc-chain/gotron-sdk/pkg/proto/api"
	"github.com/hbtc-chain/gotron-sdk/pkg/proto/core"
	"github.com/btcsuite/btcd/btcec"
	"github.com/ethereum/go-ethereum/crypto"
	pb "github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"
	"math/big"
	"testing"
	"time"
)

func TestSendTrx(t *testing.T) {
	grpcClient := tronChainAdaptor.(*ChainAdaptor).client.grpcClient

	rawTx, err := grpcClient.Transfer("TYbcQrwHHjcd3n4pKGkxmCnjtw3nPoBs8b", "TPTuUqaJfhxcYFNdt76u11oZrDrm21RRHp", 12000000)
	require.Nil(t, err)

	//compare hash
	hash := rawTx.GetTxid()
	rawData, err := pb.Marshal(rawTx.GetTransaction().GetRawData())

	hash1 := getHash(rawData)
	require.EqualValues(t, hash, hash1)

	//marshal/unmarshal
	var rawTx1 core.TransactionRaw
	err = pb.Unmarshal(rawData, &rawTx1)
	require.Nil(t, err)
	require.Equal(t, rawTx.GetTransaction().GetRawData().GetRefBlockBytes(), rawTx1.GetRefBlockBytes())

	//sign
	key := []byte("key0")
	priv, pub := btcec.PrivKeyFromBytes(btcec.S256(), key)

	addr := address.PubkeyToAddress(*pub.ToECDSA())
	require.Equal(t, "TYbcQrwHHjcd3n4pKGkxmCnjtw3nPoBs8b", addr.String())
	t.Logf("hash:%v\n", hex.EncodeToString(hash))

	//tron original signature
	signature1, err := crypto.Sign(hash, (*ecdsa.PrivateKey)(priv))
	require.Nil(t, err)

	rawTx.GetTransaction().Signature = append(rawTx.GetTransaction().Signature, signature1)

	//time.Sleep(time.Second * 20)

	//bz, err := pb.Marshal(rawTx.GetTransaction())
	//require.Nil(t, err)
	//t.Logf("bz:%v", hex.EncodeToString(bz))

	res, err := grpcClient.Broadcast(rawTx.GetTransaction())
	require.Nil(t, err)
	t.Logf("res:%v", res)

	tx, err := grpcClient.GetTransactionByID(hex.EncodeToString(hash))
	require.Nil(t, err)
	t.Logf("tx:+%v", tx)

}

func TestSendTrx2(t *testing.T) {
	grpcClient := tronChainAdaptor.(*ChainAdaptor).client.grpcClient

	rawTx, err := grpcClient.Transfer("TYbcQrwHHjcd3n4pKGkxmCnjtw3nPoBs8b", "TJbsQjACJqnU5ZRaMWhGcWvi7uiZ6wpJto", 100000)
	require.Nil(t, err)

	//compare hash
	hash := rawTx.GetTxid()
	rawData, err := pb.Marshal(rawTx.GetTransaction().GetRawData())

	hash1 := getHash(rawData)
	require.EqualValues(t, hash, hash1)

	//marshal/unmarshal
	var rawTx1 core.TransactionRaw
	err = pb.Unmarshal(rawData, &rawTx1)
	require.Nil(t, err)
	require.Equal(t, rawTx.GetTransaction().GetRawData().GetRefBlockBytes(), rawTx1.GetRefBlockBytes())

	//sign
	key := []byte("key0")
	priv, pub := btcec.PrivKeyFromBytes(btcec.S256(), key)

	addr := address.PubkeyToAddress(*pub.ToECDSA())
	require.Equal(t, "TYbcQrwHHjcd3n4pKGkxmCnjtw3nPoBs8b", addr.String())
	//t.Logf("hash:%v\n", hex.EncodeToString(hash))

	//tron original signature
	signature1, err := crypto.Sign(hash, (*ecdsa.PrivateKey)(priv))
	require.Nil(t, err)

	rawTx.GetTransaction().Signature = append(rawTx.GetTransaction().Signature, signature1)

	bz, err := pb.Marshal(rawTx.GetTransaction())
	require.Nil(t, err)
	//t.Logf("bz:%v", hex.EncodeToString(bz))

	//unmarshal bz to core.Transaction
	var tx core.Transaction
	err = pb.Unmarshal(bz, &tx)
	require.Nil(t, err)

	res, err := grpcClient.Broadcast(&tx)
	require.Nil(t, err)
	require.Equal(t, api.Return_SUCCESS, res.Code)
	//t.Logf("res:%v", res)

	txp, err := grpcClient.GetTransactionByID(hex.EncodeToString(hash))
	require.Nil(t, err)
	t.Logf("tx:+%v", txp)

}

var (
	key     = []byte("key0")
	priv, _ = btcec.PrivKeyFromBytes(btcec.S256(), key)
)

func TestSendTrxWith60sDelay(t *testing.T) {
	grpcClient := tronChainAdaptor.(*ChainAdaptor).client.grpcClient

	rawTx, err := grpcClient.Transfer("TYbcQrwHHjcd3n4pKGkxmCnjtw3nPoBs8b", "TRGvtQpC8cpk1ksbwLn7xr5DK7aYZgmWcA", 10000000)
	require.Nil(t, err)

	signature1, err := crypto.Sign(rawTx.GetTxid(), (*ecdsa.PrivateKey)(priv))
	require.Nil(t, err)

	rawTx.GetTransaction().Signature = append(rawTx.GetTransaction().Signature, signature1)

	//interval = 20s
	t.Logf("sleep 60 seconds")
	time.Sleep(time.Duration(time.Second * 60))
	_, err = grpcClient.Broadcast(rawTx.GetTransaction())
	require.NotNil(t, err)
}

func TestSendTrxWith60sDelayChangeExpiration(t *testing.T) {
	grpcClient := tronChainAdaptor.(*ChainAdaptor).client.grpcClient

	rawTx, err := grpcClient.Transfer("TYbcQrwHHjcd3n4pKGkxmCnjtw3nPoBs8b", "THW8qemoiX5mF7m7kHWv333H4ZnP1amdYa", 10000000)
	require.Nil(t, err)

	rawTx.Transaction.RawData.Expiration = rawTx.Transaction.RawData.Timestamp + MaxTimeUntillExpiration

	rawData, err := pb.Marshal(rawTx.GetTransaction().GetRawData())
	hash := getHash(rawData)
	rawTx.Txid = hash

	signature1, err := crypto.Sign(rawTx.GetTxid(), (*ecdsa.PrivateKey)(priv))
	require.Nil(t, err)

	rawTx.GetTransaction().Signature = append(rawTx.GetTransaction().Signature, signature1)

	//interval = 20s
	t.Logf("sleep 60 seconds")
	time.Sleep(time.Duration(time.Minute * 1))
	_, err = grpcClient.Broadcast(rawTx.GetTransaction())
	require.Nil(t, err)

	time.Sleep(time.Second * 5)
	_, err = grpcClient.GetTransactionByID(hex.EncodeToString(rawTx.GetTxid()))
	require.Nil(t, err)
}

func TestSendTrxInReserveOrdder(t *testing.T) {
	grpcClient := tronChainAdaptor.(*ChainAdaptor).client.grpcClient

	rawTxs := make([]*api.TransactionExtention, 0)

	numOfTx := 5

	for i := 0; i < numOfTx; i++ {
		rawTx, err := grpcClient.Transfer("TYbcQrwHHjcd3n4pKGkxmCnjtw3nPoBs8b", "TJbsQjACJqnU5ZRaMWhGcWvi7uiZ6wpJto", 100000)
		require.Nil(t, err)

		//compare hash
		hash := rawTx.GetTxid()
		rawData, err := pb.Marshal(rawTx.GetTransaction().GetRawData())

		hash1 := getHash(rawData)
		require.EqualValues(t, hash, hash1)

		//marshal/unmarshal
		var rawTx1 core.TransactionRaw
		err = pb.Unmarshal(rawData, &rawTx1)
		require.Nil(t, err)
		require.Equal(t, rawTx.GetTransaction().GetRawData().GetRefBlockBytes(), rawTx1.GetRefBlockBytes())

		//sign
		key := []byte("key0")
		priv, pub := btcec.PrivKeyFromBytes(btcec.S256(), key)

		addr := address.PubkeyToAddress(*pub.ToECDSA())
		require.Equal(t, "TYbcQrwHHjcd3n4pKGkxmCnjtw3nPoBs8b", addr.String())
		//t.Logf("hash:%v\n", hex.EncodeToString(hash))

		//tron original signature
		signature1, err := crypto.Sign(hash, (*ecdsa.PrivateKey)(priv))
		require.Nil(t, err)

		rawTx.GetTransaction().Signature = append(rawTx.GetTransaction().Signature, signature1)
		rawTxs = append(rawTxs, rawTx)
		time.Sleep(time.Second * 3)
	}

	time.Sleep(time.Second * 2)

	for i := numOfTx; i > 0; i-- {
		sendRawTx := rawTxs[i-1]
		res, err := grpcClient.Broadcast(sendRawTx.GetTransaction())
		require.Nil(t, err)
		t.Logf("res:%v", res)

		time.Sleep(time.Millisecond * 500)
		tx, err := grpcClient.GetTransactionByID(hex.EncodeToString(sendRawTx.GetTxid()))
		require.Nil(t, err)
		t.Logf("tx:+%v", tx)
	}

}

//in tron
func TestIssueTrc10(t *testing.T) {
	grpcClient := tronChainAdaptor.(*ChainAdaptor).client.grpcClient

	startTime := time.Now().UTC().UnixNano()
	endTime := startTime + 10000

	rawTx, err := grpcClient.AssetIssue("TYbcQrwHHjcd3n4pKGkxmCnjtw3nPoBs8b", "TUSDT", "Test USDT", "TUSDT", "http://news.sina.com.cn", 6, 10000000000000, startTime, endTime, 0, 0, 1, 1, 0, nil)
	require.NotNil(t, err)

	if err == nil {
		//compare hash
		hash := rawTx.GetTxid()
		rawData, err := pb.Marshal(rawTx.GetTransaction().GetRawData())

		hash1 := getHash(rawData)
		require.EqualValues(t, hash, hash1)

		//marshal/unmarshal
		var rawTx1 core.TransactionRaw
		err = pb.Unmarshal(rawData, &rawTx1)
		require.Nil(t, err)
		require.Equal(t, rawTx.GetTransaction().GetRawData().GetRefBlockBytes(), rawTx1.GetRefBlockBytes())

		//sign
		key := []byte("key0")
		priv, pub := btcec.PrivKeyFromBytes(btcec.S256(), key)

		addr := address.PubkeyToAddress(*pub.ToECDSA())
		require.Equal(t, "TYbcQrwHHjcd3n4pKGkxmCnjtw3nPoBs8b", addr.String())
		t.Logf("hash:%v\n", hex.EncodeToString(hash))

		//tron original signature
		signature1, err := crypto.Sign(hash, (*ecdsa.PrivateKey)(priv))
		require.Nil(t, err)

		rawTx.GetTransaction().Signature = append(rawTx.GetTransaction().Signature, signature1)

		res, err := grpcClient.Broadcast(rawTx.GetTransaction())
		require.Nil(t, err)
		require.Equal(t, api.Return_SUCCESS, res.Code)
		t.Logf("res:%v", res)

		//time.Sleep(time.Second * 5)
		_, err = grpcClient.GetTransactionByID(hex.EncodeToString(hash))
		require.Nil(t, err)
		//t.Logf("tx:+%v", tx)
	}

}

func TestSendTrc10(t *testing.T) {
	grpcClient := tronChainAdaptor.(*ChainAdaptor).client.grpcClient

	rawTx, err := grpcClient.TransferAsset("TYbcQrwHHjcd3n4pKGkxmCnjtw3nPoBs8b", "TJbsQjACJqnU5ZRaMWhGcWvi7uiZ6wpJto", "1000315", 1000000)
	require.Nil(t, err)

	//compare hash
	hash := rawTx.GetTxid()
	rawData, err := pb.Marshal(rawTx.GetTransaction().GetRawData())

	hash1 := getHash(rawData)
	require.EqualValues(t, hash, hash1)

	//marshal/unmarshal
	var rawTx1 core.TransactionRaw
	err = pb.Unmarshal(rawData, &rawTx1)
	require.Nil(t, err)
	require.Equal(t, rawTx.GetTransaction().GetRawData().GetRefBlockBytes(), rawTx1.GetRefBlockBytes())

	//sign
	key := []byte("key0")
	priv, pub := btcec.PrivKeyFromBytes(btcec.S256(), key)
	t.Logf("priv:%v", hex.EncodeToString(priv.Serialize()))

	addr := address.PubkeyToAddress(*pub.ToECDSA())
	require.Equal(t, "TYbcQrwHHjcd3n4pKGkxmCnjtw3nPoBs8b", addr.String())
	//t.Logf("hash:%v\n", hex.EncodeToString(hash))

	//tron original signature
	signature1, err := crypto.Sign(hash, (*ecdsa.PrivateKey)(priv))
	require.Nil(t, err)

	rawTx.GetTransaction().Signature = append(rawTx.GetTransaction().Signature, signature1)

	res, err := grpcClient.Broadcast(rawTx.GetTransaction())
	require.Nil(t, err)
	t.Logf("res:%v", res)

	//time.Sleep(time.Second * 5)
	tx, err := grpcClient.GetTransactionByID(hex.EncodeToString(hash))
	require.Nil(t, err)
	t.Logf("tx:+%v", tx)

}

func TestSendTrc20(t *testing.T) {
	grpcClient := tronChainAdaptor.(*ChainAdaptor).client.grpcClient

	rawTx, err := grpcClient.TRC20Send("TYbcQrwHHjcd3n4pKGkxmCnjtw3nPoBs8b", "TJbsQjACJqnU5ZRaMWhGcWvi7uiZ6wpJto", "TU4oHpbNZjkji932GkYf4Pja1CxhpopQnF", big.NewInt(1000000), 1000000)
	require.Nil(t, err)

	//compare hash
	hash := rawTx.GetTxid()
	rawData, err := pb.Marshal(rawTx.GetTransaction().GetRawData())

	hash1 := getHash(rawData)
	require.EqualValues(t, hash, hash1)
	//t.Logf("hash:%v", hex.EncodeToString(hash))

	//marshal/unmarshal
	var rawTx1 core.TransactionRaw
	err = pb.Unmarshal(rawData, &rawTx1)
	require.Nil(t, err)
	require.Equal(t, rawTx.GetTransaction().GetRawData().GetRefBlockBytes(), rawTx1.GetRefBlockBytes())

	//sign
	key := []byte("key0")
	priv, _ := btcec.PrivKeyFromBytes(btcec.S256(), key)
	//	t.Logf("priv:%v", hex.EncodeToString(priv.Serialize()))

	//addr := address.PubkeyToAddress(*pub.ToECDSA())
	//require.Equal(t, "TYbcQrwHHjcd3n4pKGkxmCnjtw3nPoBs8b", addr.String())
	//t.Logf("hash:%v\n", hex.EncodeToString(hash))

	//tron original signature
	signature1, err := crypto.Sign(hash, (*ecdsa.PrivateKey)(priv))
	require.Nil(t, err)

	rawTx.GetTransaction().Signature = append(rawTx.GetTransaction().Signature, signature1)

	res, err := grpcClient.Broadcast(rawTx.GetTransaction())
	require.Nil(t, err)
	require.Equal(t, api.Return_SUCCESS, res.Code)
	//t.Logf("res:%v", res)

	//time.Sleep(time.Second * 5)
	tx, err := grpcClient.GetTransactionByID(hex.EncodeToString(hash))
	require.Nil(t, err)
	t.Logf("tx:+%v", tx)

}

//This case will success only once
//func TestDeployTrc20Contract(t *testing.T) {
//	grpcClient := tronChainAdaptor.(*ChainAdaptor).client.grpcClient
//
//	startTime := time.Now().UTC().UnixNano()
//	endTime := startTime + 10000
//
//	rawTx, err := grpcClient.DeployContract("TYbcQrwHHjcd3n4pKGkxmCnjtw3nPoBs8b", "TUSDT", "Test USDT", "TUSDT", "http://news.sina.com.cn", 6, 10000000000000, startTime, endTime, 0, 0, 1, 1, 0, nil)
//	require.Nil(t, err)
//
//	//compare hash
//	hash := rawTx.GetTxid()
//	rawData, err := pb.Marshal(rawTx.GetTransaction().GetRawData())
//
//	hash1, err := getHash(rawData)
//	require.EqualValues(t, hash, hash1)
//
//	//marshal/unmarshal
//	var rawTx1 core.TransactionRaw
//	err = pb.Unmarshal(rawData, &rawTx1)
//	require.Nil(t, err)
//	require.Equal(t, rawTx.GetTransaction().GetRawData().GetRefBlockBytes(), rawTx1.GetRefBlockBytes())
//
//	//sign
//	key := []byte("key0")
//	priv, pub := btcec.PrivKeyFromBytes(btcec.S256(), key)
//
//	addr := address.PubkeyToAddress(*pub.ToECDSA())
//	require.Equal(t, "TYbcQrwHHjcd3n4pKGkxmCnjtw3nPoBs8b", addr.String())
//	t.Logf("hash:%v\n", hex.EncodeToString(hash))
//
//	//tron original signature
//	signature1, err := crypto.Sign(hash, (*ecdsa.PrivateKey)(priv))
//	require.Nil(t, err)
//
//	rawTx.GetTransaction().Signature = append(rawTx.GetTransaction().Signature, signature1)
//
//	res, err := grpcClient.Broadcast(rawTx.GetTransaction())
//	require.Nil(t, err)
//	t.Logf("res:%v", res)
//
//	//time.Sleep(time.Second * 5)
//	tx, err := grpcClient.GetTransactionByID(hex.EncodeToString(hash))
//	require.Nil(t, err)
//	t.Logf("tx:+%v", tx)
//
//}
