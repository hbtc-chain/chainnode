package bitcoin

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hbtc-chain/chainnode/cache"
	"github.com/hbtc-chain/chainnode/config"
	"github.com/hbtc-chain/chainnode/proto"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcutil"
	"github.com/stretchr/testify/assert"
)

var conf *config.Config

func TestMain(m *testing.M) {
	var err error
	conf, err = config.New("testnet.yaml")
	if err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

func newChainAdaptorWithConfig(conf *config.Config) *ChainAdaptor {
	chainAdaptor, err := NewChainAdaptor(conf)
	if err != nil {
		panic(err)
	}
	return chainAdaptor.(*ChainAdaptor)
}

func TestBtcToSatoShi2(t *testing.T) {
	input := []float64{1, 0.2, 0.00013, 400, 0.00000067, 8900000000.0}
	output := []int64{100000000, 20000000, 13000, 40000000000, 67, 890000000000000000}

	for i, a := range input {
		s := btcToSatoshi(a)
		assert.Equal(t, big.NewInt(output[i]), s)

	}
}

type Key2Addr struct {
	privKey  []byte
	pubKey   []byte
	mainAddr string // mainnet address derived from pubkey
	testAddr string // testnet address derived from pukbey
}

var keyAddrComb []Key2Addr

func genPub2Addr() {

	for i := 0; i < 10; i++ {
		key := []byte(fmt.Sprintf("key%v", i))
		btcecPriKey, btcecPublicKey := btcec.PrivKeyFromBytes(btcec.S256(), key)
		btcMainAddress, err := btcutil.NewAddressPubKey(btcecPublicKey.SerializeCompressed(), &chaincfg.MainNetParams)
		if err != nil {
			panic(1)
		}
		btcTestAddress, err := btcutil.NewAddressPubKey(btcecPublicKey.SerializeCompressed(), &chaincfg.TestNet3Params)
		if err != nil {
			panic(1)
		}

		keyAddr := Key2Addr{
			privKey:  btcecPriKey.Serialize(),
			pubKey:   btcecPublicKey.SerializeCompressed(),
			mainAddr: btcMainAddress.EncodeAddress(),
			testAddr: btcTestAddress.EncodeAddress(),
		}

		keyAddrComb = append(keyAddrComb, keyAddr)
	}
}

func TestConvertAddress(t *testing.T) {
	btcChainAdaptor := newChainAdaptorWithConfig(conf)

	genPub2Addr()
	client := btcChainAdaptor.client

	var req proto.ConvertAddressRequest
	req.Chain = ChainName

	for _, a := range keyAddrComb {
		req.PublicKey = a.pubKey
		reply, err := btcChainAdaptor.ConvertAddress(&req)
		assert.Nil(t, err)
		assert.Equal(t, a.testAddr, reply.Address)
	}

	// change to mainnet params
	client.chainConfig = &chaincfg.MainNetParams
	for _, a := range keyAddrComb {
		req.PublicKey = a.pubKey
		reply, err := btcChainAdaptor.ConvertAddress(&req)
		assert.Nil(t, err)
		assert.Equal(t, a.mainAddr, reply.Address)
	}
}

// Get 2 privatekey and address for test
func TestConvertAddress2(t *testing.T) {
	genPub2Addr()

	/*
		priv1:cMahea7zqjxrtgAbB7LSGbcQUr1uX1ojuat9jZoeiemEvoHeHdkv
		addr1:n3HsmPMAEa2ovEzMNrXKZhSMGvUEfBpHcd
		priv2:cMahea7zqjxrtgAbB7LSGbcQUr1uX1ojuat9jZoeiemEwJD8z6eD
		addr2: addr:mwF4uCj81VAqKBDjmV2yLESREGDCyZL6Ap
	*/
	for i := 0; i < 2; i++ {

		privateKey, pubKey := btcec.PrivKeyFromBytes(btcec.S256(), keyAddrComb[i].privKey)

		_, err := btcutil.NewWIF(privateKey, &chaincfg.TestNet3Params, true)
		assert.Nil(t, err)
		//	t.Logf("wif:%v", wif.String())

		_, err = btcutil.NewAddressPubKey(pubKey.SerializeCompressed(), &chaincfg.TestNet3Params)
		assert.Nil(t, err)
		// t.Logf("addr:%v", addr.EncodeAddress())
	}

}

var mainnetAddrs = []string{
	"16ftSEQ4ctQFDtVZiUBusQUjRrGhM3JYwe",
	"3MN2Yy9y3tEnNH11CpsanHRRDjuczBMrpJ",
	"bc1qrp33g0q5c5txsp9arysrx4k6zdkfs4nce4xj0gdcccefvpysxf3qccfmv3",
}

var testnetAddrs = []string{
	"mnRw8TRyxUVEv1CnfzpahuRr5BeWYsCGES",
	"tb1qw508d6qejxtdg4y5r3zarvary0c5xw7kxpjzsx",
	"tb1qrp33g0q5c5txsp9arysrx4k6zdkfs4nce4xj0gdcccefvpysxf3q0sl5k7",
	"2NCvEci5zfLk8a4dYsxVTQEQgS67nrtv4Wn",
	"2Mww6tED1opzwN2D3rqqKW9z6BdLstdRFpL",
}

var illegalAddrs = []string{
	"bc1qw508d6qejxtdg4y5r3zarvary0c5xw7kv8f3t5",
	"16ftSEQ4ctQFDtVZiUBusQUjRrGhM3KYwe",
	"MnRw8TRyxUVEv1CnfzpahuRr5BeWYsCGES",
	"Tb1qw508d6qejxtdg4y5r3zarvary0c5xw7kxpjzsx",
	"Tb1qrp33g0q5c5txsp9arysrx4k6zdkfs4nce4xj0gdcccefvpysxf3q0sl5k7",
	"2nCvEci5zfLk8a4dYsxVTQEQgS67nrtv4Wn",
	"2mww6tED1opzwN2D3rqqKW9z6BdLstdRFpL",
}

func TestValidAddress(t *testing.T) {
	btcChainAdaptor := newChainAdaptorWithConfig(conf)

	client := btcChainAdaptor.client
	genPub2Addr()

	var req proto.ValidAddressRequest
	req.Chain = ChainName
	req.Symbol = Symbol

	// testnet paramater
	for _, a := range keyAddrComb {
		req.Address = a.testAddr
		reply, err := btcChainAdaptor.ValidAddress(&req)
		assert.Nil(t, err)
		assert.Equal(t, true, reply.Valid)
		assert.Equal(t, true, reply.CanWithdrawal)
		assert.Equal(t, a.testAddr, reply.CanonicalAddress)

		req.Address = a.mainAddr
		_, err = btcChainAdaptor.ValidAddress(&req)
		assert.NotNil(t, err)
	}

	for _, a := range mainnetAddrs {
		req.Address = a
		reply, err := btcChainAdaptor.ValidAddress(&req)
		assert.NotNil(t, err)
		assert.Equal(t, proto.ReturnCode_ERROR, reply.Code)
	}

	for _, a := range testnetAddrs {
		req.Address = a
		reply, err := btcChainAdaptor.ValidAddress(&req)
		assert.Nil(t, err)
		assert.Equal(t, true, reply.Valid)
		assert.Equal(t, true, reply.CanWithdrawal)
		assert.Equal(t, a, reply.CanonicalAddress)
	}

	for _, a := range illegalAddrs {
		req.Address = a
		reply, err := btcChainAdaptor.ValidAddress(&req)
		assert.NotNil(t, err)
		assert.Equal(t, proto.ReturnCode_ERROR, reply.Code)
	}

	// mainnet params
	client.chainConfig = &chaincfg.MainNetParams
	for _, a := range keyAddrComb {
		req.Address = a.mainAddr
		reply, err := btcChainAdaptor.ValidAddress(&req)
		assert.Nil(t, err)
		assert.Equal(t, true, reply.Valid)
		assert.Equal(t, true, reply.CanWithdrawal)

		req.Address = a.testAddr
		_, err = btcChainAdaptor.ValidAddress(&req)
		assert.NotNil(t, err)
	}

	for _, a := range mainnetAddrs {
		req.Address = a
		reply, err := btcChainAdaptor.ValidAddress(&req)
		assert.Nil(t, err)
		assert.Equal(t, true, reply.Valid)
		assert.Equal(t, true, reply.CanWithdrawal)
	}

	for _, a := range testnetAddrs {
		req.Address = a
		reply, err := btcChainAdaptor.ValidAddress(&req)
		assert.NotNil(t, err)
		assert.Equal(t, proto.ReturnCode_ERROR, reply.Code)
	}

	for _, a := range illegalAddrs {
		req.Address = a
		reply, err := btcChainAdaptor.ValidAddress(&req)
		assert.NotNil(t, err)
		assert.Equal(t, proto.ReturnCode_ERROR, reply.Code)
	}
}

func TestQueryGasPrice(t *testing.T) {
	btcChainAdaptor := newChainAdaptorWithConfig(conf)

	var req proto.QueryGasPriceRequest
	req.Chain = ChainName

	reply, err := btcChainAdaptor.QueryGasPrice(&req)
	assert.Nil(t, err)
	assert.NotNil(t, reply)
}

func TestQueryUtxo(t *testing.T) {
	btcChainAdaptor := newChainAdaptorWithConfig(conf)

	var req proto.QueryUtxoRequest
	req.Chain = ChainName
	req.Symbol = Symbol

	// success
	utxo := &proto.Vin{Hash: "9ae3c919d84f4b72802de6f4f4aa0d88abcc9fd57315ddf27b8e25f032e4a180", Index: 1, Amount: 85475551, Address: "mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9"}
	req.Vin = utxo
	reply, err := btcChainAdaptor.QueryUtxo(&req)
	assert.Nil(t, err)
	assert.Equal(t, true, reply.Unspent)

	// spent utxo
	utxo = &proto.Vin{Hash: "19d4409350ab3fdf23be52e5526b4d83265a638b77327a579fc131341c0343f2", Index: 0, Amount: 43000, Address: "mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9"}
	req.Vin = utxo
	reply, err = btcChainAdaptor.QueryUtxo(&req)
	assert.NotNil(t, err)
	// assert.Equal(t, true, reply.Result)
	assert.Equal(t, "hash not found", reply.Msg)

	// unexist utxo
	utxo = &proto.Vin{Hash: "1917ded0c5be6523cf2e5bdbd68e55887bd78a9bc17f10cf1222fc30d05d9060", Index: 0, Amount: 98000, Address: "mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9"}
	req.Vin = utxo
	reply, err = btcChainAdaptor.QueryUtxo(&req)
	assert.NotNil(t, err)
	// assert.Equal(t, true, reply.Result)
	assert.Equal(t, "hash not found", reply.Msg)

	// amount mismatch
	utxo = &proto.Vin{Hash: "9ae3c919d84f4b72802de6f4f4aa0d88abcc9fd57315ddf27b8e25f032e4a180", Index: 1, Amount: 85475552, Address: "mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9"}
	req.Vin = utxo
	reply, err = btcChainAdaptor.QueryUtxo(&req)
	assert.NotNil(t, err)
	assert.Equal(t, "amount mismatch", reply.Msg)

	// address mismatch
	utxo = &proto.Vin{Hash: "9ae3c919d84f4b72802de6f4f4aa0d88abcc9fd57315ddf27b8e25f032e4a180", Index: 1, Amount: 85475551, Address: "2MthzQgsQ8Rw8vPMtTsrTdqc9HsWiDHM9VY"}
	req.Vin = utxo
	reply, err = btcChainAdaptor.QueryUtxo(&req)
	assert.NotNil(t, err)
	assert.Equal(t, "address mismatch", reply.Msg)
}

func TestQueryUtxo2(t *testing.T) {
	btcChainAdaptor := newChainAdaptorWithConfig(conf)

	var req proto.QueryUtxoRequest
	req.Chain = ChainName
	req.Symbol = Symbol

	// success
	utxo := &proto.Vin{Hash: "9f96e84aabb2e31432334220bd314738a1a437fdf29c8091dd9386537d350183", Index: 1, Amount: 500000, Address: "n28anUvZ4RvHsUchWETX7MjbwVYziFy94C"}
	req.Vin = utxo
	reply, err := btcChainAdaptor.QueryUtxo(&req)
	assert.Nil(t, err, "unexpected error", err)
	assert.Equal(t, true, reply.Unspent)
	t.Logf("reply:%v", reply)

	var req1 proto.QueryTransactionRequest
	req1.Chain = ChainName
	req1.Symbol = Symbol
	req1.TxHash = "9f96e84aabb2e31432334220bd314738a1a437fdf29c8091dd9386537d350183"

	reply1, err := btcChainAdaptor.QueryUtxoTransaction(&req1)
	assert.Nil(t, err)
	t.Logf("reply1:%v", reply1)

}

func TestQueryTransactionWithVoutNoAddress(t *testing.T) {
	btcChainAdaptor := newChainAdaptorWithConfig(conf)

	var req proto.QueryTransactionRequest
	req.Chain = ChainName
	req.Symbol = Symbol
	req.TxHash = "b71ed2cfdb05dd307ea8beaa1fe82ceacf75d5a6ee8a39624d1696a15dc02465"

	reply, err := btcChainAdaptor.QueryUtxoTransaction(&req)
	assert.Nil(t, err)
	assert.Equal(t, uint64(1635381), reply.BlockHeight)

	assert.Equal(t, 1, len(reply.Vins))
	assert.Equal(t, "04517627b390055127ef3a93f79e9bd664f1c7a2f1de96da02dd98eff2a01a91", reply.Vins[0].Hash)
	assert.Equal(t, "tb1q5dxjv4j7hhdz48ct5e0f8dm0l05vm8pc345qhm", reply.Vins[0].Address)
	assert.Equal(t, int64(6062071), reply.Vins[0].Amount)
	assert.Equal(t, uint32(1), reply.Vins[0].Index)

	assert.Equal(t, 3, len(reply.Vouts))
	assert.Equal(t, "n2QJoRHB3KaUgE8oRvb7pGk7sVa67hb5Ai", reply.Vouts[0].Address)
	assert.Equal(t, int64(1000000), reply.Vouts[0].Amount)
	assert.Equal(t, "tb1q8kdexucnuw47runnekgpm9pe8l7wv0sjysr4ul", reply.Vouts[1].Address)
	assert.Equal(t, int64(5061892), reply.Vouts[1].Amount)
	assert.Equal(t, "", reply.Vouts[2].Address)
	assert.Equal(t, int64(0), reply.Vouts[2].Amount)
	assert.Equal(t, 0, len(reply.SignHashes))
}

func TestQueryTransaction(t *testing.T) {
	btcChainAdaptor := newChainAdaptorWithConfig(conf)

	var req proto.QueryTransactionRequest
	req.Chain = ChainName
	req.Symbol = Symbol
	req.TxHash = hash

	reply, err := btcChainAdaptor.QueryUtxoTransaction(&req)
	assert.Nil(t, err)
	assert.Equal(t, "2MyXNsXWUYmhVth3Rm6DWrDnpfiia79UsPk", reply.Vins[0].Address)
	assert.Equal(t, uint64(1519611), reply.BlockHeight)
	assert.Equal(t, "3803c9d5c80e35dcdd76ecf059ed736439688ae9f884f4ab5fb3aaa3d8156726", reply.Vins[0].Hash)
	assert.Equal(t, uint32(0), reply.Vins[0].Index)
	assert.Equal(t, "n16WjT35Tt33QHtySf31SB3M3bFFBSPU9w", reply.Vouts[0].Address)
	assert.Equal(t, "2Mww6tED1opzwN2D3rqqKW9z6BdLstdRFpL", reply.Vouts[1].Address)
	// assert.Equal(t, 0, len(reply.SignHashes))
	// assert.Equal(t, "61946e95671a258120ef31f6c19c6d80f9d4c2e040b985d1e02d9a5740dbfaf8", hex.EncodeToString(reply.SignHashes[0]))

	req.TxHash = "3803c9d5c80e35dcdd76ecf059ed736439688ae9f884f4ab5fb3aaa3d8156726"
	reply, err = btcChainAdaptor.QueryUtxoTransaction(&req)
	assert.Nil(t, err)
	assert.Equal(t, "2MxqkiuCE8a8vPdZpV4Zm3FJjPaadwsE1w1", reply.Vins[0].Address)
	assert.Equal(t, uint64(1519608), reply.BlockHeight)
	assert.Equal(t, "49dad7fa200ebdf536699275c7ab9f62337f8f46f1ede23ae38d0a4a2d1eb4b0", reply.Vins[0].Hash)
	assert.Equal(t, uint32(1), reply.Vins[0].Index)
	assert.Equal(t, "2MyXNsXWUYmhVth3Rm6DWrDnpfiia79UsPk", reply.Vouts[0].Address)
	assert.Equal(t, "2NBMEXb9j6L6geoMAxEdfoKZkvZxNo9sJxF", reply.Vouts[1].Address)
	// assert.Equal(t, 1, len(reply.SignHashes))
	// assert.Equal(t, "3caadddc80b56dafcdd695f9de244743fa6ea06393a88e9cf54fd797365b0e03", hex.EncodeToString(reply.SignHashes[0]))

	req.TxHash = "bc703215720998316f66833dcea3056842d5d0565ae38c0d078caf060cb7b64c"
	reply, err = btcChainAdaptor.QueryUtxoTransaction(&req)
	assert.Nil(t, err)
	assert.Equal(t, "mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9", reply.Vins[0].Address)
	assert.Equal(t, uint64(1322962), reply.BlockHeight)
	assert.Equal(t, "c94a4debf11bfef9316681f61fb663d15ed7a6ad8698063c4f1348575aec57a9", reply.Vins[0].Hash)
	assert.Equal(t, uint32(0), reply.Vins[0].Index)
	assert.Equal(t, "mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9", reply.Vouts[0].Address)
	assert.Equal(t, "2MthzQgsQ8Rw8vPMtTsrTdqc9HsWiDHM9VY", reply.Vouts[1].Address)
	// assert.Equal(t, 1, len(reply.SignHashes))
	// assert.Equal(t, "faa1e29f8ff3c19e6f307e9f2ce2f2f0ded93930796dbcf84ea59431136a0b6e", hex.EncodeToString(reply.SignHashes[0]))

	// 2 in 2 out
	req.TxHash = "22bdf8e436a69ddba55b3cd2eee6b94abe2f73d0f4282287842d2fdee46161e9"
	reply, err = btcChainAdaptor.QueryUtxoTransaction(&req)
	assert.Nil(t, err)
	assert.Equal(t, "mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9", reply.Vins[0].Address)
	assert.Equal(t, "mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9", reply.Vins[1].Address)
	assert.Equal(t, uint64(1574657), reply.BlockHeight)
	assert.Equal(t, "37a890e02a48f515a574eb6c2e7f22542fe362a2df2f64b1ceb3d841c00f1dcf", reply.Vins[0].Hash)
	assert.Equal(t, "19d4409350ab3fdf23be52e5526b4d83265a638b77327a579fc131341c0343f2", reply.Vins[1].Hash)
	assert.Equal(t, uint32(0), reply.Vins[0].Index)
	assert.Equal(t, uint32(1), reply.Vins[1].Index)
	assert.Equal(t, "mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9", reply.Vouts[0].Address)
	assert.Equal(t, "2MthzQgsQ8Rw8vPMtTsrTdqc9HsWiDHM9VY", reply.Vouts[1].Address)
	// assert.Equal(t, 2, len(reply.SignHashes))
	// assert.Equal(t, "08ebbd73dcefec70b66bdc185c4f7a54a0ced720ad455c259d427e94c86100ce", hex.EncodeToString(reply.SignHashes[0]))
	// assert.Equal(t, "d6c5e99916f44160cc2f24667013b02bd4ca050e04ab2c9fee5df01ccb7ee7a8", hex.EncodeToString(reply.SignHashes[1]))

	// 4 in 2 out
	req.TxHash = "ee00f56bf407a3d74d611f21d6a8988da34f891d68bd4ea2a3f1140e2d26a849"
	reply, err = btcChainAdaptor.QueryUtxoTransaction(&req)
	assert.Nil(t, err)
	assert.Equal(t, "mnU8YocHVk9dsxWNbFMzrYeRTBmxAWZWfh", reply.Vins[0].Address)
	assert.Equal(t, "tb1q937nex693ag5529nfm40ggscl7jnyzh8hcqrzp", reply.Vins[1].Address)
	assert.Equal(t, "mnU8YocHVk9dsxWNbFMzrYeRTBmxAWZWfh", reply.Vins[2].Address)
	assert.Equal(t, "tb1q3ykratwr8gymlncfj6m78smxtzr5vmtj3gng5z", reply.Vins[3].Address)
	assert.Equal(t, uint64(1574657), reply.BlockHeight)
	assert.Equal(t, "4d70867eabc0563e9d82d8a0fceedd362fd2c4bce72807ded416ea9e228d0820", reply.Vins[0].Hash)
	assert.Equal(t, "cc7ab1e1c9ee981505868a8faa6462b08fceef04bdc17d5b2b7642c56197e39b", reply.Vins[2].Hash)
	assert.Equal(t, uint32(0), reply.Vins[0].Index)
	assert.Equal(t, uint32(1), reply.Vins[1].Index)
	assert.Equal(t, "2NF9Ptff1Az97fQkf7HKaEHTPqkykiyLEoB", reply.Vouts[0].Address)
	assert.Equal(t, "tb1qk2rmpcl06ndl8uphtrple5jnznm3p026ea6m3w", reply.Vouts[1].Address)
	// assert.Equal(t, 4, len(reply.SignHashes))
	// assert.Equal(t, "3ef18572cdcb63b4590d39ae18b125c32fde6b70a98f5ef106028815a691b488", hex.EncodeToString(reply.SignHashes[0]))
	// assert.Equal(t, "89377373bef20a4e6c304be8428967c8dcfb5f53396c6e133645dc6f4e70f010", hex.EncodeToString(reply.SignHashes[1]))
}

func TestQueryTransactionCached(t *testing.T) {
	btcChainAdaptor := newChainAdaptorWithConfig(conf)

	var req proto.QueryTransactionRequest
	req.Chain = ChainName
	req.Symbol = Symbol
	req.TxHash = hash

	txCache := cache.GetTxCache()
	txCache.Purge()

	reply, err := btcChainAdaptor.QueryUtxoTransaction(&req)
	assert.Nil(t, err)
	assert.Equal(t, "2MyXNsXWUYmhVth3Rm6DWrDnpfiia79UsPk", reply.Vins[0].Address)
	assert.Equal(t, uint64(1519611), reply.BlockHeight)
	assert.Equal(t, "3803c9d5c80e35dcdd76ecf059ed736439688ae9f884f4ab5fb3aaa3d8156726", reply.Vins[0].Hash)
	assert.Equal(t, uint32(0), reply.Vins[0].Index)
	assert.Equal(t, "n16WjT35Tt33QHtySf31SB3M3bFFBSPU9w", reply.Vouts[0].Address)
	assert.Equal(t, "2Mww6tED1opzwN2D3rqqKW9z6BdLstdRFpL", reply.Vouts[1].Address)

	assert.Equal(t, 1, txCache.Len())
	key := strings.Join([]string{req.Symbol, req.TxHash}, ":")
	assert.Equal(t, true, txCache.Contains(key))

	req.TxHash = "3803c9d5c80e35dcdd76ecf059ed736439688ae9f884f4ab5fb3aaa3d8156726"
	reply, err = btcChainAdaptor.QueryUtxoTransaction(&req)
	assert.Nil(t, err)
	assert.Equal(t, "2MxqkiuCE8a8vPdZpV4Zm3FJjPaadwsE1w1", reply.Vins[0].Address)
	assert.Equal(t, uint64(1519608), reply.BlockHeight)
	assert.Equal(t, "49dad7fa200ebdf536699275c7ab9f62337f8f46f1ede23ae38d0a4a2d1eb4b0", reply.Vins[0].Hash)
	assert.Equal(t, uint32(1), reply.Vins[0].Index)

	assert.Equal(t, 2, txCache.Len())
	key = strings.Join([]string{req.Symbol, req.TxHash}, ":")
	assert.Equal(t, true, txCache.Contains(key))

	req.TxHash = "bc703215720998316f66833dcea3056842d5d0565ae38c0d078caf060cb7b64c"
	reply, err = btcChainAdaptor.QueryUtxoTransaction(&req)
	assert.Nil(t, err)
	assert.Equal(t, "mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9", reply.Vins[0].Address)
	assert.Equal(t, uint64(1322962), reply.BlockHeight)
	assert.Equal(t, "c94a4debf11bfef9316681f61fb663d15ed7a6ad8698063c4f1348575aec57a9", reply.Vins[0].Hash)
	assert.Equal(t, uint32(0), reply.Vins[0].Index)

	assert.Equal(t, 3, txCache.Len())
	key = strings.Join([]string{req.Symbol, req.TxHash}, ":")
	assert.Equal(t, true, txCache.Contains(key))

	// rertieve txhash from cache
	req.TxHash = hash
	reply, err = btcChainAdaptor.QueryUtxoTransaction(&req)
	assert.Nil(t, err)
	assert.Equal(t, "2MyXNsXWUYmhVth3Rm6DWrDnpfiia79UsPk", reply.Vins[0].Address)
	assert.Equal(t, uint64(1519611), reply.BlockHeight)
	assert.Equal(t, "3803c9d5c80e35dcdd76ecf059ed736439688ae9f884f4ab5fb3aaa3d8156726", reply.Vins[0].Hash)
	assert.Equal(t, uint32(0), reply.Vins[0].Index)
	assert.Equal(t, "n16WjT35Tt33QHtySf31SB3M3bFFBSPU9w", reply.Vouts[0].Address)
	assert.Equal(t, "2Mww6tED1opzwN2D3rqqKW9z6BdLstdRFpL", reply.Vouts[1].Address)

	assert.Equal(t, 3, txCache.Len())
	key = strings.Join([]string{req.Symbol, req.TxHash}, ":")
	assert.Equal(t, true, txCache.Contains(key))

	req.TxHash = "3803c9d5c80e35dcdd76ecf059ed736439688ae9f884f4ab5fb3aaa3d8156726"
	reply, err = btcChainAdaptor.QueryUtxoTransaction(&req)
	assert.Nil(t, err)
	assert.Equal(t, "2MxqkiuCE8a8vPdZpV4Zm3FJjPaadwsE1w1", reply.Vins[0].Address)
	assert.Equal(t, uint64(1519608), reply.BlockHeight)
	assert.Equal(t, "49dad7fa200ebdf536699275c7ab9f62337f8f46f1ede23ae38d0a4a2d1eb4b0", reply.Vins[0].Hash)
	assert.Equal(t, uint32(1), reply.Vins[0].Index)

	assert.Equal(t, 3, txCache.Len())
	key = strings.Join([]string{req.Symbol, req.TxHash}, ":")
	assert.Equal(t, true, txCache.Contains(key))

}

func TestQueryTransactionNotFound(t *testing.T) {
	btcChainAdaptor := newChainAdaptorWithConfig(conf)

	var req proto.QueryTransactionRequest
	req.Chain = ChainName
	req.Symbol = Symbol
	req.TxHash = "9f96e84aabb2e31432334220bd314738a1a437fdf29c8091dd9386537d350184"

	reply, err := btcChainAdaptor.QueryUtxoTransaction(&req)
	assert.Nil(t, err, "error: %s", err)
	assert.Equal(t, proto.TxStatus_NotFound, reply.TxStatus)
}

func TestQueryTransactionFromSignedData(t *testing.T) {
	bz, err := hex.DecodeString("02000000000101266715d8a3aab35fabf484f8e98a68396473ed59f0ec76dddc350ec8d5c9033800000000171600147e6e0170c81cf74bb9a433a2f905546a2766c98bfeffffff0210270000000000001976a914d6c331c38a8b4c966397c4862f86bcfe42cd924588ac6ccdae000000000017a914336b360ddbaf7dd99716bf1c2b92ad233ca9e2aa870247304402200d6fafc20ec2d1a52b62bf6130bbb22678e14c11dec18eea997483cd1cf9340a02201b88b3508454ae7ccf1ccef245f6eafaee6209b5f55bb2c27b17de44d20db67a012103fe6fb3175dd133e95bb8312ef0ad87017cb12f5b62b59fc2dd4b2e39690b2bf3fa2f1700")
	assert.Nil(t, err)

	req := proto.QueryTransactionFromSignedDataRequest{
		Chain:        ChainName,
		Symbol:       Symbol,
		SignedTxData: bz,
	}
	vins := []*proto.Vin{
		{
			Address: "2MyXNsXWUYmhVth3Rm6DWrDnpfiia79UsPk",
			Amount:  11466020,
		},
	}

	testVinOnOff(t, (*ChainAdaptor).QueryUtxoTransactionFromSignedData, &req, vins, func(_reply interface{}, err error) {
		reply := _reply.(*proto.QueryUtxoTransactionReply)

		assert.Equal(t, "c2247fb66cf44652f27552b052a7d359d48a1c8e90a50651f6104a441041963f", reply.TxHash)
		assert.Equal(t, "2MyXNsXWUYmhVth3Rm6DWrDnpfiia79UsPk", reply.Vins[0].Address)
		assert.Equal(t, "3803c9d5c80e35dcdd76ecf059ed736439688ae9f884f4ab5fb3aaa3d8156726", reply.Vins[0].Hash)
		assert.Equal(t, 1, len(reply.SignHashes))
		assert.Equal(t, "61946e95671a258120ef31f6c19c6d80f9d4c2e040b985d1e02d9a5740dbfaf8", hex.EncodeToString(reply.SignHashes[0]))
	})

	bz1, err := hex.DecodeString("0100000001a957ec5a5748134f3c069886ada6d75ed163b61ff6816631f9fe1bf1eb4d4ac9000000008a4730440220486972701a1f11d72c575e0fec145c957c21a89df58a2c5878a4f62253eedaa1022065e13ca5d689c8b1c86bbcc5d30f05340948cc12c5e17c55cc434fca6f495ba10141043cd360fecac46da64c411c6b471d8e147504ed74c2cafd9a29329c63c4eaf1603fb5a230c1ba28d93bb6834989869259d4a4156d33fd5f99075e4b968cdbe8b8ffffffff020048e801000000001976a91419064bda7eb5049f922a4bca4c24808c6aea948d88ac005307000000000017a91410080578e54a2a66efcb55e69b073100d0da47b98700000000")
	assert.Nil(t, err)
	req = proto.QueryTransactionFromSignedDataRequest{
		Chain:        ChainName,
		Symbol:       Symbol,
		SignedTxData: bz1,
	}
	vins = []*proto.Vin{
		{
			Address: "mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9",
			Amount:  32500000,
		},
	}

	testVinOnOff(t, (*ChainAdaptor).QueryUtxoTransactionFromSignedData, &req, vins, func(_reply interface{}, err error) {
		reply := _reply.(*proto.QueryUtxoTransactionReply)

		assert.Nil(t, err, "unexpected error: ", err)
		assert.Equal(t, "bc703215720998316f66833dcea3056842d5d0565ae38c0d078caf060cb7b64c", reply.TxHash)
		assert.Equal(t, "mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9", reply.Vins[0].Address)
		assert.Equal(t, "c94a4debf11bfef9316681f61fb663d15ed7a6ad8698063c4f1348575aec57a9", reply.Vins[0].Hash)
		assert.Equal(t, 1, len(reply.SignHashes))
		assert.Equal(t, "faa1e29f8ff3c19e6f307e9f2ce2f2f0ded93930796dbcf84ea59431136a0b6e", hex.EncodeToString(reply.SignHashes[0]))
	})
	// err
	bz2, err := hex.DecodeString("02000000000101266715d8a3abbbbbbbbbbab35fabf484f8e98a68396473ed59f0ec76dddc350ec8d5c9033800000000171600147e6e0170c81cf74bb9a433a2f905546a2766c98bfeffffff0210270000000000001976a914d6c331c38a8b4c966397c4862f86bcfe42cd924588ac6ccdae000000000017a914336b360ddbaf7dd99716bf1c2b92ad233ca9e2aa870247304402200d6fafc20ec2d1a52b62bf6130bbb22678e14c11dec18eea997483cd1cf9340a02201b88b3508454ae7ccf1ccef245f6eafaee6209b5f55bb2c27b17de44d20db67a012103fe6fb3175dd133e95bb8312ef0ad87017cb12f5b62b59fc2dd4b2e39690b2bf3fa2f17001d3476")
	assert.Nil(t, err)
	req.SignedTxData = bz2

	testVinOnOff(t, (*ChainAdaptor).QueryUtxoTransactionFromSignedData, &req, vins, func(_reply interface{}, err error) {
		reply := _reply.(*proto.QueryUtxoTransactionReply)

		assert.NotNil(t, err)
		assert.Equal(t, proto.ReturnCode_ERROR, reply.Code)
	})
}

func TestQueryTransactionFromSignedData2(t *testing.T) {
	bz1, err := hex.DecodeString("0100000001651e85675d438360c4cd1dbb48d4ed9506409cf9878d93f0db4b330dfedfcad5000000006a47304402204dda7e0745aaea39f8e80ba396d14c2c6a0b806372d993dce04e8b0b0e924dff022059c77f751c9bd6e726688c01cce56b77d83ca6dc55eddca3814631b3d1888b79012103d2ad1893f1178cd8dff57cfc5e842ada8b81465747548bf495c5951a2ca2da6effffffff0108520000000000001976a914f5e14e43e474738731b7f1ec64b8d090e0e1f9fe88ac00000000")
	assert.Nil(t, err)

	var req proto.QueryTransactionFromSignedDataRequest
	req.Chain = ChainName
	req.Symbol = Symbol
	req.SignedTxData = bz1
	vins := []*proto.Vin{
		{
			Address: "myfTHMo2f3yn1ZJCfgiv2xa18UQa6KZvj2",
			Amount:  22000,
		},
	}

	t.Logf("signTx:%v", hex.EncodeToString(bz1))
	testVinOnOff(t, (*ChainAdaptor).QueryUtxoTransactionFromSignedData, &req, vins, func(_reply interface{}, err error) {
		reply := _reply.(*proto.QueryUtxoTransactionReply)

		assert.Nil(t, err, "unexpected error", err)
		assert.Equal(t, "0e845d4c474278ca5c9c61c487f60d2ccd53c66b8f11999b7619ea2e7f1b7523", reply.TxHash)
		assert.Equal(t, "d5cadffe0d334bdbf0938d87f99c400695edd448bb1dcdc46083435d67851e65", reply.Vins[0].Hash)
		assert.Equal(t, "myfTHMo2f3yn1ZJCfgiv2xa18UQa6KZvj2", reply.Vins[0].Address)
		assert.Equal(t, uint32(0), reply.Vins[0].Index)
		assert.Equal(t, int64(22000), reply.Vins[0].Amount)

		assert.Equal(t, "n3w3kZvxmHKrjvg5wP1NEgwkVCgsUsCgm4", reply.Vouts[0].Address)
		assert.Equal(t, int64(21000), reply.Vouts[0].Amount)
		assert.Equal(t, 1, len(reply.SignHashes))
	})

}

func TestQueryTransactionFromSignedData3(t *testing.T) {

	expectedHashStr := "2b15f47fb958b3f42342f3b502234ce830e0f88e3d894412126148b246ff79d6"
	bz, err := hex.DecodeString("01000000027e3c8ba8fb21b2404cc29c5e66f1c24cdebc38cad2bdfb6c8f670c010c415a94000000006a47304402207e4a96ad4e622ea8d31a4235da7dc0d9d925feb9588d363e193de7401b9bbdeb02204d491658507dc2d9cdeade7e207ff6e6f70791ad19c42e4517fe7a9af666a43201210325dbba4831ec1d0f180f5709e6a4f2d259b26f9f2581974fa4b687ea4b5bdd28ffffffffb3dc031ed1b4837caf6db13ae68acc6a2710271d900fe49ecbbcb2079006236a000000006b483045022100b3bcf5beb5b61d85bc2dd3deaf372e5eaac4ecb18e28073d5535274ffaecf00f02200f02f10a654e019bcecaa7ae9eeacfea26ff32c08c08e0d3775289f341a8ec03012103d2ad1893f1178cd8dff57cfc5e842ada8b81465747548bf495c5951a2ca2da6effffffff0118790000000000001976a914f5e14e43e474738731b7f1ec64b8d090e0e1f9fe88ac00000000")
	assert.Nil(t, err)

	var req proto.QueryTransactionFromSignedDataRequest
	req.Chain = ChainName
	req.Symbol = Symbol
	req.SignedTxData = bz
	vins := []*proto.Vin{
		{
			Address: "mgbXsi4Qap7V88iy1RXL1ehQKxhhcPZrfq",
			Amount:  10000,
		},
		{
			Address: "myfTHMo2f3yn1ZJCfgiv2xa18UQa6KZvj2",
			Amount:  22000,
		},
	}
	t.Logf("signTx:%v", hex.EncodeToString(bz))

	testVinOnOff(t, (*ChainAdaptor).QueryUtxoTransactionFromSignedData, &req, vins, func(_reply interface{}, err error) {
		reply := _reply.(*proto.QueryUtxoTransactionReply)

		assert.Nil(t, err)
		//	t.Logf("reply:%+v", reply)
		assert.Equal(t, expectedHashStr, reply.TxHash)
		assert.Equal(t, "945a410c010c678f6cfbbdd2ca38bcde4cc2f1665e9cc24c40b221fba88b3c7e", reply.Vins[0].Hash)
		assert.Equal(t, uint32(0), reply.Vins[0].Index)
		assert.Equal(t, "mgbXsi4Qap7V88iy1RXL1ehQKxhhcPZrfq", reply.Vins[0].Address)
		assert.Equal(t, int64(10000), reply.Vins[0].Amount)

		assert.Equal(t, "6a23069007b2bccb9ee40f901d2710276acc8ae63ab16daf7c83b4d11e03dcb3", reply.Vins[1].Hash)
		assert.Equal(t, uint32(0), reply.Vins[1].Index)
		assert.Equal(t, "myfTHMo2f3yn1ZJCfgiv2xa18UQa6KZvj2", reply.Vins[1].Address)
		assert.Equal(t, int64(22000), reply.Vins[1].Amount)

		assert.Equal(t, "n3w3kZvxmHKrjvg5wP1NEgwkVCgsUsCgm4", reply.Vouts[0].Address)
		assert.Equal(t, int64(31000), reply.Vouts[0].Amount)
		assert.Equal(t, 2, len(reply.SignHashes))
	})

}

func TestCreateTransaction(t *testing.T) {
	btcChainAdaptor := newChainAdaptorWithConfig(conf)

	expectedTxData := "0100000001a957ec5a5748134f3c069886ada6d75ed163b61ff6816631f9fe1bf1eb4d4ac90000000000ffffffff020048e801000000001976a91419064bda7eb5049f922a4bca4c24808c6aea948d88ac005307000000000017a91410080578e54a2a66efcb55e69b073100d0da47b98700000000"
	expectedHash := "faa1e29f8ff3c19e6f307e9f2ce2f2f0ded93930796dbcf84ea59431136a0b6e"
	vin := []*proto.Vin{
		{Hash: "c94a4debf11bfef9316681f61fb663d15ed7a6ad8698063c4f1348575aec57a9", Index: uint32(0), Amount: int64(32500000), Address: "mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9"},
	}

	vout := []*proto.Vout{
		{Address: "mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9", Amount: int64(32000000)},
		{Address: "2MthzQgsQ8Rw8vPMtTsrTdqc9HsWiDHM9VY", Amount: int64(480000)},
	}

	req1 := proto.CreateUtxoTransactionRequest{
		Chain:  ChainName,
		Symbol: Symbol,
		Vins:   vin,
		Vouts:  vout,
		Fee:    big.NewInt(0).SetInt64(20000).String(),
	}

	reply1, err := btcChainAdaptor.CreateUtxoTransaction(&req1)
	assert.Nil(t, err)
	assert.Equal(t, expectedTxData, hex.EncodeToString(reply1.TxData))
	assert.Equal(t, 1, len(reply1.SignHashes))
	assert.Equal(t, expectedHash, hex.EncodeToString(reply1.SignHashes[0]))

	req2 := &proto.QueryTransactionFromDataRequest{
		Chain:   ChainName,
		Symbol:  Symbol,
		RawData: reply1.TxData,
	}

	testVinOnOff(t, (*ChainAdaptor).QueryUtxoTransactionFromData, req2, vin, func(replyA interface{}, err error) {
		reply := replyA.(*proto.QueryUtxoTransactionReply)

		assert.Nil(t, err)
		assert.Equal(t, reply1.SignHashes, reply.SignHashes)
		assert.Equal(t, 1, len(reply1.SignHashes))
		assert.Equal(t, expectedHash, hex.EncodeToString(reply.SignHashes[0]))
	})
}

func TestQueryTransactionFromData(t *testing.T) {
	data, err := hex.DecodeString("0100000001a957ec5a5748134f3c069886ada6d75ed163b61ff6816631f9fe1bf1eb4d4ac90000000000ffffffff020048e801000000001976a91419064bda7eb5049f922a4bca4c24808c6aea948d88ac005307000000000017a91410080578e54a2a66efcb55e69b073100d0da47b98700000000")
	expectedHash := "faa1e29f8ff3c19e6f307e9f2ce2f2f0ded93930796dbcf84ea59431136a0b6e"
	assert.Nil(t, err)

	req := &proto.QueryTransactionFromDataRequest{
		Chain:   ChainName,
		Symbol:  Symbol,
		RawData: data,
	}
	vins := []*proto.Vin{
		{Hash: "c94a4debf11bfef9316681f61fb663d15ed7a6ad8698063c4f1348575aec57a9", Index: uint32(0), Amount: int64(32500000), Address: "mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9"},
	}

	testVinOnOff(t, (*ChainAdaptor).QueryUtxoTransactionFromData, req, vins, func(replyA interface{}, err error) {
		reply := replyA.(*proto.QueryUtxoTransactionReply)

		assert.Nil(t, err)
		assert.Equal(t, expectedHash, hex.EncodeToString(reply.SignHashes[0]))
		assert.Equal(t, "c94a4debf11bfef9316681f61fb663d15ed7a6ad8698063c4f1348575aec57a9", reply.Vins[0].Hash)
		assert.Equal(t, uint32(0), reply.Vins[0].Index)
		assert.Equal(t, int64(32500000), reply.Vins[0].Amount)
		assert.Equal(t, "mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9", reply.Vins[0].Address)
		assert.Equal(t, 1, len(reply.SignHashes))
		assert.Equal(t, "20000", reply.CostFee)
	})
}

func TestQueryTransactionFromData2(t *testing.T) {
	data, err := hex.DecodeString("010000000282f957d1598291a946d34a114e5391b166cfbf8228325a12fe703cf8db18343f0000000000ffffffff82f957d1598291a946d34a114e5391b166cfbf8228325a12fe703cf8db18343f0100000000ffffffff02b4270100000000001976a91419064bda7eb5049f922a4bca4c24808c6aea948d88acf40100000000000017a91410080578e54a2a66efcb55e69b073100d0da47b98700000000")
	expectedHash1 := "890c6bb02e75bebf6ae1f92cb3138bb11ec7dd3035938d1224384ad861c52041"
	assert.Nil(t, err)

	/*
		vout := []*proto.Vout{
			&proto.Vout{Address: "mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9", Amount: int64(75700)},
			&proto.Vout{Address: "2MthzQgsQ8Rw8vPMtTsrTdqc9HsWiDHM9VY", Amount: int64(500)},
		}
	*/

	req := &proto.QueryTransactionFromDataRequest{
		Chain:   ChainName,
		Symbol:  Symbol,
		RawData: data,
	}
	vins := []*proto.Vin{
		{Hash: "3f3418dbf83c70fe125a322882bfcf66b191534e114ad346a9918259d157f982", Index: uint32(0), Amount: int64(33700), Address: "n3HsmPMAEa2ovEzMNrXKZhSMGvUEfBpHcd"},
		{Hash: "3f3418dbf83c70fe125a322882bfcf66b191534e114ad346a9918259d157f982", Index: uint32(1), Amount: int64(43000), Address: "mwF4uCj81VAqKBDjmV2yLESREGDCyZL6Ap"},
	}

	testVinOnOff(t, (*ChainAdaptor).QueryUtxoTransactionFromData, req, vins, func(replyA interface{}, err error) {
		reply := replyA.(*proto.QueryUtxoTransactionReply)

		assert.Nil(t, err)
		assert.Equal(t, 2, len(reply.Vins))
		assert.Equal(t, "3f3418dbf83c70fe125a322882bfcf66b191534e114ad346a9918259d157f982", reply.Vins[0].Hash)
		assert.Equal(t, uint32(0), reply.Vins[0].Index)
		assert.Equal(t, "n3HsmPMAEa2ovEzMNrXKZhSMGvUEfBpHcd", reply.Vins[0].Address)
		assert.Equal(t, int64(33700), reply.Vins[0].Amount)

		assert.Equal(t, "3f3418dbf83c70fe125a322882bfcf66b191534e114ad346a9918259d157f982", reply.Vins[1].Hash)
		assert.Equal(t, uint32(1), reply.Vins[1].Index)
		assert.Equal(t, "mwF4uCj81VAqKBDjmV2yLESREGDCyZL6Ap", reply.Vins[1].Address)
		assert.Equal(t, int64(43000), reply.Vins[1].Amount)

		assert.Equal(t, "mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9", reply.Vouts[0].Address)
		assert.Equal(t, int64(75700), reply.Vouts[0].Amount)
		assert.Equal(t, "2MthzQgsQ8Rw8vPMtTsrTdqc9HsWiDHM9VY", reply.Vouts[1].Address)
		assert.Equal(t, int64(500), reply.Vouts[1].Amount)

		assert.Equal(t, 2, len(reply.SignHashes))
		assert.Equal(t, expectedHash1, hex.EncodeToString(reply.SignHashes[0]))
		assert.Equal(t, "8d295b75352045f64513919470ba46c42ef3945b7513ed3dac15ae7042e4319c", hex.EncodeToString(reply.SignHashes[1]))
		assert.Equal(t, "500", reply.CostFee)
	})
}

func testVinOnOff(t *testing.T, method interface{}, req interface{}, vins []*proto.Vin, callback func(interface{}, error)) {
	testF := func(name string, adaptor *ChainAdaptor, reqValue reflect.Value) {
		t.Run(name, func(t *testing.T) {
			results := reflect.ValueOf(method).Call([]reflect.Value{
				reflect.ValueOf(adaptor),
				reqValue,
			})
			assert.Equal(t, 2, len(results))
			var err error
			if results[1].Interface() == nil {
				err = nil
			} else {
				var ok bool
				err, ok = results[1].Interface().(error)
				assert.True(t, ok)
			}
			callback(results[0].Interface(), err)
		})
	}
	reqValue := reflect.ValueOf(req)

	testF("without vin", newChainAdaptorWithConfig(conf), reqValue)

	vinValue := reflect.ValueOf(vins)
	reqValue.Elem().FieldByName("Vins").Set(vinValue)
	testF("with vin", NewLocalChainAdaptor(config.TestNet).(*ChainAdaptor), reqValue)

	reqValue.Elem().FieldByName("Vins").Set(reflect.Zero(vinValue.Type()))
}

func TestQueryTransactionFromData3(t *testing.T) {
	data := []byte{1, 0, 0, 0, 1, 101, 30, 133, 103, 93, 67, 131, 96, 196, 205, 29, 187, 72, 212, 237, 149, 6, 64, 156, 249, 135, 141, 147, 240, 219, 75, 51, 13, 254, 223, 202, 213, 0, 0, 0, 0, 0, 255, 255, 255, 255, 1, 8, 82, 0, 0, 0, 0, 0, 0, 25, 118, 169, 20, 245, 225, 78, 67, 228, 116, 115, 135, 49, 183, 241, 236, 100, 184, 208, 144, 224, 225, 249, 254, 136, 172, 0, 0, 0, 0}

	req := &proto.QueryTransactionFromDataRequest{
		Chain:   ChainName,
		Symbol:  Symbol,
		RawData: data,
	}
	vins := []*proto.Vin{
		{
			Hash:    "d5cadffe0d334bdbf0938d87f99c400695edd448bb1dcdc46083435d67851e65",
			Index:   0,
			Amount:  22000,
			Address: "myfTHMo2f3yn1ZJCfgiv2xa18UQa6KZvj2",
		},
	}
	testVinOnOff(t, (*ChainAdaptor).QueryUtxoTransactionFromData, req, vins, func(_reply interface{}, err error) {
		reply := _reply.(*proto.QueryUtxoTransactionReply)
		assert.Nil(t, err)
		assert.Equal(t, uint32(0), reply.Vins[0].Index)
		assert.Equal(t, int64(22000), reply.Vins[0].Amount)
		assert.Equal(t, "myfTHMo2f3yn1ZJCfgiv2xa18UQa6KZvj2", reply.Vins[0].Address)
		assert.Equal(t, "n3w3kZvxmHKrjvg5wP1NEgwkVCgsUsCgm4", reply.Vouts[0].Address)
		assert.Equal(t, int64(21000), reply.Vouts[0].Amount)
		assert.Equal(t, 1, len(reply.SignHashes))
		assert.Equal(t, "1000", reply.CostFee)
	})
}

func TestCreateTransactionAmountMismatch(t *testing.T) {
	btcChainAdaptor := newChainAdaptorWithConfig(conf)

	vin := []*proto.Vin{
		{Hash: "c94a4debf11bfef9316681f61fb663d15ed7a6ad8698063c4f1348575aec57a9", Index: uint32(0), Amount: int64(32500000), Address: "mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9"},
	}

	vout := []*proto.Vout{
		{Address: "mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9", Amount: int64(32000000)},
		{Address: "2MthzQgsQ8Rw8vPMtTsrTdqc9HsWiDHM9VY", Amount: int64(480000)},
	}

	req := proto.CreateUtxoTransactionRequest{
		Chain:  ChainName,
		Symbol: Symbol,
		Vins:   vin,
		Vouts:  vout,
		Fee:    big.NewInt(0).SetInt64(40000).String(),
	}

	reply, err := btcChainAdaptor.CreateUtxoTransaction(&req)
	assert.NotNil(t, err)
	assert.Equal(t, "CreateTransaction, total amount in != total amount out + fee", reply.Msg)
}

func TestCreateAndSignTransactionOneTxinUnCompressed(t *testing.T) {
	btcChainAdaptor := newChainAdaptorWithConfig(conf)

	client := btcChainAdaptor.client
	client.compressed = false

	expectedHash, _ := hex.DecodeString("4bc328ba8fcf517d612b0ba01e0cd005e5829a16b295e33150dc2e2afbbb56e6")
	expectedTx, _ := hex.DecodeString("0100000001cf1d0fc041d8b3ceb1642fdfa262e32f54227f2e6ceb74a515f5482ae090a8370000000000ffffffff02007d0000000000001976a91419064bda7eb5049f922a4bca4c24808c6aea948d88acf40100000000000017a91410080578e54a2a66efcb55e69b073100d0da47b98700000000")
	expectedPkData, _ := hex.DecodeString("043cd360fecac46da64c411c6b471d8e147504ed74c2cafd9a29329c63c4eaf1603fb5a230c1ba28d93bb6834989869259d4a4156d33fd5f99075e4b968cdbe8b8")
	expectedSig, _ := hex.DecodeString("3044022017e44e2cb5861720a8c705fdabbde74355245592e79f7b2f7802c22b26851d7b02204d47f86b671ffdf3336ecec673225c92f86928c899cf44b7e0fe6c30fc681b6b01")
	expectedSignedTx, _ := hex.DecodeString("0100000001cf1d0fc041d8b3ceb1642fdfa262e32f54227f2e6ceb74a515f5482ae090a837000000008a473044022017e44e2cb5861720a8c705fdabbde74355245592e79f7b2f7802c22b26851d7b02204d47f86b671ffdf3336ecec673225c92f86928c899cf44b7e0fe6c30fc681b6b0141043cd360fecac46da64c411c6b471d8e147504ed74c2cafd9a29329c63c4eaf1603fb5a230c1ba28d93bb6834989869259d4a4156d33fd5f99075e4b968cdbe8b8ffffffff02007d0000000000001976a91419064bda7eb5049f922a4bca4c24808c6aea948d88acf40100000000000017a91410080578e54a2a66efcb55e69b073100d0da47b98700000000")
	vin := []*proto.Vin{
		{Hash: "37a890e02a48f515a574eb6c2e7f22542fe362a2df2f64b1ceb3d841c00f1dcf", Index: uint32(0), Amount: int64(33000), Address: "mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9"},
	}

	vout := []*proto.Vout{
		{Address: "mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9", Amount: int64(32000)},
		{Address: "2MthzQgsQ8Rw8vPMtTsrTdqc9HsWiDHM9VY", Amount: int64(500)},
	}

	req1 := proto.CreateUtxoTransactionRequest{
		Chain:  ChainName,
		Symbol: Symbol,
		Vins:   vin,
		Vouts:  vout,
		Fee:    big.NewInt(0).SetInt64(500).String(),
	}

	reply1, err := btcChainAdaptor.CreateUtxoTransaction(&req1)
	assert.Nil(t, err)
	assert.Equal(t, expectedTx, reply1.TxData)
	assert.Equal(t, expectedHash, reply1.SignHashes[0])
	assert.Equal(t, 1, len(reply1.SignHashes))

	// Sign with Prviate key
	priWif, err := btcutil.DecodeWIF("cMqUKKvaEzKfPnooWk5fRe5c5CXGw2R1KEKefdkktTDSzSGmqfxN")
	assert.Nil(t, err)
	privKey := priWif.PrivKey
	pkData, sig0 := signOneVin(privKey, reply1.SignHashes[0], client.compressed)
	assert.Equal(t, expectedPkData, pkData)
	assert.Equal(t, expectedSig, sig0)

	originSig0, err := btcec.ParseSignature(sig0, btcec.S256())
	assert.Nil(t, err)
	r, s := originSig0.R.Bytes(), originSig0.S.Bytes()
	sig := make([]byte, 65)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)
	sig[64] = 0

	// creat sigscript and verify
	r1 := new(big.Int).SetBytes(sig[0:32])
	s1 := new(big.Int).SetBytes(sig[32:64])
	assert.True(t, originSig0.R.Cmp(r1) == 0)
	assert.True(t, originSig0.S.Cmp(s1) == 0)

	req2 := proto.CreateUtxoSignedTransactionRequest{
		Chain:      ChainName,
		Symbol:     Symbol,
		TxData:     reply1.TxData,
		PublicKeys: [][]byte{pkData},
		Signatures: [][]byte{sig},
	}
	reply2, err := btcChainAdaptor.CreateUtxoSignedTransaction(&req2)
	assert.Nil(t, err)
	assert.Equal(t, expectedSignedTx, reply2.SignedTxData)

}

