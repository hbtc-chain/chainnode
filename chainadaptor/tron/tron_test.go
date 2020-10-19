package tron

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hbtc-chain/chainnode/chainadaptor"
	"github.com/hbtc-chain/chainnode/config"
	"github.com/hbtc-chain/chainnode/proto"
	"github.com/hbtc-chain/gotron-sdk/pkg/address"
	"github.com/btcsuite/btcd/btcec"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

var tronChainAdaptor chainadaptor.ChainAdaptor
var tronChainAdaptorWithoutFullNode chainadaptor.ChainAdaptor

func TestMain(m *testing.M) {
	conf, err := config.New("testnet.yaml")
	if err != nil {
		panic(err)
	}

	tronChainAdaptor, err = NewChainAdaptor(conf)
	if err != nil {
		panic(err)
	}
	tronChainAdaptorWithoutFullNode = NewLocalChainAdaptor(config.TestNet)
	os.Exit(m.Run())
}

func TestConvertAddress(t *testing.T) {
	for i := 0; i < 10; i++ {
		key := []byte(fmt.Sprintf("key%v", i))
		_, btcecPublicKey := btcec.PrivKeyFromBytes(btcec.S256(), key)

		compPubKeyBytes := btcecPublicKey.SerializeCompressed()
		req1 := &proto.ConvertAddressRequest{
			Chain:     ChainName,
			PublicKey: compPubKeyBytes,
		}

		res1, err := tronChainAdaptor.ConvertAddress(req1)
		require.Nil(t, err)

		uncompPubKeyBytes := btcecPublicKey.SerializeUncompressed()
		req2 := &proto.ConvertAddressRequest{
			Chain:     ChainName,
			PublicKey: uncompPubKeyBytes,
		}

		res2, err := tronChainAdaptor.ConvertAddress(req2)
		require.Nil(t, err)
		require.Equal(t, res1.Address, res2.Address)
		//t.Logf("uncompPubKeyBytes:%v, compPubKeyBytes:%v res:%v", hex.EncodeToString(uncompPubKeyBytes), hex.EncodeToString(compPubKeyBytes), res1.Address)
	}
}

func TestValidAddress(t *testing.T) {
	testdata := []struct {
		address       string
		isValid       bool
		canWithdrawal bool
		stdAddress    string
	}{
		{"TYbcQrwHHjcd3n4pKGkxmCnjtw3nPoBs8b", true, true, "TYbcQrwHHjcd3n4pKGkxmCnjtw3nPoBs8b"},
		{"TJbsQjACJqnU5ZRaMWhGcWvi7uiZ6wpJto", true, true, "TJbsQjACJqnU5ZRaMWhGcWvi7uiZ6wpJto"},
		{"TDBtQ5eFNwWAUWJLhFgWm1MVrQYTQbJQii", true, true, "TDBtQ5eFNwWAUWJLhFgWm1MVrQYTQbJQii"},
		{"THtbMw6byXuiFhsRv1o1BQRtzvube9X1jx", true, true, "THtbMw6byXuiFhsRv1o1BQRtzvube9X1jx"},
		{"TU4oHpbNZjkji932GkYf4Pja1CxhpopQnF", true, false, "TU4oHpbNZjkji932GkYf4Pja1CxhpopQnF"},
		{"tYbcQrwHHjcd3n4pKGkxmCnjtw3nPoBs8b", false, true, ""},
		{"TYbcQrwHHjcd3n4pKGkxmCnjtw3nPoBs8a", false, true, ""},
		{"tYbcQrwHHjcd3n4pKGkxmCnjtw3nPoBs8a", false, true, ""},
		{"TYbcQrwHHjcd3n4pKGkxmCn", false, true, ""},
		{"TybcQrwHHjcd3n4pKGkxmCnjtw3nPoBs8b", false, true, ""},
		{"TRGvtQpC8cpk1ksbwLn7xr5DK7aYZgmWcA", true, true, "TRGvtQpC8cpk1ksbwLn7xr5DK7aYZgmWcA"},
		{"1000315", true, false, "1000315"},
		//should add base58 check ok, but other failed cases
	}

	for _, data := range testdata {
		req := &proto.ValidAddressRequest{
			Chain:   ChainName,
			Symbol:  TronSymbol,
			Address: data.address,
		}

		res, err := tronChainAdaptor.ValidAddress(req)
		if data.isValid {
			require.Nil(t, err)
		} else {
			require.NotNil(t, err)
		}

		require.Equal(t, data.isValid, res.Valid)
		if res.Valid {
			require.Equal(t, data.stdAddress, res.CanonicalAddress)
			require.Equal(t, data.canWithdrawal, res.CanWithdrawal)
		}
	}
}

func TestQueryTrxBalance(t *testing.T) {
	req := &proto.QueryBalanceRequest{
		Chain:   ChainName,
		Symbol:  TronSymbol,
		Address: "TYbcQrwHHjcd3n4pKGkxmCnjtw3nPoBs8b",
	}

	res, err := tronChainAdaptor.QueryBalance(req)
	require.Nil(t, err)
	require.NotEmpty(t, res.Balance)
}

func TestQueryTrc10Balance(t *testing.T) {
	req := &proto.QueryBalanceRequest{
		Chain:   ChainName,
		Symbol:  "1000315",
		Address: "TYbcQrwHHjcd3n4pKGkxmCnjtw3nPoBs8b",
	}

	res, err := tronChainAdaptor.QueryBalance(req)
	require.Nil(t, err)
	require.NotEmpty(t, res.Balance)
	//t.Logf("res:%v", res)
}

func TestQueryTrc20Balance(t *testing.T) {
	req := &proto.QueryBalanceRequest{
		Chain:           ChainName,
		Symbol:          "USDT",
		Address:         "TYbcQrwHHjcd3n4pKGkxmCnjtw3nPoBs8b",
		ContractAddress: "TU4oHpbNZjkji932GkYf4Pja1CxhpopQnF",
	}

	res, err := tronChainAdaptor.QueryBalance(req)
	require.Nil(t, err)
	require.NotEmpty(t, res.Balance)
	//t.Logf("res:%v", res)
}

//func TestQueryTx(t *testing.T) {
//	req := &proto.QueryTransactionRequest{
//		Chain:  ChainName,
//		Symbol: TronSymbol,
//		TxHash: "b6a64168c325ddb5715c7e4a44ea49c3998809699b61c3aa9b906e98fe4c6443",
//	}
//
//	res, err := tronChainAdaptor.QueryAccountTransaction(req)
//	require.Nil(t, err)
//	t.Logf("res:%v", res)
//}
//
//func TestQueryTrc10Tx(t *testing.T) {
//	req := &proto.QueryTransactionRequest{
//		Chain:  ChainName,
//		Symbol: "1002000",
//		TxHash: "7f8a7107f075cf9ea7e17a5279da0bf8f0addf88e46bfdd1c647f0475956efb1",
//	}
//
//	res, err := tronChainAdaptor.QueryAccountTransaction(req)
//	require.Nil(t, err)
//	t.Logf("res:%v", res)
//}

func TestQueryTrc20Tx(t *testing.T) {
	from := "TYbcQrwHHjcd3n4pKGkxmCnjtw3nPoBs8b"
	to := "TJbsQjACJqnU5ZRaMWhGcWvi7uiZ6wpJto"
	contract := "TU4oHpbNZjkji932GkYf4Pja1CxhpopQnF"
	amount := "2000000"
	gasLimit := "1000000"
	gasPrice := "1"

	hash1 := "94adbf2a03d4d10d1582fd315e30f1ffd57eaa6d769f43cb1f0f26d813b71bce"
	req1 := &proto.QueryTransactionRequest{
		Chain:  ChainName,
		Symbol: "USDT",
		TxHash: hash1,
	}

	reply1, err := tronChainAdaptor.QueryAccountTransaction(req1)
	//t.Logf("reply1:%v", reply1)

	require.Nil(t, err)
	require.Equal(t, hash1, reply1.TxHash)
	require.Equal(t, proto.ReturnCode_SUCCESS, reply1.Code)
	require.Equal(t, proto.TxStatus_Success, reply1.TxStatus)
	require.Equal(t, gasLimit, reply1.GasLimit)
	require.Equal(t, gasPrice, reply1.GasPrice)
	require.Equal(t, "137280", reply1.CostFee)
	require.EqualValues(t, uint64(7215183), reply1.BlockHeight)
	require.EqualValues(t, uint64(1597733904000), reply1.BlockTime)
	require.Equal(t, from, reply1.From)
	require.Equal(t, to, reply1.To)
	require.Equal(t, contract, reply1.ContractAddress)
	require.Equal(t, amount, reply1.Amount)

	//failed trc20 transfer
	hash2 := "ff7b5648bc1f7e1be25053460cb83b0d8d1368a8c805b43a4b95b71adc804c2e"
	req2 := &proto.QueryTransactionRequest{
		Chain:  ChainName,
		Symbol: "USDT",
		TxHash: hash2,
	}

	reply2, err := tronChainAdaptor.QueryAccountTransaction(req2)
	//t.Logf("reply2:%v", reply2)

	require.Nil(t, err)
	require.Equal(t, hash2, reply2.TxHash)
	require.Equal(t, proto.ReturnCode_SUCCESS, reply2.Code)
	require.Equal(t, proto.TxStatus_Failed, reply2.TxStatus)
	require.Equal(t, "10000", reply2.GasLimit) //change gas limit for
	require.Equal(t, gasPrice, reply2.GasPrice)
	require.Equal(t, "13430", reply2.CostFee)
	require.EqualValues(t, uint64(7215097), reply2.BlockHeight)
	require.EqualValues(t, uint64(1597733640000), reply2.BlockTime)
	require.Equal(t, "", reply2.From)
	require.Equal(t, "", reply2.To)
	require.Equal(t, "", reply2.ContractAddress)
	require.Equal(t, "", reply2.Amount)
}