func btcecSigToBHSigBytes(sigBytes []byte) ([]byte, error) {
	sig, err := btcec.ParseSignature(sigBytes, btcec.S256())
	if err != nil {
		return nil, err
	}
	r, s := sig.R.Bytes(), sig.S.Bytes()
	bhSigBytes := make([]byte, 64)
	copy(bhSigBytes[32-len(r):32], r)
	copy(bhSigBytes[64-len(s):64], s)
	return bhSigBytes, nil
}

func TestCreateAndSignTransactionMultiTxinUnCompressed(t *testing.T) {
	btcChainAdaptor := newChainAdaptorWithConfig(conf)

	client := btcChainAdaptor.client
	client.compressed = false

	expectedHash0, _ := hex.DecodeString("08ebbd73dcefec70b66bdc185c4f7a54a0ced720ad455c259d427e94c86100ce")
	expectedHash1, _ := hex.DecodeString("d6c5e99916f44160cc2f24667013b02bd4ca050e04ab2c9fee5df01ccb7ee7a8")
	expectedTx, _ := hex.DecodeString("0100000002cf1d0fc041d8b3ceb1642fdfa262e32f54227f2e6ceb74a515f5482ae090a8370000000000fffffffff243031c3431c19f577a32778b635a26834d6b52e552be23df3fab509340d4190100000000ffffffff02902d0100000000001976a91419064bda7eb5049f922a4bca4c24808c6aea948d88ac2c0100000000000017a91410080578e54a2a66efcb55e69b073100d0da47b98700000000")
	expectedPkData, _ := hex.DecodeString("043cd360fecac46da64c411c6b471d8e147504ed74c2cafd9a29329c63c4eaf1603fb5a230c1ba28d93bb6834989869259d4a4156d33fd5f99075e4b968cdbe8b8")
	expectedSig0, _ := hex.DecodeString("304402201889ee7147582c897f9d2dcc81bfdb321eddd33bf47820f72c7fa3237425a4cc022057d359d3c495824e48d7f6cc77a9998667f0ea388fb13e2dd24df32bca977a7e01")
	expectedSig1, _ := hex.DecodeString("304402200c0221775e00b66a437cae72393c0364197bbf5925a434f4e504647bad2b739802201cde65725a699552029ef00629495542a8a0b9781ce962cb6afc0d313cd7f04001")

	expectedSignedTx, _ := hex.DecodeString("0100000002cf1d0fc041d8b3ceb1642fdfa262e32f54227f2e6ceb74a515f5482ae090a837000000008a47304402201889ee7147582c897f9d2dcc81bfdb321eddd33bf47820f72c7fa3237425a4cc022057d359d3c495824e48d7f6cc77a9998667f0ea388fb13e2dd24df32bca977a7e0141043cd360fecac46da64c411c6b471d8e147504ed74c2cafd9a29329c63c4eaf1603fb5a230c1ba28d93bb6834989869259d4a4156d33fd5f99075e4b968cdbe8b8fffffffff243031c3431c19f577a32778b635a26834d6b52e552be23df3fab509340d419010000008a47304402200c0221775e00b66a437cae72393c0364197bbf5925a434f4e504647bad2b739802201cde65725a699552029ef00629495542a8a0b9781ce962cb6afc0d313cd7f0400141043cd360fecac46da64c411c6b471d8e147504ed74c2cafd9a29329c63c4eaf1603fb5a230c1ba28d93bb6834989869259d4a4156d33fd5f99075e4b968cdbe8b8ffffffff02902d0100000000001976a91419064bda7eb5049f922a4bca4c24808c6aea948d88ac2c0100000000000017a91410080578e54a2a66efcb55e69b073100d0da47b98700000000")
	vin := []*proto.Vin{
		{Hash: "37a890e02a48f515a574eb6c2e7f22542fe362a2df2f64b1ceb3d841c00f1dcf", Index: uint32(0), Amount: int64(33000), Address: "mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9"},
		{Hash: "19d4409350ab3fdf23be52e5526b4d83265a638b77327a579fc131341c0343f2", Index: uint32(1), Amount: int64(45000), Address: "mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9"},
	}

	vout := []*proto.Vout{
		{Address: "mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9", Amount: int64(77200)},
		{Address: "2MthzQgsQ8Rw8vPMtTsrTdqc9HsWiDHM9VY", Amount: int64(300)},
	}

	req1 := proto.CreateUtxoTransactionRequest{
		Chain:  ChainName,
		Symbol: Symbol,
		Vins:   vin,
		Vouts:  vout,
		Fee:    big.NewInt(0).SetInt64(500).String(),
	}

	reply1, err := btcChainAdaptor.CreateUtxoTransaction(&req1)
	assert.Nil(t, err)
	// t.Logf("SignHash[0]:%x\n", reply1.SignHashes[0])
	// t.Logf("SignHash[1]:%x\n", reply1.SignHashes[1])
	// t.Logf("TxData:%x\n", reply1.TxData)
	assert.Equal(t, expectedTx, reply1.TxData)
	assert.Equal(t, expectedHash0, reply1.SignHashes[0])
	assert.Equal(t, expectedHash1, reply1.SignHashes[1])
	assert.Equal(t, 2, len(reply1.SignHashes))

	// Sign with Prviate key
	priWif, err := btcutil.DecodeWIF("cMqUKKvaEzKfPnooWk5fRe5c5CXGw2R1KEKefdkktTDSzSGmqfxN")
	assert.Nil(t, err)
	privKey := priWif.PrivKey
	pkData, sig0 := signOneVin(privKey, reply1.SignHashes[0], client.compressed)
	_, sig1 := signOneVin(privKey, reply1.SignHashes[1], client.compressed)
	assert.Equal(t, expectedPkData, pkData)
	assert.Equal(t, expectedSig0, sig0)
	assert.Equal(t, expectedSig1, sig1)
	// t.Logf("sig[0]:%x\n", sigs[0])
	// t.Logf("sig[1]:%x\n", sigs[1])
	bhSigs0, err := btcecSigToBHSigBytes(sig0)
	assert.Nil(t, err)
	bhSigs1, err := btcecSigToBHSigBytes(sig1)
	assert.Nil(t, err)
	req2 := proto.CreateUtxoSignedTransactionRequest{
		Chain:      ChainName,
		Symbol:     Symbol,
		TxData:     reply1.TxData,
		PublicKeys: [][]byte{pkData, pkData},
		Signatures: [][]byte{bhSigs0, bhSigs1},
	}
	reply2, err := btcChainAdaptor.CreateUtxoSignedTransaction(&req2)
	assert.Nil(t, err)
	// t.Logf("reply2:%x\n",reply2.SignedTxData)
	assert.Equal(t, expectedSignedTx, reply2.SignedTxData)

	req3 := proto.VerifySignedTransactionRequest{
		Chain:        ChainName,
		Symbol:       Symbol,
		SignedTxData: reply2.SignedTxData,
		Vins:         vin,
	}
	testVinOnOff(t, (*ChainAdaptor).VerifyUtxoSignedTransaction, &req3, vin, func(_reply interface{}, err error) {
		reply := _reply.(*proto.VerifySignedTransactionReply)

		assert.Nil(t, err)
		assert.Equal(t, true, reply.Verified)
	})

}

func TestCreateAndSignTransactionOneTxInUnCompressed(t *testing.T) {
	btcChainAdaptor := newChainAdaptorWithConfig(conf)

	client := btcChainAdaptor.client
	client.compressed = false // mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9 is uncomprssed address

	expectedHash0, _ := hex.DecodeString("cf9b979b88f7971205c6be656e09f416e3d6f76d3579e0481b870ee858d9c4bf")
	expectedTx, _ := hex.DecodeString("0100000001e96161e4de2f2d84872228f4d0732fbe4ab9e6eed23c5ba5db9da636e4f8bd220000000000ffffffff02a4830000000000001976a914eed9944e930bf91b0d0636be81430ec141eae51988acf8a70000000000001976a914ac80df1fa9dd5740c6f17e03b0ca6ce8d871c9a088ac00000000")
	expectedPkData, _ := hex.DecodeString("043cd360fecac46da64c411c6b471d8e147504ed74c2cafd9a29329c63c4eaf1603fb5a230c1ba28d93bb6834989869259d4a4156d33fd5f99075e4b968cdbe8b8")
	expectedSig0, _ := hex.DecodeString("3044022059f877fde8c8a005a070f97793e03cadf66590016d059cc4c782f42ff2f12832022029eaf1777de755306e653a6f8aef330d3ab872c12f51f5104907ce57206d50e301")

	expectedSignedTx, _ := hex.DecodeString("0100000001e96161e4de2f2d84872228f4d0732fbe4ab9e6eed23c5ba5db9da636e4f8bd22000000008a473044022059f877fde8c8a005a070f97793e03cadf66590016d059cc4c782f42ff2f12832022029eaf1777de755306e653a6f8aef330d3ab872c12f51f5104907ce57206d50e30141043cd360fecac46da64c411c6b471d8e147504ed74c2cafd9a29329c63c4eaf1603fb5a230c1ba28d93bb6834989869259d4a4156d33fd5f99075e4b968cdbe8b8ffffffff02a4830000000000001976a914eed9944e930bf91b0d0636be81430ec141eae51988acf8a70000000000001976a914ac80df1fa9dd5740c6f17e03b0ca6ce8d871c9a088ac00000000")
	vin := []*proto.Vin{
		{Hash: "22bdf8e436a69ddba55b3cd2eee6b94abe2f73d0f4282287842d2fdee46161e9", Index: uint32(0), Amount: int64(77200), Address: "mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9"},
	}

	vout := []*proto.Vout{
		{Address: "n3HsmPMAEa2ovEzMNrXKZhSMGvUEfBpHcd", Amount: int64(33700)},
		{Address: "mwF4uCj81VAqKBDjmV2yLESREGDCyZL6Ap", Amount: int64(43000)},
	}

	req1 := proto.CreateUtxoTransactionRequest{
		Chain:  ChainName,
		Symbol: Symbol,
		Vins:   vin,
		Vouts:  vout,
		Fee:    big.NewInt(0).SetInt64(500).String(),
	}

	reply1, err := btcChainAdaptor.CreateUtxoTransaction(&req1)
	assert.Nil(t, err)
	// t.Logf("SignHash[0]:%x\n", reply1.SignHashes[0])
	// t.Logf("TxData:%x\n", reply1.TxData)
	assert.Equal(t, expectedTx, reply1.TxData)
	assert.Equal(t, expectedHash0, reply1.SignHashes[0])
	assert.Equal(t, 1, len(reply1.SignHashes))

	// Sign with Prviate key
	priWif, err := btcutil.DecodeWIF("cMqUKKvaEzKfPnooWk5fRe5c5CXGw2R1KEKefdkktTDSzSGmqfxN")
	assert.Nil(t, err)
	privKey := priWif.PrivKey
	pkData, sig := signOneVin(privKey, reply1.SignHashes[0], client.compressed)
	assert.Equal(t, expectedPkData, pkData)
	assert.Equal(t, expectedSig0, sig)
	// t.Logf("sig:%x\n", sig)
	bhSigs0, err := btcecSigToBHSigBytes(sig)
	assert.Nil(t, err)
	req2 := proto.CreateUtxoSignedTransactionRequest{
		Chain:      ChainName,
		Symbol:     Symbol,
		TxData:     reply1.TxData,
		PublicKeys: [][]byte{pkData},
		Signatures: [][]byte{bhSigs0},
	}
	reply2, err := btcChainAdaptor.CreateUtxoSignedTransaction(&req2)
	assert.Nil(t, err)
	// t.Logf("reply2:%x\n", reply2.SignedTxData)
	assert.Equal(t, expectedSignedTx, reply2.SignedTxData)

	req3 := proto.VerifySignedTransactionRequest{
		Chain:        ChainName,
		Symbol:       Symbol,
		SignedTxData: reply2.SignedTxData,
	}
	testVinOnOff(t, (*ChainAdaptor).VerifyUtxoSignedTransaction, &req3, vin, func(_reply interface{}, err error) {
		reply := _reply.(*proto.VerifySignedTransactionReply)

		assert.Nil(t, err)
		assert.Equal(t, true, reply.Verified)
	})

}