func TestCreateSingedBrocastTrcSendTransaction(t *testing.T) {
	from := "TYbcQrwHHjcd3n4pKGkxmCnjtw3nPoBs8b"
	to := "TEgKVx5j3htRKgjjtLBroh2eoDq5eeDJ6W"
	amount := "100000000"
	gasLimit := "1000000"

	req1 := &proto.CreateAccountTransactionRequest{
		Chain:    ChainName,
		Symbol:   TronSymbol,
		From:     from,
		To:       to,
		Amount:   amount,
		GasPrice: "1",
		GasLimit: gasLimit,
	}

	reply1, err := tronChainAdaptor.CreateAccountTransaction(req1)
	require.Nil(t, err)
	require.NotNil(t, reply1)
	t.Logf("tx Data:%v", hex.EncodeToString(reply1.TxData))
	t.Logf("res:%v", hex.EncodeToString(reply1.SignHash))

	hash := reply1.SignHash
	key := []byte("key0")
	priv, pub := btcec.PrivKeyFromBytes(btcec.S256(), key)

	addr := address.PubkeyToAddress(*pub.ToECDSA())
	require.Equal(t, "TYbcQrwHHjcd3n4pKGkxmCnjtw3nPoBs8b", addr.String())
	t.Logf("hash:%v\n", hex.EncodeToString(hash))

	//tron original signature
	sig1, err := crypto.Sign(hash, (*ecdsa.PrivateKey)(priv))
	require.Nil(t, err)

	req2 := &proto.CreateAccountSignedTransactionRequest{
		Chain:     ChainName,
		Symbol:    TronSymbol,
		TxData:    reply1.TxData,
		Signature: sig1,
		PublicKey: pub.SerializeCompressed(),
	}

	reply2, err := tronChainAdaptor.CreateAccountSignedTransaction(req2)
	require.Equal(t, reply1.SignHash, reply2.Hash)
	require.Nil(t, err)
	t.Logf("signed Tx Data:%v", hex.EncodeToString(reply2.SignedTxData))
	t.Logf("res:%v", hex.EncodeToString(reply2.Hash))

	req3 := &proto.VerifySignedTransactionRequest{
		Chain:        ChainName,
		Symbol:       TronSymbol,
		Sender:       "TYbcQrwHHjcd3n4pKGkxmCnjtw3nPoBs8b",
		SignedTxData: reply2.SignedTxData,
	}

	reply3, err := tronChainAdaptor.VerifyAccountSignedTransaction(req3)
	require.Equal(t, true, reply3.Verified)

	req4 := &proto.BroadcastTransactionRequest{
		Chain:        ChainName,
		Symbol:       TronSymbol,
		SignedTxData: reply2.SignedTxData,
	}

	reply4, err := tronChainAdaptor.BroadcastTransaction(req4)
	require.Equal(t, proto.ReturnCode_SUCCESS, reply4.Code)
	require.Equal(t, hex.EncodeToString(hash), reply4.TxHash)

	req5 := &proto.QueryTransactionFromDataRequest{
		Chain:   ChainName,
		Symbol:  TronSymbol,
		RawData: reply1.TxData,
	}

	reply5, err := tronChainAdaptor.QueryAccountTransactionFromData(req5)
	require.Equal(t, from, reply5.From)
	require.Equal(t, to, reply5.To)
	require.Equal(t, amount, reply5.Amount)
	require.Equal(t, hex.EncodeToString(reply1.SignHash), reply5.TxHash)
	require.Equal(t, reply1.SignHash, reply5.SignHash)
	require.Equal(t, "1", reply5.GasPrice)

	req6 := &proto.QueryTransactionFromSignedDataRequest{
		Chain:        ChainName,
		Symbol:       TronSymbol,
		SignedTxData: reply2.SignedTxData,
	}

	reply6, err := tronChainAdaptor.QueryAccountTransactionFromSignedData(req6)
	require.Equal(t, from, reply6.From)
	require.Equal(t, to, reply6.To)
	require.Equal(t, amount, reply6.Amount)
	require.Equal(t, hex.EncodeToString(reply1.SignHash), reply6.TxHash)
	require.Equal(t, reply1.SignHash, reply6.SignHash)
	require.Equal(t, "1", reply6.GasPrice)

	time.Sleep(5 * time.Second)
	req7 := &proto.QueryTransactionRequest{
		Chain:  ChainName,
		Symbol: TronSymbol,
		TxHash: hex.EncodeToString(hash),
	}

	reply7, err := tronChainAdaptor.QueryAccountTransaction(req7)
	require.Nil(t, err)
	require.Equal(t, from, reply7.From)
	require.Equal(t, to, reply7.To)
	require.Equal(t, amount, reply7.Amount)
	require.Equal(t, hex.EncodeToString(reply1.SignHash), reply7.TxHash)
	require.Equal(t, "1", reply6.GasPrice)

}