func signOneVin(privKey *btcec.PrivateKey, signHash []byte, compressed bool) (pkData []byte, sigs []byte) {
	pk := (*btcec.PublicKey)(&privKey.PublicKey)
	if compressed {
		pkData = pk.SerializeCompressed()
	} else {
		pkData = pk.SerializeUncompressed()
	}

	sig, err := privKey.Sign(signHash)
	if err != nil {
		return nil, nil
	}
	sig2 := append(sig.Serialize(), byte(txscript.SigHashAll))
	return pkData, sig2
}

func TestCreateSignedTransaction(t *testing.T) {
	btcChainAdaptor := newChainAdaptorWithConfig(conf)

	pkData, err := hex.DecodeString("043cd360fecac46da64c411c6b471d8e147504ed74c2cafd9a29329c63c4eaf1603fb5a230c1ba28d93bb6834989869259d4a4156d33fd5f99075e4b968cdbe8b8")
	require.NoError(t, err)

	expectedSignedTx, err := hex.DecodeString("0100000002cf1d0fc041d8b3ceb1642fdfa262e32f54227f2e6ceb74a515f5482ae090a837000000008a47304402201889ee7147582c897f9d2dcc81bfdb321eddd33bf47820f72c7fa3237425a4cc022057d359d3c495824e48d7f6cc77a9998667f0ea388fb13e2dd24df32bca977a7e0141043cd360fecac46da64c411c6b471d8e147504ed74c2cafd9a29329c63c4eaf1603fb5a230c1ba28d93bb6834989869259d4a4156d33fd5f99075e4b968cdbe8b8fffffffff243031c3431c19f577a32778b635a26834d6b52e552be23df3fab509340d419010000008a47304402200c0221775e00b66a437cae72393c0364197bbf5925a434f4e504647bad2b739802201cde65725a699552029ef00629495542a8a0b9781ce962cb6afc0d313cd7f0400141043cd360fecac46da64c411c6b471d8e147504ed74c2cafd9a29329c63c4eaf1603fb5a230c1ba28d93bb6834989869259d4a4156d33fd5f99075e4b968cdbe8b8ffffffff02902d0100000000001976a91419064bda7eb5049f922a4bca4c24808c6aea948d88ac2c0100000000000017a91410080578e54a2a66efcb55e69b073100d0da47b98700000000")
	require.NoError(t, err)
	sig0, err := hex.DecodeString("304402201889ee7147582c897f9d2dcc81bfdb321eddd33bf47820f72c7fa3237425a4cc022057d359d3c495824e48d7f6cc77a9998667f0ea388fb13e2dd24df32bca977a7e")
	require.NoError(t, err)
	sig1, err := hex.DecodeString("304402200c0221775e00b66a437cae72393c0364197bbf5925a434f4e504647bad2b739802201cde65725a699552029ef00629495542a8a0b9781ce962cb6afc0d313cd7f040")
	require.NoError(t, err)

	txByte, err := hex.DecodeString("0100000002cf1d0fc041d8b3ceb1642fdfa262e32f54227f2e6ceb74a515f5482ae090a8370000000000fffffffff243031c3431c19f577a32778b635a26834d6b52e552be23df3fab509340d4190100000000ffffffff02902d0100000000001976a91419064bda7eb5049f922a4bca4c24808c6aea948d88ac2c0100000000000017a91410080578e54a2a66efcb55e69b073100d0da47b98700000000")
	assert.Nil(t, err)

	bhSigs0, err := btcecSigToBHSigBytes(sig0)
	assert.Nil(t, err)
	bhSigs1, err := btcecSigToBHSigBytes(sig1)
	assert.Nil(t, err)
	req := proto.CreateUtxoSignedTransactionRequest{
		Chain:      ChainName,
		Symbol:     Symbol,
		TxData:     txByte,
		PublicKeys: [][]byte{pkData, pkData},
		Signatures: [][]byte{bhSigs0, bhSigs1},
	}

	reply, err := btcChainAdaptor.CreateUtxoSignedTransaction(&req)
	assert.Nil(t, err)
	assert.Equal(t, expectedSignedTx, reply.SignedTxData)

	req1 := proto.BroadcastTransactionRequest{
		Chain:        ChainName,
		Symbol:       Symbol,
		SignedTxData: reply.SignedTxData,
	}

	reply1, err := btcChainAdaptor.BroadcastTransaction(&req1)
	assert.NotNil(t, err)
	assert.Equal(t, btcjson.RPCErrorCode(-27), err.(*btcjson.RPCError).Code, "unexpected error: ", reply1.Msg)

}

func TestCreateSignedTransactionError(t *testing.T) {
	btcChainAdaptor := newChainAdaptorWithConfig(conf)

	sigByte, err := hex.DecodeString("6730440220486972701a1f11d72c575e0fec145c957c21a89df58a2c5878a4f62253eedaa1022065e13ca5d689c8b1c86bbcc5d30f05340948cc12c5e17c55cc434fca6f495ba10141043cd360fecac46da64c411c6b471d8e147504ed74c2cafd9a29329c63c4eaf1603fb5a230c1ba28d93bb6834989869259d4a4156d33fd5f99075e4b968cdbe8b8")
	assert.Nil(t, err)
	_, err = btcecSigToBHSigBytes(sigByte)
	assert.Error(t, err)

	sigByte1, err := hex.DecodeString("4730440220486972701a1f11d72c575e0fec145c957c21a89df58a2c5878a4f62253eedaa1022065e13ca5d689c8b1c86bbcc5d30f05340948cc12c5e17c55cc434fca6f495ba10141043cd360fecac46da64c411c6b471d8e147504ed74c2cafd9a29329c63c4eaf1603fb5a230c1ba28d93bb6834989869259d4a4156d33fd5f99075e4b968cdbe8b8")
	assert.Nil(t, err)
	_, err = btcecSigToBHSigBytes(sigByte1)
	assert.Error(t, err)

	txByte1, err := hex.DecodeString("0200000001a957ec5a5748134f3c069886ada6d75ed163b61ff6816631f9fe1bf1eb4d4ac90000000000ffffffff020048e801000000001976a91419064bda7eb5049f922a4bca4c24808c6aea948d88ac005307000000000017a91410080578e54a2a66efcb55e69b073100d0da47b98700000000")
	assert.Nil(t, err)

	pubByte := []byte{4, 238, 115, 111, 207, 24, 255, 104, 248, 144, 93, 56, 0, 73, 72, 1, 41, 148, 117, 241, 108, 66, 22, 61, 49, 106, 106, 223, 23, 69, 38, 230, 162, 211, 28, 135, 245, 219, 216, 130, 253, 193, 58, 188, 104, 179, 233, 215, 22, 119, 168, 163, 65, 176, 223, 2, 119, 7, 246, 174, 251, 123, 195, 95, 253}
	req1 := proto.CreateUtxoSignedTransactionRequest{
		Chain:      ChainName,
		Symbol:     Symbol,
		TxData:     txByte1,
		PublicKeys: [][]byte{pubByte},
		Signatures: [][]byte{make([]byte, 33)}, // Invalid signature length
	}

	_, err = btcChainAdaptor.CreateUtxoSignedTransaction(&req1)
	assert.NotNil(t, err)

	sigByteIncorrect, err := hex.DecodeString("304402201889ee7147582c897f9d2dcc81bfdb321eddd33bf47820f72c7fa3237425a4cc022057d359d3c495824e48d7f6cc77a9998667f0ea388fb13e2dd24df32bca977a7e")
	assert.Nil(t, err)
	bhSigIncorrect, err := btcecSigToBHSigBytes(sigByteIncorrect)
	assert.Nil(t, err)

	reqIncorrect := proto.CreateUtxoSignedTransactionRequest{
		Chain:      ChainName,
		Symbol:     Symbol,
		TxData:     txByte1,
		PublicKeys: [][]byte{pubByte},
		Signatures: [][]byte{bhSigIncorrect},
	}

	_, err = btcChainAdaptor.CreateUtxoSignedTransaction(&reqIncorrect)
	assert.NotNil(t, err)

	reqEmptyPubkey := proto.CreateUtxoSignedTransactionRequest{
		Chain:      ChainName,
		Symbol:     Symbol,
		TxData:     txByte1,
		Signatures: [][]byte{bhSigIncorrect},
	}

	_, err = btcChainAdaptor.CreateUtxoSignedTransaction(&reqEmptyPubkey)
	assert.NotNil(t, err)

}

func TestVerifySignedTransactionFromSameAddress(t *testing.T) {
	bz, err := hex.DecodeString("02000000000101266715d8a3aab35fabf484f8e98a68396473ed59f0ec76dddc350ec8d5c9033800000000171600147e6e0170c81cf74bb9a433a2f905546a2766c98bfeffffff0210270000000000001976a914d6c331c38a8b4c966397c4862f86bcfe42cd924588ac6ccdae000000000017a914336b360ddbaf7dd99716bf1c2b92ad233ca9e2aa870247304402200d6fafc20ec2d1a52b62bf6130bbb22678e14c11dec18eea997483cd1cf9340a02201b88b3508454ae7ccf1ccef245f6eafaee6209b5f55bb2c27b17de44d20db67a012103fe6fb3175dd133e95bb8312ef0ad87017cb12f5b62b59fc2dd4b2e39690b2bf3fa2f1700")
	assert.Nil(t, err)

	var req proto.VerifySignedTransactionRequest
	req.Chain = ChainName
	req.Symbol = Symbol
	req.SignedTxData = bz
	vins := []*proto.Vin{
		{
			Amount:  11466020,
			Address: "2MyXNsXWUYmhVth3Rm6DWrDnpfiia79UsPk",
		},
	}

	testVinOnOff(t, (*ChainAdaptor).VerifyUtxoSignedTransaction, &req, vins, func(_reply interface{}, err error) {
		reply := _reply.(*proto.VerifySignedTransactionReply)

		assert.Nil(t, err)
		assert.Equal(t, true, reply.Verified)
	})

	vins[0].Address = "mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9"
	req.Vins = vins
	btcChainAdaptor := newChainAdaptorWithConfig(conf)
	reply, err := btcChainAdaptor.VerifyUtxoSignedTransaction(&req)
	assert.NotNil(t, err)
	assert.Equal(t, proto.ReturnCode_ERROR, reply.Code)
	req.Vins = nil

	bz, err = hex.DecodeString("02000000000101266715d8a3abbbbbbbbbbab35fabf484f8e98a68396473ed59f0ec76dddc350ec8d5c9033800000000171600147e6e0170c81cf74bb9a433a2f905546a2766c98bfeffffff0210270000000000001976a914d6c331c38a8b4c966397c4862f86bcfe42cd924588ac6ccdae000000000017a914336b360ddbaf7dd99716bf1c2b92ad233ca9e2aa870247304402200d6fafc20ec2d1a52b62bf6130bbb22678e14c11dec18eea997483cd1cf9340a02201b88b3508454ae7ccf1ccef245f6eafaee6209b5f55bb2c27b17de44d20db67a012103fe6fb3175dd133e95bb8312ef0ad87017cb12f5b62b59fc2dd4b2e39690b2bf3fa2f17001d3476")
	assert.Nil(t, err)
	req.SignedTxData = bz
	vins[0].Address = "2MyXNsXWUYmhVth3Rm6DWrDnpfiia79UsPk"

	testVinOnOff(t, (*ChainAdaptor).VerifyUtxoSignedTransaction, &req, vins, func(_reply interface{}, err error) {
		reply := _reply.(*proto.VerifySignedTransactionReply)

		assert.NotNil(t, err)
		assert.Equal(t, proto.ReturnCode_ERROR, reply.Code)
	})

	bz1, err := hex.DecodeString("0100000001a957ec5a5748134f3c069886ada6d75ed163b61ff6816631f9fe1bf1eb4d4ac9000000008a4730440220486972701a1f11d72c575e0fec145c957c21a89df58a2c5878a4f62253eedaa1022065e13ca5d689c8b1c86bbcc5d30f05340948cc12c5e17c55cc434fca6f495ba10141043cd360fecac46da64c411c6b471d8e147504ed74c2cafd9a29329c63c4eaf1603fb5a230c1ba28d93bb6834989869259d4a4156d33fd5f99075e4b968cdbe8b8ffffffff020048e801000000001976a91419064bda7eb5049f922a4bca4c24808c6aea948d88ac005307000000000017a91410080578e54a2a66efcb55e69b073100d0da47b98700000000")
	assert.Nil(t, err)
	req.SignedTxData = bz1
	vins[0].Address = "mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9"
	testVinOnOff(t, (*ChainAdaptor).VerifyUtxoSignedTransaction, &req, vins, func(_reply interface{}, err error) {
		reply := _reply.(*proto.VerifySignedTransactionReply)

		assert.Nil(t, err)
		assert.Equal(t, true, reply.Verified)
	})
}

func TestVerifySignedTransactionFromDifferentAddress(t *testing.T) {
	btcChainAdaptor := newChainAdaptorWithConfig(conf)

	bz, err := hex.DecodeString("02000000000101266715d8a3aab35fabf484f8e98a68396473ed59f0ec76dddc350ec8d5c9033800000000171600147e6e0170c81cf74bb9a433a2f905546a2766c98bfeffffff0210270000000000001976a914d6c331c38a8b4c966397c4862f86bcfe42cd924588ac6ccdae000000000017a914336b360ddbaf7dd99716bf1c2b92ad233ca9e2aa870247304402200d6fafc20ec2d1a52b62bf6130bbb22678e14c11dec18eea997483cd1cf9340a02201b88b3508454ae7ccf1ccef245f6eafaee6209b5f55bb2c27b17de44d20db67a012103fe6fb3175dd133e95bb8312ef0ad87017cb12f5b62b59fc2dd4b2e39690b2bf3fa2f1700")
	assert.Nil(t, err)

	var req0 proto.QueryUtxoInsFromDataRequest
	req0.Chain = ChainName
	req0.Symbol = Symbol
	req0.Data = bz

	reply0, err := btcChainAdaptor.QueryUtxoInsFromData(&req0)
	require.NoError(t, err)
	t.Logf("reply0:%v", reply0)

	req := proto.VerifySignedTransactionRequest{
		Chain:        ChainName,
		Symbol:       Symbol,
		SignedTxData: bz,
	}
	vins := []*proto.Vin{
		{
			Amount:  11466020,
			Address: "2MyXNsXWUYmhVth3Rm6DWrDnpfiia79UsPk",
		},
	}

	testVinOnOff(t, (*ChainAdaptor).VerifyUtxoSignedTransaction, &req, vins, func(_reply interface{}, err error) {
		reply := _reply.(*proto.VerifySignedTransactionReply)

		assert.Nil(t, err)
		assert.Equal(t, true, reply.Verified)
	})

	vins[0].Address = "mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9"
	req.Vins = vins
	reply, err := btcChainAdaptor.VerifyUtxoSignedTransaction(&req)
	assert.NotNil(t, err)
	assert.Equal(t, proto.ReturnCode_ERROR, reply.Code)
	req.Vins = nil

	bz, err = hex.DecodeString("02000000000101266715d8a3abbbbbbbbbbab35fabf484f8e98a68396473ed59f0ec76dddc350ec8d5c9033800000000171600147e6e0170c81cf74bb9a433a2f905546a2766c98bfeffffff0210270000000000001976a914d6c331c38a8b4c966397c4862f86bcfe42cd924588ac6ccdae000000000017a914336b360ddbaf7dd99716bf1c2b92ad233ca9e2aa870247304402200d6fafc20ec2d1a52b62bf6130bbb22678e14c11dec18eea997483cd1cf9340a02201b88b3508454ae7ccf1ccef245f6eafaee6209b5f55bb2c27b17de44d20db67a012103fe6fb3175dd133e95bb8312ef0ad87017cb12f5b62b59fc2dd4b2e39690b2bf3fa2f17001d3476")
	assert.Nil(t, err)
	req.SignedTxData = bz
	vins[0].Address = "2MyXNsXWUYmhVth3Rm6DWrDnpfiia79UsPk"

	testVinOnOff(t, (*ChainAdaptor).VerifyUtxoSignedTransaction, &req, vins, func(_reply interface{}, err error) {
		reply := _reply.(*proto.VerifySignedTransactionReply)

		assert.NotNil(t, err)
		assert.Equal(t, proto.ReturnCode_ERROR, reply.Code)
	})

	bz1, err := hex.DecodeString("0100000001a957ec5a5748134f3c069886ada6d75ed163b61ff6816631f9fe1bf1eb4d4ac9000000008a4730440220486972701a1f11d72c575e0fec145c957c21a89df58a2c5878a4f62253eedaa1022065e13ca5d689c8b1c86bbcc5d30f05340948cc12c5e17c55cc434fca6f495ba10141043cd360fecac46da64c411c6b471d8e147504ed74c2cafd9a29329c63c4eaf1603fb5a230c1ba28d93bb6834989869259d4a4156d33fd5f99075e4b968cdbe8b8ffffffff020048e801000000001976a91419064bda7eb5049f922a4bca4c24808c6aea948d88ac005307000000000017a91410080578e54a2a66efcb55e69b073100d0da47b98700000000")
	assert.Nil(t, err)
	req.SignedTxData = bz1
	vins[0].Address = "mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9"
	testVinOnOff(t, (*ChainAdaptor).VerifyUtxoSignedTransaction, &req, vins, func(_reply interface{}, err error) {
		reply := _reply.(*proto.VerifySignedTransactionReply)

		assert.Nil(t, err)
		assert.Equal(t, true, reply.Verified)
	})
}