func TestCreateSingedBrocastTrc10SendTransaction(t *testing.T) {
	from := "TYbcQrwHHjcd3n4pKGkxmCnjtw3nPoBs8b"
	to := "TEgKVx5j3htRKgjjtLBroh2eoDq5eeDJ6W"
	amount := "110000000"
	gas := "10000"
	symbol := "trx10usdt"
	contractAddress := "1000315"

	req1 := &proto.CreateAccountTransactionRequest{
		Chain:           ChainName,
		Symbol:          symbol,
		From:            from,
		To:              to,
		Amount:          amount,
		GasPrice:        "1",
		GasLimit:        gas,
		ContractAddress: contractAddress,
	}

	reply1, err := tronChainAdaptor.CreateAccountTransaction(req1)
	require.Nil(t, err)

	hash := reply1.SignHash
	key := []byte("key0")
	priv, pub := btcec.PrivKeyFromBytes(btcec.S256(), key)

	addr := address.PubkeyToAddress(*pub.ToECDSA())
	require.Equal(t, from, addr.String())
	t.Logf("hash:%v\n", hex.EncodeToString(hash))

	//tron original signature
	sig1, err := crypto.Sign(hash, (*ecdsa.PrivateKey)(priv))
	require.Nil(t, err)

	req2 := &proto.CreateAccountSignedTransactionRequest{
		Chain:     ChainName,
		Symbol:    symbol,
		TxData:    reply1.TxData,
		Signature: sig1,
		PublicKey: pub.SerializeCompressed(),
	}

	reply2, err := tronChainAdaptor.CreateAccountSignedTransaction(req2)
	require.Equal(t, reply1.SignHash, reply2.Hash)
	require.Nil(t, err)
	//t.Logf("res:%v", hex.EncodeToString(reply2.SignedTxData))
	//t.Logf("res:%v", hex.EncodeToString(reply2.Hash))

	req3 := &proto.VerifySignedTransactionRequest{
		Chain:        ChainName,
		Symbol:       symbol,
		Sender:       from,
		SignedTxData: reply2.SignedTxData,
	}

	reply3, err := tronChainAdaptor.VerifyAccountSignedTransaction(req3)
	require.Equal(t, true, reply3.Verified)

	req4 := &proto.BroadcastTransactionRequest{
		Chain:        ChainName,
		Symbol:       symbol,
		SignedTxData: reply2.SignedTxData,
	}

	reply4, err := tronChainAdaptor.BroadcastTransaction(req4)
	require.Equal(t, proto.ReturnCode_SUCCESS, reply4.Code)
	require.Equal(t, hex.EncodeToString(hash), reply4.TxHash)

	req5 := &proto.QueryTransactionFromDataRequest{
		Chain:   ChainName,
		Symbol:  symbol,
		RawData: reply1.TxData,
	}

	reply5, err := tronChainAdaptor.QueryAccountTransactionFromData(req5)
	require.Equal(t, from, reply5.From)
	require.Equal(t, to, reply5.To)
	require.Equal(t, amount, reply5.Amount)
	require.Equal(t, hex.EncodeToString(reply1.SignHash), reply5.TxHash)
	require.Equal(t, reply1.SignHash, reply5.SignHash)
	require.Equal(t, "1", reply5.GasPrice)
	require.Equal(t, contractAddress, reply5.ContractAddress)

	req6 := &proto.QueryTransactionFromSignedDataRequest{
		Chain:        ChainName,
		Symbol:       symbol,
		SignedTxData: reply2.SignedTxData,
	}

	reply6, err := tronChainAdaptor.QueryAccountTransactionFromSignedData(req6)
	require.Equal(t, from, reply6.From)
	require.Equal(t, to, reply6.To)
	require.Equal(t, amount, reply6.Amount)
	require.Equal(t, hex.EncodeToString(reply1.SignHash), reply6.TxHash)
	require.Equal(t, reply1.SignHash, reply6.SignHash)
	require.Equal(t, "1", reply6.GasPrice)
	require.Equal(t, contractAddress, reply6.ContractAddress)

	time.Sleep(5 * time.Second)
	req7 := &proto.QueryTransactionRequest{
		Chain:  ChainName,
		Symbol: symbol,
		TxHash: hex.EncodeToString(hash),
	}

	reply7, err := tronChainAdaptor.QueryAccountTransaction(req7)
	require.Nil(t, err)
	require.Equal(t, from, reply7.From)
	require.Equal(t, to, reply7.To)
	require.Equal(t, amount, reply7.Amount)
	require.Equal(t, hex.EncodeToString(reply1.SignHash), reply7.TxHash)
	require.Equal(t, "1", reply7.GasPrice)
	require.Equal(t, contractAddress, reply7.ContractAddress)

}

func TestBigAmountTrc20TransactionEncoding(t *testing.T) {
	from := "TYbcQrwHHjcd3n4pKGkxmCnjtw3nPoBs8b"
	to := "TEgKVx5j3htRKgjjtLBroh2eoDq5eeDJ6W"
	contract := "TU4oHpbNZjkji932GkYf4Pja1CxhpopQnF"
	amount := "200000000000000000000000000"
	gasLimit := "1000000"
	symbol := "trx20usdt"

	req1 := &proto.CreateAccountTransactionRequest{
		Chain:           ChainName,
		Symbol:          symbol,
		From:            from,
		To:              to,
		ContractAddress: contract,
		Amount:          amount,
		GasPrice:        "1",
		GasLimit:        gasLimit,
	}

	reply, err := tronChainAdaptor.CreateAccountTransaction(req1)
	require.Nil(t, err)

	req := &proto.QueryTransactionFromDataRequest{
		Chain:   ChainName,
		Symbol:  symbol,
		RawData: reply.TxData,
	}

	res, err := tronChainAdaptor.QueryAccountTransactionFromData(req)
	require.Nil(t, err)
	require.Equal(t, from, res.From)
	require.Equal(t, to, res.To)
	require.Equal(t, contract, res.ContractAddress)
	require.Equal(t, amount, res.Amount)
	require.Equal(t, gasLimit, res.GasLimit)
	require.Equal(t, "1", res.GasPrice)
}