func TestBroadcastTransaction(t *testing.T) {
	btcChainAdaptor := newChainAdaptorWithConfig(conf)

	// an already exist transactions
	/*
			txFrom := []*TxFrom{
			&TxFrom{
				Address:    "mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9",
				PrivateKey: "cMqUKKvaEzKfPnooWk5fRe5c5CXGw2R1KEKefdkktTDSzSGmqfxN",
				UtxoHash:   "c94a4debf11bfef9316681f61fb663d15ed7a6ad8698063c4f1348575aec57a9",
				Index:      0,
				Balance:    32500000,
			},
		}
		txTo := []*TxTo{
			&TxTo{
				Address: "mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9",
				Amount:  32000000,
			},
			&TxTo{
				Address: "2MthzQgsQ8Rw8vPMtTsrTdqc9HsWiDHM9VY",
				Amount:  480000,
			},
		}
	*/
	bz, err := hex.DecodeString("0100000001a957ec5a5748134f3c069886ada6d75ed163b61ff6816631f9fe1bf1eb4d4ac9000000008a4730440220486972701a1f11d72c575e0fec145c957c21a89df58a2c5878a4f62253eedaa1022065e13ca5d689c8b1c86bbcc5d30f05340948cc12c5e17c55cc434fca6f495ba10141043cd360fecac46da64c411c6b471d8e147504ed74c2cafd9a29329c63c4eaf1603fb5a230c1ba28d93bb6834989869259d4a4156d33fd5f99075e4b968cdbe8b8ffffffff020048e801000000001976a91419064bda7eb5049f922a4bca4c24808c6aea948d88ac005307000000000017a91410080578e54a2a66efcb55e69b073100d0da47b98700000000")
	assert.Nil(t, err)

	req := proto.BroadcastTransactionRequest{
		Chain:        ChainName,
		Symbol:       Symbol,
		SignedTxData: bz,
	}

	reply, err := btcChainAdaptor.BroadcastTransaction(&req)
	assert.NotNil(t, err)
	assert.Equal(t, btcjson.RPCErrorCode(-27), err.(*btcjson.RPCError).Code, "unexpected error: ", reply.Msg)

	// Used a spent hash as UtxoIn
	/*
			txFrom := []*TxFrom{
			&TxFrom{
				Address:    "mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9",
				PrivateKey: "cMqUKKvaEzKfPnooWk5fRe5c5CXGw2R1KEKefdkktTDSzSGmqfxN",
				UtxoHash:   "c94a4debf11bfef9316681f61fb663d15ed7a6ad8698063c4f1348575aec57a9",
				Index:      0,
				Balance:    32500000,
			},
		}
		txTo := []*TxTo{
			&TxTo{
				Address: "2N6EuM7kNUo6GyPuYcmvH2JoRbpUbSkP7RS",
				Amount:  22000000,
			},
			&TxTo{
				Address: "mh1DurxerNqH3nf9p3ivyn7yjgit1ep2Gg",
				Amount:  10480000,
			},
		}
	*/

	bz1, err := hex.DecodeString("0100000001a957ec5a5748134f3c069886ada6d75ed163b61ff6816631f9fe1bf1eb4d4ac9000000008a4730440220229ad9dc1b658238c4542ceb5f9d23474145884b46d583620e6ee6867f798ad9022042d134f8763f4b4b8e5ef890db34996fb80080592e60cf1f3d6dae1ad76450730141043cd360fecac46da64c411c6b471d8e147504ed74c2cafd9a29329c63c4eaf1603fb5a230c1ba28d93bb6834989869259d4a4156d33fd5f99075e4b968cdbe8b8ffffffff0280b14f010000000017a9148e8a131ac4d20dbaafb2a2a09046c975d02e2fb28780e99f00000000001976a9141050cf37fbf38c8ac4a95fe4df0aa7fa0add7e2188ac00000000")
	assert.Nil(t, err)

	req1 := proto.BroadcastTransactionRequest{
		Chain:        ChainName,
		Symbol:       Symbol,
		SignedTxData: bz1,
	}

	reply1, err := btcChainAdaptor.BroadcastTransaction(&req1)
	assert.NotNil(t, err)
	assert.Equal(t, "-25: Missing inputs", reply1.Msg)

	// Noexist utxohash
	/*
		txFrom := []*TxFrom{
			&TxFrom{
				Address:    "mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9",
				PrivateKey: "cMqUKKvaEzKfPnooWk5fRe5c5CXGw2R1KEKefdkktTDSzSGmqfxN",
				UtxoHash:   "a3f1c4c8db42151363eded4ffdd5e01bb73ed6914902fa5173eaa461290deb58", //correct value a3f1c4c8db42151363eded4ffdd5e01bb73ed6914902fa5173eaa461290deb57
				Index:      1,
				Balance:    31998200,
			},
		}
		txTo := []*TxTo{
			&TxTo{
				Address: "2N6EuM7kNUo6GyPuYcmvH2JoRbpUbSkP7RS",
				Amount:  30000000,
			},
		}
	*/

	bz2, err := hex.DecodeString("010000000158eb0d2961a4ea7351fa024991d63eb71be0d5fd4feded63131542dbc8c4f1a3010000008b483045022100b17d83b8e9c9b9d499011e2d65bfcbaabf92a585aa29dbb56e4159f2981372fa0220457c64f5fb6719c72984efc887ee56d6625e51924614639b328288cf6333ce160141043cd360fecac46da64c411c6b471d8e147504ed74c2cafd9a29329c63c4eaf1603fb5a230c1ba28d93bb6834989869259d4a4156d33fd5f99075e4b968cdbe8b8ffffffff0180c3c9010000000017a9148e8a131ac4d20dbaafb2a2a09046c975d02e2fb28700000000")
	assert.Nil(t, err)

	req2 := proto.BroadcastTransactionRequest{
		Chain:        ChainName,
		Symbol:       Symbol,
		SignedTxData: bz2,
	}

	reply2, err := btcChainAdaptor.BroadcastTransaction(&req2)
	assert.NotNil(t, err)
	assert.Equal(t, "-25: Missing inputs", reply2.Msg)

	// incorrect UtxoIn amount
	// utxoin's amount is not considered in constructing  transaction,

	// incorret UtxoIn index
	/*
			txFrom := []*TxFrom{
			&TxFrom{
				Address:    "mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9",
				PrivateKey: "cMqUKKvaEzKfPnooWk5fRe5c5CXGw2R1KEKefdkktTDSzSGmqfxN",
				UtxoHash:   "a3f1c4c8db42151363eded4ffdd5e01bb73ed6914902fa5173eaa461290deb57",
				Index:      0,
				Balance:    31998200, //correct value: 31998200
			},
		}
		txTo := []*TxTo{
			&TxTo{
				Address: "2N6EuM7kNUo6GyPuYcmvH2JoRbpUbSkP7RS",
				Amount:  30000000,
			},
		}

	*/

	bz4, err := hex.DecodeString("010000000157eb0d2961a4ea7351fa024991d63eb71be0d5fd4feded63131542dbc8c4f1a3000000008b483045022100f2bbaa12b2329a83e2bc83a80d4bf0578c676cdea4f59b5ee4f39eaae49c6c55022026cc78a150a7038f29253dc1a9d5623b3ac525e7cf39e689d3e5e0e256e4ea2b0141043cd360fecac46da64c411c6b471d8e147504ed74c2cafd9a29329c63c4eaf1603fb5a230c1ba28d93bb6834989869259d4a4156d33fd5f99075e4b968cdbe8b8ffffffff0180c3c9010000000017a9148e8a131ac4d20dbaafb2a2a09046c975d02e2fb28700000000")
	assert.Nil(t, err)

	req4 := proto.BroadcastTransactionRequest{
		Chain:        ChainName,
		Symbol:       Symbol,
		SignedTxData: bz4,
	}

	reply4, err := btcChainAdaptor.BroadcastTransaction(&req4)
	assert.NotNil(t, err)
	assert.Equal(t, "-25: Missing inputs", reply4.Msg)

	// Success once @2019/7/16
	/*
		txFrom := []*TxFrom{
			&TxFrom{
				Address:    "mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9",
				PrivateKey: "cMqUKKvaEzKfPnooWk5fRe5c5CXGw2R1KEKefdkktTDSzSGmqfxN",
				UtxoHash:   "c121a91a45d3692a24eb5faea1d756e24a000ee38ba28afc223bba04ae42be3c",
				Index:      1,
				Balance:    100000,
			},
		}
		txTo := []*TxTo{
			&TxTo{
				Address: "mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9",
				Amount:  98000,
			},
		}
	*/
	// bz5, err := hex.DecodeString("01000000013cbe42ae04ba3b22fc8aa28be30e004ae256d7a1ae5feb242a69d3451aa921c1010000008a47304402202af8e50143fd2752e7d1e602bcd40f32ce1e275ce565ddadd48a61bdf43cc94a0220229aaf84d87e8de2d61aaa93ebd35bc614052f9cb4d70507821e1f5599493d450141043cd360fecac46da64c411c6b471d8e147504ed74c2cafd9a29329c63c4eaf1603fb5a230c1ba28d93bb6834989869259d4a4156d33fd5f99075e4b968cdbe8b8ffffffff01d07e0100000000001976a91419064bda7eb5049f922a4bca4c24808c6aea948d88ac00000000")
	// assert.Nil(t, err)
	//
	// req5 := proto.BroadcastTransactionRequest{
	//	Symbol:       symbol,
	//	SignedTxData: bz5,
	// }
	//
	// reply5, err := btcChainAdaptor.BroadcastTransaction(&req5)
	// assert.Nil(t, err)
	// assert.Equal(t, proto.ReturnCode_SUCCESS, reply5.Code)

	/*
		txFrom := []*TxFrom{
			&TxFrom{
				Address:    "mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9",
				PrivateKey: "cMqUKKvaEzKfPnooWk5fRe5c5CXGw2R1KEKefdkktTDSzSGmqfxN",
				UtxoHash:   "1917ded0c5be6523cf2e5bdbd68e55887bd78a9bc17f10cf1222fc30d05d9069",
				Index:      0,
				Balance:    98000,
			},
		}
		txTo := []*TxTo{
			&TxTo{
				Address: "mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9",
				Amount:  43000,
			},
			&TxTo{
				Address: "mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9",
				Amount:  45000,
			},
			&TxTo{
				Address: "2MthzQgsQ8Rw8vPMtTsrTdqc9HsWiDHM9VY",
				Amount:  500,
			},
			&TxTo{
				Address: "2MthzQgsQ8Rw8vPMtTsrTdqc9HsWiDHM9VY",
				Amount:  200,
			},

		}
	*/

	bz6, err := hex.DecodeString("010000000169905dd030fc2212cf107fc19b8ad77b88558ed6db5b2ecf2365bec5d0de1719000000008a473044022077a3a814794255028173f1177907b65567a275b1cf4bab3872af8ae1f379a627022023dd12693b41f34898296d8ed2843986a717a0a9aa3ed9c3cfeb1a73833219450141043cd360fecac46da64c411c6b471d8e147504ed74c2cafd9a29329c63c4eaf1603fb5a230c1ba28d93bb6834989869259d4a4156d33fd5f99075e4b968cdbe8b8ffffffff04f8a70000000000001976a91419064bda7eb5049f922a4bca4c24808c6aea948d88acc8af0000000000001976a91419064bda7eb5049f922a4bca4c24808c6aea948d88acf40100000000000017a91410080578e54a2a66efcb55e69b073100d0da47b987c80000000000000017a91410080578e54a2a66efcb55e69b073100d0da47b98700000000")
	assert.Nil(t, err)

	req6 := proto.BroadcastTransactionRequest{
		Chain:        ChainName,
		Symbol:       Symbol,
		SignedTxData: bz6,
	}

	reply6, err := btcChainAdaptor.BroadcastTransaction(&req6)
	assert.NotNil(t, err)
	assert.Equal(t, btcjson.RPCErrorCode(-27), err.(*btcjson.RPCError).Code, "unexpected error: ", reply6.Msg)

}

func TestBroadcastTransaction2(t *testing.T) {
	btcChainAdaptor := newChainAdaptorWithConfig(conf)

	// Success once @2019/8/15
	// vin := []*proto.UtxoIn{
	//	&proto.UtxoIn{Hash: "37a890e02a48f515a574eb6c2e7f22542fe362a2df2f64b1ceb3d841c00f1dcf", Index: uint32(0), Amount: int64(33000)},
	//	&proto.UtxoIn{Hash: "19d4409350ab3fdf23be52e5526b4d83265a638b77327a579fc131341c0343f2", Index: uint32(1), Amount: int64(45000)},
	//
	// }
	//
	// vout := []*proto.Vout{
	//	&proto.Vout{Address: "mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9", Amount: int64(77200)},
	//	&proto.Vout{Address: "2MthzQgsQ8Rw8vPMtTsrTdqc9HsWiDHM9VY", Amount: int64(300)},
	// }

	bz, err := hex.DecodeString("0100000002cf1d0fc041d8b3ceb1642fdfa262e32f54227f2e6ceb74a515f5482ae090a837000000008a47304402201889ee7147582c897f9d2dcc81bfdb321eddd33bf47820f72c7fa3237425a4cc022057d359d3c495824e48d7f6cc77a9998667f0ea388fb13e2dd24df32bca977a7e0141043cd360fecac46da64c411c6b471d8e147504ed74c2cafd9a29329c63c4eaf1603fb5a230c1ba28d93bb6834989869259d4a4156d33fd5f99075e4b968cdbe8b8fffffffff243031c3431c19f577a32778b635a26834d6b52e552be23df3fab509340d419010000008a47304402200c0221775e00b66a437cae72393c0364197bbf5925a434f4e504647bad2b739802201cde65725a699552029ef00629495542a8a0b9781ce962cb6afc0d313cd7f0400141043cd360fecac46da64c411c6b471d8e147504ed74c2cafd9a29329c63c4eaf1603fb5a230c1ba28d93bb6834989869259d4a4156d33fd5f99075e4b968cdbe8b8ffffffff02902d0100000000001976a91419064bda7eb5049f922a4bca4c24808c6aea948d88ac2c0100000000000017a91410080578e54a2a66efcb55e69b073100d0da47b98700000000")
	require.NoError(t, err)
	req := proto.BroadcastTransactionRequest{
		Chain:        ChainName,
		Symbol:       Symbol,
		SignedTxData: bz,
	}

	reply, err := btcChainAdaptor.BroadcastTransaction(&req)
	assert.NotNil(t, err)
	assert.Equal(t, btcjson.RPCErrorCode(-27), err.(*btcjson.RPCError).Code, "unexpected error: ", reply.Msg)
}

// TestUseUnconfirmedUtxo success only once@7/18, will fail hereafter
func TestUseUnconfirmedUtxo(t *testing.T) {
	btcChainAdaptor := newChainAdaptorWithConfig(conf)

	/*
		txFrom := []*TxFrom{
			&TxFrom{
				Address:    "mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9",
				PrivateKey: "cMqUKKvaEzKfPnooWk5fRe5c5CXGw2R1KEKefdkktTDSzSGmqfxN",
				UtxoHash:   "19d4409350ab3fdf23be52e5526b4d83265a638b77327a579fc131341c0343f2",
				Index:      0,
				Balance:    43000,
			},
		}
		txTo := []*TxTo{
			&TxTo{
				Address: "mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9",
				Amount:  42000,
			},

			&TxTo{
				Address: "2MthzQgsQ8Rw8vPMtTsrTdqc9HsWiDHM9VY",
				Amount:  500,
			},
		}
	*/

	bz, err := hex.DecodeString("0100000001f243031c3431c19f577a32778b635a26834d6b52e552be23df3fab509340d419000000008a47304402203a6c87fbd1b39cf17071ffa560875d483c79fc769872ed83bca395fde538e21a02207adeee2f1c8ff4c2cfc73a1bb8a2748246af7416a2f02f7be8e6bd9345d1d5510141043cd360fecac46da64c411c6b471d8e147504ed74c2cafd9a29329c63c4eaf1603fb5a230c1ba28d93bb6834989869259d4a4156d33fd5f99075e4b968cdbe8b8ffffffff0210a40000000000001976a91419064bda7eb5049f922a4bca4c24808c6aea948d88acf40100000000000017a91410080578e54a2a66efcb55e69b073100d0da47b98700000000")
	assert.Nil(t, err)

	req := proto.BroadcastTransactionRequest{
		Chain:        ChainName,
		Symbol:       Symbol,
		SignedTxData: bz,
	}

	reply, err := btcChainAdaptor.BroadcastTransaction(&req)
	assert.NotNil(t, err)
	assert.Equal(t, proto.ReturnCode_ERROR, reply.Code)

	bz1, err := hex.DecodeString("0100000001eecdb4d1f22fad8823ea87a651e1c0dc8f634f5474793947a1b63952d9a51542000000008a47304402201c58f959aa08b2c9ed1917868d17f714d4f69db1fc678de50bb1469be648a2f30220367ce60166e908f4cd4a6fcf334731fb177585af0660dd4874ee5222326c78820141043cd360fecac46da64c411c6b471d8e147504ed74c2cafd9a29329c63c4eaf1603fb5a230c1ba28d93bb6834989869259d4a4156d33fd5f99075e4b968cdbe8b8ffffffff02b8880000000000001976a91419064bda7eb5049f922a4bca4c24808c6aea948d88acf40100000000000017a91410080578e54a2a66efcb55e69b073100d0da47b98700000000")
	assert.Nil(t, err)

	req1 := proto.BroadcastTransactionRequest{
		Chain:        ChainName,
		Symbol:       Symbol,
		SignedTxData: bz1,
	}

	reply1, err := btcChainAdaptor.BroadcastTransaction(&req1)
	assert.NotNil(t, err)
	assert.Equal(t, proto.ReturnCode_ERROR, reply1.Code)

}

func TestCreateSignedTransactionForCompressedAddressAndTx(t *testing.T) {
	btcChainAdaptor := newChainAdaptorWithConfig(conf)

	txByte := []byte{1, 0, 0, 0, 1, 170, 225, 76, 238, 195, 200, 12, 124, 86, 141, 46, 163, 251, 49, 185, 133, 14, 66, 207, 209, 185, 188, 36, 59, 236, 105, 32, 216, 253, 93, 73, 160, 1, 0, 0, 0, 0, 255, 255, 255, 255, 1, 57, 26, 105, 0, 0, 0, 0, 0, 25, 118, 169, 20, 167, 58, 186, 103, 199, 10, 81, 119, 192, 253, 109, 242, 162, 134, 139, 67, 213, 109, 27, 211, 136, 172, 0, 0, 0, 0}
	pubByte := []byte{4, 238, 115, 111, 207, 24, 255, 104, 248, 144, 93, 56, 0, 73, 72, 1, 41, 148, 117, 241, 108, 66, 22, 61, 49, 106, 106, 223, 23, 69, 38, 230, 162, 211, 28, 135, 245, 219, 216, 130, 253, 193, 58, 188, 104, 179, 233, 215, 22, 119, 168, 163, 65, 176, 223, 2, 119, 7, 246, 174, 251, 123, 195, 95, 253}
	sigByte, _ := hex.DecodeString("72e5f597f16485eef5f725479dd058656159392c64fde66b20e8a7a0bded9328729812f7a98c2ba3870f3acf78ada6c87d855f9e3878943b16db6060d654c7af00")
	signHash, _ := hex.DecodeString("124908e664d5df4502200a88bdf489fcd1e27fe5cd891724cf3cc63d873daf7e")
	sig := btcec.Signature{
		R: new(big.Int).SetBytes(sigByte[0:32]),
		S: new(big.Int).SetBytes(sigByte[32:64]),
	}
	pubKey, err := btcec.ParsePubKey(pubByte, btcec.S256())
	assert.Nil(t, err)

	// Test compressed pubkey
	reqAddress := proto.ConvertAddressRequest{
		Chain:     ChainName,
		PublicKey: pubKey.SerializeCompressed(),
	}
	replyAddress, err := btcChainAdaptor.ConvertAddress(&reqAddress)
	assert.Nil(t, err)
	assert.Equal(t, "n17KByno4FbTenvfQkp3rMjaSSfTWUKUKa", replyAddress.Address)

	assert.True(t, sig.Verify(signHash, pubKey))

	// Test compressed pubkey
	req := proto.CreateUtxoSignedTransactionRequest{
		Chain:      ChainName,
		Symbol:     Symbol,
		TxData:     txByte,
		PublicKeys: [][]byte{pubKey.SerializeCompressed()},
		Signatures: [][]byte{sigByte},
	}
	reply, err := btcChainAdaptor.CreateUtxoSignedTransaction(&req)
	assert.Nil(t, err)
	reqVerify := proto.VerifySignedTransactionRequest{
		Chain:        ChainName,
		Symbol:       Symbol,
		Addresses:    []string{"n17KByno4FbTenvfQkp3rMjaSSfTWUKUKa"},
		SignedTxData: reply.SignedTxData,
	}
	vins := []*proto.Vin{
		{
			Amount:  0,
			Address: "n17KByno4FbTenvfQkp3rMjaSSfTWUKUKa",
		},
	}

	testVinOnOff(t, (*ChainAdaptor).VerifyUtxoSignedTransaction, &reqVerify, vins, func(_reply interface{}, err error) {
		reply := _reply.(*proto.VerifySignedTransactionReply)

		assert.Nil(t, err)
		assert.True(t, reply.Verified)
	})
}

func TestCreateAndSignedTransactioFromDifferentAddressInCompress(t *testing.T) {
	btcChainAdaptor := newChainAdaptorWithConfig(conf)

	client := btcChainAdaptor.client
	client.compressed = true
	expectedTxData := "010000000282f957d1598291a946d34a114e5391b166cfbf8228325a12fe703cf8db18343f0000000000ffffffff82f957d1598291a946d34a114e5391b166cfbf8228325a12fe703cf8db18343f0100000000ffffffff02b4270100000000001976a91419064bda7eb5049f922a4bca4c24808c6aea948d88acf40100000000000017a91410080578e54a2a66efcb55e69b073100d0da47b98700000000"
	expectedHash1 := "890c6bb02e75bebf6ae1f92cb3138bb11ec7dd3035938d1224384ad861c52041"
	expectedPkData1 := "0395a2c1cddab943e4eef857968e2a5cc2e587c92b598e8517830869c5f3203f3b"
	expectedSig1 := "304402206099bad8198530cf2608ccb62b580d5013381c530feff6136879f2c99221615f02203132c0a881e4e1632fba238f1724fbc69c65a30b0e7fe7ca1512a281ab0e665201"
	expectedHash2 := "8d295b75352045f64513919470ba46c42ef3945b7513ed3dac15ae7042e4319c"
	expectedPkData2 := "02f06a16cea42494b5dbb3701eb43d4e96e6e8bb71c5f360abaed9e3ccd55c5a1d"
	expectedSig2 := "30440220585d7060a8810a07259f5b1291e82f640d02d2f536ddd749b9834a93d3d5b53b02201bb447ccf5f1694e308a544d7189f77378f038b6b20d26a48d425a9f041f893a01"
	expectedHash := "4614f1fc0e21e21c78d694dd4bc2132cc49b843311cda36e023af7f2b423a549"
	expectedSignedTx := "010000000282f957d1598291a946d34a114e5391b166cfbf8228325a12fe703cf8db18343f000000006a47304402206099bad8198530cf2608ccb62b580d5013381c530feff6136879f2c99221615f02203132c0a881e4e1632fba238f1724fbc69c65a30b0e7fe7ca1512a281ab0e665201210395a2c1cddab943e4eef857968e2a5cc2e587c92b598e8517830869c5f3203f3bffffffff82f957d1598291a946d34a114e5391b166cfbf8228325a12fe703cf8db18343f010000006a4730440220585d7060a8810a07259f5b1291e82f640d02d2f536ddd749b9834a93d3d5b53b02201bb447ccf5f1694e308a544d7189f77378f038b6b20d26a48d425a9f041f893a012102f06a16cea42494b5dbb3701eb43d4e96e6e8bb71c5f360abaed9e3ccd55c5a1dffffffff02b4270100000000001976a91419064bda7eb5049f922a4bca4c24808c6aea948d88acf40100000000000017a91410080578e54a2a66efcb55e69b073100d0da47b98700000000"

	vin := []*proto.Vin{
		{Hash: "3f3418dbf83c70fe125a322882bfcf66b191534e114ad346a9918259d157f982", Index: uint32(0), Amount: int64(33700), Address: "n3HsmPMAEa2ovEzMNrXKZhSMGvUEfBpHcd"},
		{Hash: "3f3418dbf83c70fe125a322882bfcf66b191534e114ad346a9918259d157f982", Index: uint32(1), Amount: int64(43000), Address: "mwF4uCj81VAqKBDjmV2yLESREGDCyZL6Ap"},
	}

	vout := []*proto.Vout{
		{Address: "mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9", Amount: int64(75700)},
		{Address: "2MthzQgsQ8Rw8vPMtTsrTdqc9HsWiDHM9VY", Amount: int64(500)},
	}

	req := proto.CreateUtxoTransactionRequest{
		Chain:  ChainName,
		Symbol: Symbol,
		Vins:   vin,
		Vouts:  vout,
		Fee:    big.NewInt(500).String(),
	}

	reply1, err := btcChainAdaptor.CreateUtxoTransaction(&req)
	assert.Nil(t, err)
	assert.Equal(t, expectedTxData, hex.EncodeToString(reply1.TxData))
	assert.Equal(t, expectedHash1, hex.EncodeToString(reply1.SignHashes[0]))
	assert.Equal(t, expectedHash2, hex.EncodeToString(reply1.SignHashes[1]))
	// t.Logf("reply.TxData:%s", hex.EncodeToString(reply1.TxData))
	// t.Logf("hash[0]:%s", hex.EncodeToString(reply1.SignHashes[0]))
	// t.Logf("hash[1]:%s", hex.EncodeToString(reply1.SignHashes[1]))

	// Sign with Prviate key
	priWif1, err := btcutil.DecodeWIF("cMahea7zqjxrtgAbB7LSGbcQUr1uX1ojuat9jZoeiemEvoHeHdkv")
	assert.Nil(t, err)
	privKey1 := priWif1.PrivKey
	pkData1, sigs1 := signOneVin(privKey1, reply1.SignHashes[0], client.compressed)
	// t.Logf("pkData1:%v\n", hex.EncodeToString(pkData1))
	// t.Logf("sgis1:%v\n", hex.EncodeToString(sigs1))
	assert.Equal(t, expectedPkData1, hex.EncodeToString(pkData1))
	assert.Equal(t, expectedSig1, hex.EncodeToString(sigs1))

	originSig1, err := btcec.ParseSignature(sigs1, btcec.S256())
	assert.Nil(t, err)
	r, s := originSig1.R.Bytes(), originSig1.S.Bytes()
	sig := make([]byte, 65)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)
	sig[64] = 0

	// creat sigscript and verify
	r1 := new(big.Int).SetBytes(sig[0:32])
	s1 := new(big.Int).SetBytes(sig[32:64])
	assert.True(t, originSig1.R.Cmp(r1) == 0)
	assert.True(t, originSig1.S.Cmp(s1) == 0)

	priWif2, err := btcutil.DecodeWIF("cMahea7zqjxrtgAbB7LSGbcQUr1uX1ojuat9jZoeiemEwJD8z6eD")
	assert.Nil(t, err)
	privKey2 := priWif2.PrivKey
	pkData2, sigs2 := signOneVin(privKey2, reply1.SignHashes[1], client.compressed)
	// t.Logf("pkData2:%v\n", hex.EncodeToString(pkData2))
	// t.Logf("sgis2:%v\n", hex.EncodeToString(sigs2))
	assert.Equal(t, expectedPkData2, hex.EncodeToString(pkData2))
	assert.Equal(t, expectedSig2, hex.EncodeToString(sigs2))
	originSig2, err := btcec.ParseSignature(sigs2, btcec.S256())
	assert.Nil(t, err)
	r, s = originSig2.R.Bytes(), originSig2.S.Bytes()
	sig = make([]byte, 65)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)
	sig[64] = 0

	// creat sigscript and verify
	r1 = new(big.Int).SetBytes(sig[0:32])
	s1 = new(big.Int).SetBytes(sig[32:64])
	assert.True(t, originSig2.R.Cmp(r1) == 0)
	assert.True(t, originSig2.S.Cmp(s1) == 0)

	bhSigs1, err := btcecSigToBHSigBytes(sigs1)
	assert.Nil(t, err)
	bhSigs2, err := btcecSigToBHSigBytes(sigs2)
	assert.Nil(t, err)

	req2 := proto.CreateUtxoSignedTransactionRequest{
		Chain:      ChainName,
		Symbol:     Symbol,
		TxData:     reply1.TxData,
		PublicKeys: [][]byte{pkData1, pkData2},
		Signatures: [][]byte{bhSigs1, bhSigs2},
	}

	reply2, err := btcChainAdaptor.CreateUtxoSignedTransaction(&req2)
	assert.Nil(t, err)
	// t.Logf("reply2:%x\n", reply2.SignedTxData)
	assert.Equal(t, expectedSignedTx, hex.EncodeToString(reply2.SignedTxData))

	res, err := btcChainAdaptor.decodeTx(reply2.SignedTxData, vin, true)
	assert.Nil(t, err)
	assert.Equal(t, expectedHash, res.Hash)

	req3 := proto.VerifySignedTransactionRequest{
		Chain:        ChainName,
		Symbol:       Symbol,
		SignedTxData: reply2.SignedTxData,
	}
	testVinOnOff(t, (*ChainAdaptor).VerifyUtxoSignedTransaction, &req3, vin, func(_reply interface{}, err error) {
		reply := _reply.(*proto.VerifySignedTransactionReply)

		assert.Nil(t, err)
		assert.Equal(t, true, reply.Verified)
	})
}

func TestBroadcastTransaction3(t *testing.T) {
	btcChainAdaptor := newChainAdaptorWithConfig(conf)

	// Success once @2019/11/23
	/*
			vin := []*proto.Vin{
			&proto.Vin{Hash: "3f3418dbf83c70fe125a322882bfcf66b191534e114ad346a9918259d157f982", Index: uint32(0), Amount: int64(33700), Address: "n3HsmPMAEa2ovEzMNrXKZhSMGvUEfBpHcd"},
			&proto.Vin{Hash: "3f3418dbf83c70fe125a322882bfcf66b191534e114ad346a9918259d157f982", Index: uint32(1), Amount: int64(43000), Address: "mwF4uCj81VAqKBDjmV2yLESREGDCyZL6Ap"},
		}

		vout := []*proto.Vout{
			&proto.Vout{Address: "mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9", Amount: int64(75700)},
			&proto.Vout{Address: "2MthzQgsQ8Rw8vPMtTsrTdqc9HsWiDHM9VY", Amount: int64(500)},
		}
	*/

	bz, err := hex.DecodeString("010000000282f957d1598291a946d34a114e5391b166cfbf8228325a12fe703cf8db18343f000000006a47304402206099bad8198530cf2608ccb62b580d5013381c530feff6136879f2c99221615f02203132c0a881e4e1632fba238f1724fbc69c65a30b0e7fe7ca1512a281ab0e665201210395a2c1cddab943e4eef857968e2a5cc2e587c92b598e8517830869c5f3203f3bffffffff82f957d1598291a946d34a114e5391b166cfbf8228325a12fe703cf8db18343f010000006a4730440220585d7060a8810a07259f5b1291e82f640d02d2f536ddd749b9834a93d3d5b53b02201bb447ccf5f1694e308a544d7189f77378f038b6b20d26a48d425a9f041f893a012102f06a16cea42494b5dbb3701eb43d4e96e6e8bb71c5f360abaed9e3ccd55c5a1dffffffff02b4270100000000001976a91419064bda7eb5049f922a4bca4c24808c6aea948d88acf40100000000000017a91410080578e54a2a66efcb55e69b073100d0da47b98700000000")
	require.NoError(t, err)
	req := proto.BroadcastTransactionRequest{
		Chain:        ChainName,
		Symbol:       Symbol,
		SignedTxData: bz,
	}

	reply, err := btcChainAdaptor.BroadcastTransaction(&req)
	// assert.Nil(t, err)
	// assert.Equal(t, proto.ReturnCode_SUCCESS, reply.Code)
	assert.NotNil(t, err)
	assert.Equal(t, btcjson.RPCErrorCode(-27), err.(*btcjson.RPCError).Code, "unexpected error: ", reply.Msg)
}

// TestCreateAndSingedTransaction2 success @2020/1/6
func TestCreateAndSingedTransaction2(t *testing.T) {
	btcChainAdaptor := newChainAdaptorWithConfig(conf)

	client := btcChainAdaptor.client
	expectedRawDataStr := "010000000145138613310ded962b679f31268faaf73ca2cdf1e6e44c61d16cd21d97be71ed0000000000ffffffff02a0bb0d00000000001976a914eed9944e930bf91b0d0636be81430ec141eae51988acd07e0100000000001976a914ac80df1fa9dd5740c6f17e03b0ca6ce8d871c9a088ac00000000"
	expectedSignHashStr := "1e589854304006c8669ca7fd9fbef23594eb33484886a4423ca8dc62908fd0be"
	expectedPkDataStr := "0395a2c1cddab943e4eef857968e2a5cc2e587c92b598e8517830869c5f3203f3b"
	expectedSigStr := "3045022100af824d0774c234628a0302a307b9abd2ed6bffc60d59eb6f85ba1d8dd445ea080220492ab01d0ed4dc565fa857613e33ec1663c6c46d580357aa349d6bc0fea5437301"
	expectedSignedTxStr := "010000000145138613310ded962b679f31268faaf73ca2cdf1e6e44c61d16cd21d97be71ed000000006b483045022100af824d0774c234628a0302a307b9abd2ed6bffc60d59eb6f85ba1d8dd445ea080220492ab01d0ed4dc565fa857613e33ec1663c6c46d580357aa349d6bc0fea5437301210395a2c1cddab943e4eef857968e2a5cc2e587c92b598e8517830869c5f3203f3bffffffff02a0bb0d00000000001976a914eed9944e930bf91b0d0636be81430ec141eae51988acd07e0100000000001976a914ac80df1fa9dd5740c6f17e03b0ca6ce8d871c9a088ac00000000"
	expectedHashStr := "7b99b5913b6f026d4c91e540fca678cbc119e66bf89329747aa2f48ace484080"
	// reverseHashStr := "804048ce8af4a27a742993f86be619c1cb78a6fc40e5914c6d026f3b91b5997b"

	vin := []*proto.Vin{
		{Hash: "ed71be971dd26cd1614ce4e6f1cda23cf7aa8f26319f672b96ed0d3113861345", Index: uint32(0), Amount: int64(1000000), Address: "n3HsmPMAEa2ovEzMNrXKZhSMGvUEfBpHcd"},
	}
	vout := []*proto.Vout{
		{Address: "n3HsmPMAEa2ovEzMNrXKZhSMGvUEfBpHcd", Amount: int64(900000)},
		{Address: "mwF4uCj81VAqKBDjmV2yLESREGDCyZL6Ap", Amount: int64(98000)},
	}

	req1 := proto.CreateUtxoTransactionRequest{
		Chain:  ChainName,
		Symbol: Symbol,
		Vins:   vin,
		Vouts:  vout,
		Fee:    big.NewInt(0).SetInt64(2000).String(),
	}

	reply1, err := btcChainAdaptor.CreateUtxoTransaction(&req1)
	assert.Nil(t, err)
	assert.Equal(t, expectedRawDataStr, hex.EncodeToString(reply1.TxData))
	assert.Equal(t, 1, len(reply1.SignHashes))
	assert.Equal(t, expectedSignHashStr, hex.EncodeToString(reply1.SignHashes[0]))

	//
	req2 := &proto.QueryTransactionFromDataRequest{
		Chain:   ChainName,
		Symbol:  Symbol,
		RawData: reply1.TxData,
	}
	vins := []*proto.Vin{
		{Hash: "ed71be971dd26cd1614ce4e6f1cda23cf7aa8f26319f672b96ed0d3113861345", Index: uint32(0), Amount: int64(1000000), Address: "n3HsmPMAEa2ovEzMNrXKZhSMGvUEfBpHcd"},
	}

	testVinOnOff(t, (*ChainAdaptor).QueryUtxoTransactionFromData, req2, vins, func(replyA interface{}, err error) {
		reply := replyA.(*proto.QueryUtxoTransactionReply)

		assert.Nil(t, err)
		assert.Equal(t, reply1.SignHashes, reply.SignHashes)
		assert.Equal(t, 1, len(reply1.SignHashes))
		assert.Equal(t, expectedSignHashStr, hex.EncodeToString(reply.SignHashes[0]))
	})

	// Sign with Prviate key
	priWif, err := btcutil.DecodeWIF("cMahea7zqjxrtgAbB7LSGbcQUr1uX1ojuat9jZoeiemEvoHeHdkv")
	assert.Nil(t, err)
	privKey := priWif.PrivKey
	pkData, sig0 := signOneVin(privKey, reply1.SignHashes[0], client.compressed)
	assert.Equal(t, expectedPkDataStr, hex.EncodeToString(pkData))
	assert.Equal(t, expectedSigStr, hex.EncodeToString(sig0))

	originSig0, err := btcec.ParseSignature(sig0, btcec.S256())
	assert.Nil(t, err)
	r, s := originSig0.R.Bytes(), originSig0.S.Bytes()
	sig := make([]byte, 65)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)
	sig[64] = 0

	// creat sigscript and verify
	r1 := new(big.Int).SetBytes(sig[0:32])
	s1 := new(big.Int).SetBytes(sig[32:64])
	assert.True(t, originSig0.R.Cmp(r1) == 0)
	assert.True(t, originSig0.S.Cmp(s1) == 0)

	req3 := proto.CreateUtxoSignedTransactionRequest{
		Chain:      ChainName,
		Symbol:     Symbol,
		TxData:     reply1.TxData, // rawData
		PublicKeys: [][]byte{pkData},
		Signatures: [][]byte{sig},
	}

	reply3, err := btcChainAdaptor.CreateUtxoSignedTransaction(&req3)
	assert.Nil(t, err)
	hash3, err := chainhash.NewHash(reply3.Hash)
	assert.Nil(t, err)
	assert.Equal(t, expectedSignedTxStr, hex.EncodeToString(reply3.SignedTxData))
	assert.Equal(t, expectedHashStr, hash3.String())

	req4 := proto.QueryTransactionFromSignedDataRequest{
		Chain:        ChainName,
		Symbol:       Symbol,
		SignedTxData: reply3.SignedTxData,
	}

	testVinOnOff(t, (*ChainAdaptor).QueryUtxoTransactionFromSignedData, &req4, vin, func(_reply interface{}, err error) {
		reply := _reply.(*proto.QueryUtxoTransactionReply)

		assert.Equal(t, hash3.String(), reply.TxHash)
	})

	req5 := proto.BroadcastTransactionRequest{
		Chain:        ChainName,
		Symbol:       Symbol,
		SignedTxData: reply3.SignedTxData,
	}

	reply5, err := btcChainAdaptor.BroadcastTransaction(&req5)
	assert.NotNil(t, err)
	assert.Equal(t, btcjson.RPCErrorCode(-27), err.(*btcjson.RPCError).Code, "unexpected error: ", reply5.Msg)
	// assert.Equal(t, expectedHashStr, reply5.TxHash)

}