func TestCreateSingedBrocastTrc20SendTransactionSuccess(t *testing.T) {
	from := "TYbcQrwHHjcd3n4pKGkxmCnjtw3nPoBs8b"
	to := "TEgKVx5j3htRKgjjtLBroh2eoDq5eeDJ6W"
	contract := "TU4oHpbNZjkji932GkYf4Pja1CxhpopQnF"
	amount := "200000000"
	gasLimit := "1000000" //big.NewInt(defultGasLimit).String()
	symbol := "trx20usdt"

	req1 := &proto.CreateAccountTransactionRequest{
		Chain:           ChainName,
		Symbol:          symbol,
		From:            from,
		To:              to,
		ContractAddress: contract,
		Amount:          amount,
		GasPrice:        "1",
		GasLimit:        gasLimit,
	}

	reply1, err := tronChainAdaptor.CreateAccountTransaction(req1)
	require.Nil(t, err)

	hash := reply1.SignHash
	key := []byte("key0")
	priv, pub := btcec.PrivKeyFromBytes(btcec.S256(), key)

	addr := address.PubkeyToAddress(*pub.ToECDSA())
	require.Equal(t, from, addr.String())
	t.Logf("hash:%v\n", hex.EncodeToString(hash))

	//tron original signature
	sig1, err := crypto.Sign(hash, (*ecdsa.PrivateKey)(priv))
	require.Nil(t, err)

	req2 := &proto.CreateAccountSignedTransactionRequest{
		Chain:     ChainName,
		Symbol:    symbol,
		TxData:    reply1.TxData,
		Signature: sig1,
		PublicKey: pub.SerializeCompressed(),
	}

	reply2, err := tronChainAdaptor.CreateAccountSignedTransaction(req2)
	require.Equal(t, reply1.SignHash, reply2.Hash)
	require.Nil(t, err)

	req3 := &proto.VerifySignedTransactionRequest{
		Chain:        ChainName,
		Symbol:       symbol,
		Sender:       from,
		SignedTxData: reply2.SignedTxData,
	}

	reply3, err := tronChainAdaptor.VerifyAccountSignedTransaction(req3)
	require.Equal(t, true, reply3.Verified)

	req4 := &proto.BroadcastTransactionRequest{
		Chain:        ChainName,
		Symbol:       symbol,
		SignedTxData: reply2.SignedTxData,
	}

	reply4, err := tronChainAdaptor.BroadcastTransaction(req4)
	require.Equal(t, proto.ReturnCode_SUCCESS, reply4.Code)
	require.Equal(t, hex.EncodeToString(hash), reply4.TxHash)

	req5 := &proto.QueryTransactionFromDataRequest{
		Chain:   ChainName,
		Symbol:  symbol,
		RawData: reply1.TxData,
	}

	reply5, err := tronChainAdaptor.QueryAccountTransactionFromData(req5)
	require.Equal(t, from, reply5.From)
	require.Equal(t, to, reply5.To)
	require.Equal(t, amount, reply5.Amount)
	require.Equal(t, hex.EncodeToString(reply1.SignHash), reply5.TxHash)
	require.Equal(t, reply1.SignHash, reply5.SignHash)
	require.Equal(t, "1", reply5.GasPrice)
	require.Equal(t, contract, reply5.ContractAddress)

	req6 := &proto.QueryTransactionFromSignedDataRequest{
		Chain:        ChainName,
		Symbol:       symbol,
		SignedTxData: reply2.SignedTxData,
	}

	reply6, err := tronChainAdaptor.QueryAccountTransactionFromSignedData(req6)
	require.Equal(t, from, reply6.From)
	require.Equal(t, to, reply6.To)
	require.Equal(t, amount, reply6.Amount)
	require.Equal(t, hex.EncodeToString(reply1.SignHash), reply6.TxHash)
	require.Equal(t, reply1.SignHash, reply6.SignHash)
	require.Equal(t, "1", reply6.GasPrice)
	require.Equal(t, contract, reply6.ContractAddress)

	time.Sleep(10 * time.Second)
	req7 := &proto.QueryTransactionRequest{
		Chain:  ChainName,
		Symbol: symbol,
		TxHash: reply6.TxHash,
	}

	reply7, err := tronChainAdaptor.QueryAccountTransaction(req7)
	require.Nil(t, err)
	require.Equal(t, hex.EncodeToString(hash), reply7.TxHash)
	require.Equal(t, proto.ReturnCode_SUCCESS, reply7.Code)
	require.Equal(t, proto.TxStatus_Success, reply7.TxStatus)
	require.Equal(t, gasLimit, reply7.GasLimit)
	require.Equal(t, "1", reply7.GasPrice)
	require.Equal(t, from, reply7.From)
	require.Equal(t, to, reply7.To)
	require.Equal(t, contract, reply7.ContractAddress)
	require.Equal(t, amount, reply7.Amount)

}

//
//func TestCreateSingedBrocastTrc20SendTransactionFailOutOfEnergy(t *testing.T) {
//	from := "TYbcQrwHHjcd3n4pKGkxmCnjtw3nPoBs8b"
//	to := "TJbsQjACJqnU5ZRaMWhGcWvi7uiZ6wpJto"
//	contract := "TU4oHpbNZjkji932GkYf4Pja1CxhpopQnF"
//	amount := "2000000"
//	gasLimit := "10000" //big.NewInt(defultGasLimit).String(), if gasLimit < energyused, trx will fail, if
//	symbol := "USDT"
//
//	req1 := &proto.CreateAccountTransactionRequest{
//		Chain:           ChainName,
//		Symbol:          symbol,
//		From:            from,
//		To:              to,
//		ContractAddress: contract,
//		Amount:          amount,
//		GasPrice:        "1",
//		GasLimit:        gasLimit,
//	}
//
//	reply1, err := tronChainAdaptor.CreateAccountTransaction(req1)
//	require.Nil(t, err)
//
//	hash := reply1.SignHash
//	key := []byte("key0")
//	priv, pub := btcec.PrivKeyFromBytes(btcec.S256(), key)
//
//	addr := address.PubkeyToAddress(*pub.ToECDSA())
//	require.Equal(t, from, addr.String())
//	t.Logf("hash:%v\n", hex.EncodeToString(hash))
//
//	//tron original signature
//	sig1, err := crypto.Sign(hash, (*ecdsa.PrivateKey)(priv))
//	require.Nil(t, err)
//
//	req2 := &proto.CreateAccountSignedTransactionRequest{
//		Chain:     ChainName,
//		Symbol:    symbol,
//		TxData:    reply1.TxData,
//		Signature: sig1,
//		PublicKey: pub.SerializeCompressed(),
//	}
//
//	reply2, err := tronChainAdaptor.CreateAccountSignedTransaction(req2)
//	require.Equal(t, reply1.SignHash, reply2.Hash)
//	require.Nil(t, err)
//
//	req3 := &proto.VerifySignedTransactionRequest{
//		Chain:        ChainName,
//		Symbol:       symbol,
//		Sender:       from,
//		SignedTxData: reply2.SignedTxData,
//	}
//
//	reply3, err := tronChainAdaptor.VerifyAccountSignedTransaction(req3)
//	require.Equal(t, true, reply3.Verified)
//
//	req4 := &proto.BroadcastTransactionRequest{
//		Chain:        ChainName,
//		Symbol:       symbol,
//		SignedTxData: reply2.SignedTxData,
//	}
//
//	reply4, err := tronChainAdaptor.BroadcastTransaction(req4)
//	require.Equal(t, proto.ReturnCode_SUCCESS, reply4.Code)
//	require.Equal(t, hex.EncodeToString(hash), reply4.TxHash)
//
//	req5 := &proto.QueryTransactionFromDataRequest{
//		Chain:   ChainName,
//		Symbol:  symbol,
//		RawData: reply1.TxData,
//	}
//
//	reply5, err := tronChainAdaptor.QueryAccountTransactionFromData(req5)
//	require.Equal(t, from, reply5.From)
//	require.Equal(t, to, reply5.To)
//	require.Equal(t, amount, reply5.Amount)
//	require.Equal(t, hex.EncodeToString(reply1.SignHash), reply5.TxHash)
//	require.Equal(t, reply1.SignHash, reply5.SignHash)
//	require.Equal(t, "1", reply5.GasPrice)
//	require.Equal(t, contract, reply5.ContractAddress)
//
//	req6 := &proto.QueryTransactionFromSignedDataRequest{
//		Chain:        ChainName,
//		Symbol:       symbol,
//		SignedTxData: reply2.SignedTxData,
//	}
//
//	reply6, err := tronChainAdaptor.QueryAccountTransactionFromSignedData(req6)
//	require.Equal(t, from, reply6.From)
//	require.Equal(t, to, reply6.To)
//	require.Equal(t, amount, reply6.Amount)
//	require.Equal(t, hex.EncodeToString(reply1.SignHash), reply6.TxHash)
//	require.Equal(t, reply1.SignHash, reply6.SignHash)
//	require.Equal(t, "1", reply6.GasPrice)
//	require.Equal(t, contract, reply6.ContractAddress)
//
//	time.Sleep(10 * time.Second)
//	req7 := &proto.QueryTransactionRequest{
//		Chain:  ChainName,
//		Symbol: symbol,
//		TxHash: reply6.TxHash,
//	}
//
//	reply7, err := tronChainAdaptor.QueryAccountTransaction(req7)
//	require.Nil(t, err)
//	require.Equal(t, hex.EncodeToString(hash), reply7.TxHash)
//	require.Equal(t, proto.ReturnCode_SUCCESS, reply7.Code)
//	require.Equal(t, proto.TxStatus_Failed, reply7.TxStatus)
//	require.Equal(t, gasLimit, reply7.GasLimit)
//	//require.Equal(t, "13430", reply7.CostFee) //"13430" maybe change
//	require.Equal(t, "1", reply7.GasPrice)
//	require.Equal(t, from, reply7.From)
//	require.Equal(t, to, reply7.To)
//	require.Equal(t, contract, reply7.ContractAddress)
//	require.Equal(t, amount, reply7.Amount)
//}

func TestCreateAccountSignedTransaction(t *testing.T) {
	txData, err := hex.DecodeString("0a02655922083256b74f1d4d5a8440f0a9ebdcc12e5a69080112650a2d747970652e676f6f676c65617069732e636f6d2f70726f746f636f6c2e5472616e73666572436f6e747261637412340a1541a7e3ec19ab1c18a2a3f6d8d22a8a75175da4c8d312154192a8844d0610275b9e21270f39adcf23e5bdfb2e18c0bc90e10370b1eee7dcc12e")
	require.Nil(t, err)
	sig, err := hex.DecodeString("37981bfa2280621b4309dcad61c956a1f58683242a5a9915e6145afecbcb803424cdc6b76ac95596be3a9fcbe2ea3d4a262f5f45b7682652b9b675dfffb5d7fe01")
	require.Nil(t, err)
	pubKey, err := hex.DecodeString("030bef4b4ce79dc7576be276028b28201f423e108e5667ce62e515affe002bf88b")
	require.Nil(t, err)

	req := &proto.CreateAccountSignedTransactionRequest{
		Chain:     ChainName,
		Symbol:    TronSymbol,
		TxData:    txData,
		Signature: sig,
		PublicKey: pubKey,
	}

	reply, err := tronChainAdaptor.CreateAccountSignedTransaction(req)
	require.Equal(t, getHash(txData), reply.Hash)
	require.Nil(t, err)

}