// TestCreateAndSingedTransaction2 success @2020/1/6
func TestCreateAndSingedTransaction3(t *testing.T) {
	btcChainAdaptor := newChainAdaptorWithConfig(conf)

	client := btcChainAdaptor.client
	expectedRawDataStr := "0100000001804048ce8af4a27a742993f86be619c1cb78a6fc40e5914c6d026f3b91b5997b0000000000ffffffff01b8b70d00000000001976a914f5e14e43e474738731b7f1ec64b8d090e0e1f9fe88ac00000000"
	expectedSignHashStr := "88ff62637e4ada4d18843cf988c02c4a6e97264b18e99806e6956fc29a83009c"
	expectedPkDataStr := "0395a2c1cddab943e4eef857968e2a5cc2e587c92b598e8517830869c5f3203f3b"
	expectedSigStr := "3045022100a2d2aed29be2ec1f07c92ece9c8377c592d6c232288f800202489a1847e7c92b022054465020298d2f65487b86e93b58d90e0b9c29558e118af61dc19835a984290101"
	expectedSignedTxStr := "0100000001804048ce8af4a27a742993f86be619c1cb78a6fc40e5914c6d026f3b91b5997b000000006b483045022100a2d2aed29be2ec1f07c92ece9c8377c592d6c232288f800202489a1847e7c92b022054465020298d2f65487b86e93b58d90e0b9c29558e118af61dc19835a984290101210395a2c1cddab943e4eef857968e2a5cc2e587c92b598e8517830869c5f3203f3bffffffff01b8b70d00000000001976a914f5e14e43e474738731b7f1ec64b8d090e0e1f9fe88ac00000000"
	expectedHashStr := "942a605a1d57e974c02a6a63be8cb88046c0638958ce05ebea2700407dbd1f6f"
	// reverseHashStr := "804048ce8af4a27a742993f86be619c1cb78a6fc40e5914c6d026f3b91b5997b"

	vin := []*proto.Vin{
		{Hash: "7b99b5913b6f026d4c91e540fca678cbc119e66bf89329747aa2f48ace484080", Index: uint32(0), Amount: int64(900000), Address: "n3HsmPMAEa2ovEzMNrXKZhSMGvUEfBpHcd"},
	}
	vout := []*proto.Vout{
		{Address: "n3w3kZvxmHKrjvg5wP1NEgwkVCgsUsCgm4", Amount: int64(899000)},
	}

	req1 := proto.CreateUtxoTransactionRequest{
		Chain:  ChainName,
		Symbol: Symbol,
		Vins:   vin,
		Vouts:  vout,
		Fee:    big.NewInt(0).SetInt64(1000).String(),
	}

	reply1, err := btcChainAdaptor.CreateUtxoTransaction(&req1)
	assert.Nil(t, err)
	assert.Equal(t, expectedRawDataStr, hex.EncodeToString(reply1.TxData))
	assert.Equal(t, 1, len(reply1.SignHashes))
	assert.Equal(t, expectedSignHashStr, hex.EncodeToString(reply1.SignHashes[0]))

	//
	req2 := &proto.QueryTransactionFromDataRequest{
		Chain:   ChainName,
		Symbol:  Symbol,
		RawData: reply1.TxData,
	}
	vins := []*proto.Vin{
		{Hash: "7b99b5913b6f026d4c91e540fca678cbc119e66bf89329747aa2f48ace484080", Index: uint32(0), Amount: int64(900000), Address: "n3HsmPMAEa2ovEzMNrXKZhSMGvUEfBpHcd"},
	}

	testVinOnOff(t, (*ChainAdaptor).QueryUtxoTransactionFromData, req2, vins, func(replyA interface{}, err error) {
		reply := replyA.(*proto.QueryUtxoTransactionReply)

		assert.Nil(t, err)
		assert.Equal(t, reply1.SignHashes, reply.SignHashes)
		assert.Equal(t, 1, len(reply1.SignHashes))
		assert.Equal(t, expectedSignHashStr, hex.EncodeToString(reply.SignHashes[0]))
	})

	// Sign with Prviate key
	priWif, err := btcutil.DecodeWIF("cMahea7zqjxrtgAbB7LSGbcQUr1uX1ojuat9jZoeiemEvoHeHdkv")
	assert.Nil(t, err)
	privKey := priWif.PrivKey
	pkData, sig0 := signOneVin(privKey, reply1.SignHashes[0], client.compressed)
	assert.Equal(t, expectedPkDataStr, hex.EncodeToString(pkData))
	assert.Equal(t, expectedSigStr, hex.EncodeToString(sig0))

	originSig0, err := btcec.ParseSignature(sig0, btcec.S256())
	assert.Nil(t, err)
	r, s := originSig0.R.Bytes(), originSig0.S.Bytes()
	sig := make([]byte, 65)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)
	sig[64] = 0

	// creat sigscript and verify
	r1 := new(big.Int).SetBytes(sig[0:32])
	s1 := new(big.Int).SetBytes(sig[32:64])
	assert.True(t, originSig0.R.Cmp(r1) == 0)
	assert.True(t, originSig0.S.Cmp(s1) == 0)

	req3 := proto.CreateUtxoSignedTransactionRequest{
		Chain:      ChainName,
		Symbol:     Symbol,
		TxData:     reply1.TxData, // rawData
		PublicKeys: [][]byte{pkData},
		Signatures: [][]byte{sig},
	}

	reply3, err := btcChainAdaptor.CreateUtxoSignedTransaction(&req3)
	assert.Nil(t, err)
	hash3, err := chainhash.NewHash(reply3.Hash)
	assert.Nil(t, err)
	assert.Equal(t, expectedSignedTxStr, hex.EncodeToString(reply3.SignedTxData))
	assert.Equal(t, expectedHashStr, hash3.String())

	req4 := proto.QueryTransactionFromSignedDataRequest{
		Chain:        ChainName,
		Symbol:       Symbol,
		SignedTxData: reply3.SignedTxData,
	}

	testVinOnOff(t, (*ChainAdaptor).QueryUtxoTransactionFromSignedData, &req4, vin, func(_reply interface{}, err error) {
		reply := _reply.(*proto.QueryUtxoTransactionReply)

		assert.Equal(t, hash3.String(), reply.TxHash)
	})

	req5 := proto.BroadcastTransactionRequest{
		Chain:        ChainName,
		Symbol:       Symbol,
		SignedTxData: reply3.SignedTxData,
	}

	reply5, err := btcChainAdaptor.BroadcastTransaction(&req5)
	assert.NotNil(t, err)
	assert.Equal(t, btcjson.RPCErrorCode(-27), err.(*btcjson.RPCError).Code, "unexpected error: ", reply5.Msg)
	// assert.Equal(t, expectedHashStr, reply5.TxHash)
	//
}

// used to send
func TestCreateAndSingedTransaction4(t *testing.T) {
	btcChainAdaptor := newChainAdaptorWithConfig(conf)

	client := btcChainAdaptor.client
	client.compressed = false
	// expectedRawDataStr := "010000000145138613310ded962b679f31268faaf73ca2cdf1e6e44c61d16cd21d97be71ed0000000000ffffffff02a0bb0d00000000001976a914eed9944e930bf91b0d0636be81430ec141eae51988acd07e0100000000001976a914ac80df1fa9dd5740c6f17e03b0ca6ce8d871c9a088ac00000000"
	// expectedSignHashStr := "1e589854304006c8669ca7fd9fbef23594eb33484886a4423ca8dc62908fd0be"
	// expectedPkDataStr := "0395a2c1cddab943e4eef857968e2a5cc2e587c92b598e8517830869c5f3203f3b"
	// expectedSigStr := "3045022100af824d0774c234628a0302a307b9abd2ed6bffc60d59eb6f85ba1d8dd445ea080220492ab01d0ed4dc565fa857613e33ec1663c6c46d580357aa349d6bc0fea5437301"
	// expectedSignedTxStr := "010000000145138613310ded962b679f31268faaf73ca2cdf1e6e44c61d16cd21d97be71ed000000006b483045022100af824d0774c234628a0302a307b9abd2ed6bffc60d59eb6f85ba1d8dd445ea080220492ab01d0ed4dc565fa857613e33ec1663c6c46d580357aa349d6bc0fea5437301210395a2c1cddab943e4eef857968e2a5cc2e587c92b598e8517830869c5f3203f3bffffffff02a0bb0d00000000001976a914eed9944e930bf91b0d0636be81430ec141eae51988acd07e0100000000001976a914ac80df1fa9dd5740c6f17e03b0ca6ce8d871c9a088ac00000000"
	// expectedHashStr := "7b99b5913b6f026d4c91e540fca678cbc119e66bf89329747aa2f48ace484080"
	// reverseHashStr := "804048ce8af4a27a742993f86be619c1cb78a6fc40e5914c6d026f3b91b5997b"

	vin := []*proto.Vin{
		{Hash: "68f1dd81e44fc474535a36545d4748b236062145f2580f1369f891f37eec1df8", Index: uint32(0), Amount: int64(720000), Address: "mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9"},
	}
	vout := []*proto.Vout{
		{Address: "mt9guuanGED4j23FCJKAhLQ3nRTNjPqK8J", Amount: int64(719500)},
	}

	req1 := proto.CreateUtxoTransactionRequest{
		Chain:  ChainName,
		Symbol: Symbol,
		Vins:   vin,
		Vouts:  vout,
		Fee:    big.NewInt(0).SetInt64(500).String(),
	}

	reply1, err := btcChainAdaptor.CreateUtxoTransaction(&req1)
	require.NoError(t, err)

	// Sign with Prviate key
	priWif, err := btcutil.DecodeWIF("cMqUKKvaEzKfPnooWk5fRe5c5CXGw2R1KEKefdkktTDSzSGmqfxN")
	assert.Nil(t, err)
	privKey := priWif.PrivKey
	pkData, sig0 := signOneVin(privKey, reply1.SignHashes[0], client.compressed)

	originSig0, err := btcec.ParseSignature(sig0, btcec.S256())
	assert.Nil(t, err)
	r, s := originSig0.R.Bytes(), originSig0.S.Bytes()
	sig := make([]byte, 65)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)
	sig[64] = 0

	// creat sigscript and verify
	r1 := new(big.Int).SetBytes(sig[0:32])
	s1 := new(big.Int).SetBytes(sig[32:64])
	assert.True(t, originSig0.R.Cmp(r1) == 0)
	assert.True(t, originSig0.S.Cmp(s1) == 0)

	req3 := proto.CreateUtxoSignedTransactionRequest{
		Chain:      ChainName,
		Symbol:     Symbol,
		TxData:     reply1.TxData, // rawData
		PublicKeys: [][]byte{pkData},
		Signatures: [][]byte{sig},
	}

	reply3, err := btcChainAdaptor.CreateUtxoSignedTransaction(&req3)
	assert.Nil(t, err)

	req5 := proto.BroadcastTransactionRequest{
		Chain:        ChainName,
		Symbol:       Symbol,
		SignedTxData: reply3.SignedTxData,
	}

	reply5, err := btcChainAdaptor.BroadcastTransaction(&req5)
	require.NoError(t, err)
	t.Logf("txhash:%v", reply5.TxHash)

}

func TestBroadcastTx(t *testing.T) {
	t.Skip("test with prepared env")
	btcChainAdaptor := newChainAdaptorWithConfig(conf)

	client := btcChainAdaptor.client
	client.compressed = true

	vin := []*proto.Vin{
		{Hash: "bc3ea29b93fa6f7df2ffb5ab20be9a5f92811f43be24ed59d54f658fd98151be", Index: uint32(0), Amount: int64(5000000000), Address: "mqTUxiyd43AqDKNbTBadPQLqSv6hd1s6gQ"},
	}
	vout := []*proto.Vout{
		{Address: "mqTUxiyd43AqDKNbTBadPQLqSv6hd1s6gQ", Amount: int64(5000000000 - 200)},
	}

	req1 := proto.CreateUtxoTransactionRequest{
		Chain:  ChainName,
		Symbol: Symbol,
		Vins:   vin,
		Vouts:  vout,
		Fee:    big.NewInt(0).SetInt64(200).String(),
	}

	reply1, err := btcChainAdaptor.CreateUtxoTransaction(&req1)
	require.NoError(t, err)

	// Sign with Prviate key
	priWif, err := btcutil.DecodeWIF("cRxBToQKZZE3cZw5oEAre1rfdzRNFSsKPZPCEyzWjjeqH8nm6VtZ")
	assert.Nil(t, err)
	privKey := priWif.PrivKey
	pkData, sig0 := signOneVin(privKey, reply1.SignHashes[0], client.compressed)

	originSig0, err := btcec.ParseSignature(sig0, btcec.S256())
	assert.Nil(t, err)
	r, s := originSig0.R.Bytes(), originSig0.S.Bytes()
	sig := make([]byte, 65)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)
	sig[64] = 0

	// creat sigscript and verify
	r1 := new(big.Int).SetBytes(sig[0:32])
	s1 := new(big.Int).SetBytes(sig[32:64])
	assert.True(t, originSig0.R.Cmp(r1) == 0)
	assert.True(t, originSig0.S.Cmp(s1) == 0)

	req3 := proto.CreateUtxoSignedTransactionRequest{
		Chain:      ChainName,
		Symbol:     Symbol,
		TxData:     reply1.TxData, // rawData
		PublicKeys: [][]byte{pkData},
		Signatures: [][]byte{sig},
	}

	reply3, err := btcChainAdaptor.CreateUtxoSignedTransaction(&req3)
	assert.Nil(t, err)

	req5 := proto.BroadcastTransactionRequest{
		Chain:        ChainName,
		Symbol:       Symbol,
		SignedTxData: reply3.SignedTxData,
	}

	reply5, err := btcChainAdaptor.BroadcastTransaction(&req5)
	assert.Nil(t, err)
	t.Logf("txhash:%v", reply5.TxHash)

}