func TestVerifyAccountBasedSignedTx(t *testing.T) {
	signedTxData, err := hex.DecodeString("0a87010a02655922083256b74f1d4d5a8440f0a9ebdcc12e5a69080112650a2d747970652e676f6f676c65617069732e636f6d2f70726f746f636f6c2e5472616e73666572436f6e747261637412340a1541a7e3ec19ab1c18a2a3f6d8d22a8a75175da4c8d312154192a8844d0610275b9e21270f39adcf23e5bdfb2e18c0bc90e10370b1eee7dcc12e124137981bfa2280621b4309dcad61c956a1f58683242a5a9915e6145afecbcb803424cdc6b76ac95596be3a9fcbe2ea3d4a262f5f45b7682652b9b675dfffb5d7fe01")
	require.Nil(t, err)
	req := &proto.VerifySignedTransactionRequest{
		Chain:        ChainName,
		Symbol:       TronSymbol,
		Sender:       "TRGvtQpC8cpk1ksbwLn7xr5DK7aYZgmWcA",
		SignedTxData: signedTxData,
	}

	reply, err := tronChainAdaptor.VerifyAccountSignedTransaction(req)
	require.True(t, reply.Verified)
	require.Nil(t, err)
}

func TestGetAccountTransactionByHeight(t *testing.T) {
	errCh := make(chan error, 20)
	replyCh := make(chan *proto.QueryAccountTransactionReply, 20)

	from := "TYbcQrwHHjcd3n4pKGkxmCnjtw3nPoBs8b"
	to := "TJbsQjACJqnU5ZRaMWhGcWvi7uiZ6wpJto"

	//Query TRX transfer
	tronChainAdaptor.GetAccountTransactionByHeight(7239271, replyCh, errCh)
	require.Equal(t, 2, len(replyCh))
	tx := <-replyCh
	require.Equal(t, from, tx.From)
	require.Equal(t, to, tx.To)

	tx = <-replyCh
	require.Equal(t, from, tx.From)
	require.Equal(t, to, tx.To)

	//Query TRC10 transfer
	tronChainAdaptor.GetAccountTransactionByHeight(7239293, replyCh, errCh)
	require.Equal(t, 1, len(replyCh))
	tx = <-replyCh
	require.Equal(t, from, tx.From)
	require.Equal(t, to, tx.To)
	require.Equal(t, "c54a18d6854b2fc27ccac980cfbf96554b48f0526761b570089a81e79d14b9cc", tx.TxHash)

	//Query TRC20 transfer
	tronChainAdaptor.GetAccountTransactionByHeight(7239272, replyCh, errCh)
	require.Equal(t, 2, len(replyCh))

}

func TestGetAccountTransactionByHeight2(t *testing.T) {
	errCh := make(chan error, 20)
	replyCh := make(chan *proto.QueryAccountTransactionReply, 20)

	//Query TRX transfer, there are 2 TRC20 tx,  both fail, but no error is expected
	tronChainAdaptor.GetAccountTransactionByHeight(7395496, replyCh, errCh)
	require.Equal(t, 0, len(replyCh))
	require.Equal(t, 0, len(errCh))

}

func TestGetAccountTransactionByHeight3(t *testing.T) {
	errCh := make(chan error, 20)
	replyCh := make(chan *proto.QueryAccountTransactionReply, 20)

	//Query TRX transfer, there are 2 TRC20 tx,  both fail, but no error is expected
	tronChainAdaptor.GetAccountTransactionByHeight(7443104, replyCh, errCh)
	require.Equal(t, 1, len(replyCh))
	require.Equal(t, 0, len(errCh))

}

func TestGetAccountTransactionByHeight4(t *testing.T) {
	errCh := make(chan error, 20)
	replyCh := make(chan *proto.QueryAccountTransactionReply, 20)

	//Query TRX transfer, there are 2 TRC20 tx,  both fail, but no error is expected
	tronChainAdaptor.GetAccountTransactionByHeight(7444666, replyCh, errCh)
	require.Equal(t, 1, len(replyCh))
	require.Equal(t, 0, len(errCh))

}
