package bitcoin

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hbtc-chain/chainnode/config"
	"github.com/hbtc-chain/chainnode/proto"
)

func TestConvertAddressNoFullNode(t *testing.T) {
	btcChainAdaptorWithoutFullNode := newChainAdaptorWithClients([]*btcClient{newLocalBtcClient(config.TestNet)})

	genPub2Addr()

	var req proto.ConvertAddressRequest
	req.Chain = ChainName

	for _, a := range keyAddrComb {
		req.PublicKey = a.pubKey
		reply, err := btcChainAdaptorWithoutFullNode.ConvertAddress(&req)
		assert.Nil(t, err)
		assert.Equal(t, a.testAddr, reply.Address)
	}

	// change to mainnet params
	btcChainAdaptorWithoutFullNode = newChainAdaptorWithClients([]*btcClient{newLocalBtcClient(config.MainNet)})
	for _, a := range keyAddrComb {
		req.PublicKey = a.pubKey
		reply, err := btcChainAdaptorWithoutFullNode.ConvertAddress(&req)
		assert.Nil(t, err)
		assert.Equal(t, a.mainAddr, reply.Address)
	}
}

func TestValidAddressNoFullNode(t *testing.T) {
	btcChainAdaptorWithoutFullNode := newChainAdaptorWithClients([]*btcClient{newLocalBtcClient(config.TestNet)})

	genPub2Addr()

	var req proto.ValidAddressRequest
	req.Chain = ChainName
	req.Symbol = Symbol

	// testnet paramater
	for _, a := range keyAddrComb {
		req.Address = a.testAddr
		reply, err := btcChainAdaptorWithoutFullNode.ValidAddress(&req)
		assert.Nil(t, err)
		assert.Equal(t, true, reply.Valid)
		assert.Equal(t, true, reply.CanWithdrawal)
		assert.Equal(t, a.testAddr, reply.CanonicalAddress)

		req.Address = a.mainAddr
		_, err = btcChainAdaptorWithoutFullNode.ValidAddress(&req)
		assert.NotNil(t, err)
	}

	for _, a := range mainnetAddrs {
		req.Address = a
		reply, err := btcChainAdaptorWithoutFullNode.ValidAddress(&req)
		assert.NotNil(t, err)
		assert.Equal(t, proto.ReturnCode_ERROR, reply.Code)
	}

	for _, a := range testnetAddrs {
		req.Address = a
		reply, err := btcChainAdaptorWithoutFullNode.ValidAddress(&req)
		assert.Nil(t, err)
		assert.Equal(t, true, reply.Valid)
		assert.Equal(t, true, reply.CanWithdrawal)
		assert.Equal(t, a, reply.CanonicalAddress)
	}

	for _, a := range illegalAddrs {
		req.Address = a
		reply, err := btcChainAdaptorWithoutFullNode.ValidAddress(&req)
		assert.NotNil(t, err)
		assert.Equal(t, proto.ReturnCode_ERROR, reply.Code)
	}

	// mainnet params
	btcChainAdaptorWithoutFullNode = newChainAdaptorWithClients([]*btcClient{newLocalBtcClient(config.MainNet)})
	for _, a := range keyAddrComb {
		req.Address = a.mainAddr
		reply, err := btcChainAdaptorWithoutFullNode.ValidAddress(&req)
		assert.Nil(t, err)
		assert.Equal(t, true, reply.Valid)
		assert.Equal(t, true, reply.CanWithdrawal)

		req.Address = a.testAddr
		_, err = btcChainAdaptorWithoutFullNode.ValidAddress(&req)
		assert.NotNil(t, err)
	}

	for _, a := range mainnetAddrs {
		req.Address = a
		reply, err := btcChainAdaptorWithoutFullNode.ValidAddress(&req)
		assert.Nil(t, err)
		assert.Equal(t, true, reply.Valid)
		assert.Equal(t, true, reply.CanWithdrawal)
	}

	for _, a := range testnetAddrs {
		req.Address = a
		reply, err := btcChainAdaptorWithoutFullNode.ValidAddress(&req)
		assert.NotNil(t, err)
		assert.Equal(t, proto.ReturnCode_ERROR, reply.Code)
	}

	for _, a := range illegalAddrs {
		req.Address = a
		reply, err := btcChainAdaptorWithoutFullNode.ValidAddress(&req)
		assert.NotNil(t, err)
		assert.Equal(t, proto.ReturnCode_ERROR, reply.Code)
	}
}

func TestQueryTransactionFromSignedDataNoFullNode(t *testing.T) {
	bz, err := hex.DecodeString("02000000000101266715d8a3aab35fabf484f8e98a68396473ed59f0ec76dddc350ec8d5c9033800000000171600147e6e0170c81cf74bb9a433a2f905546a2766c98bfeffffff0210270000000000001976a914d6c331c38a8b4c966397c4862f86bcfe42cd924588ac6ccdae000000000017a914336b360ddbaf7dd99716bf1c2b92ad233ca9e2aa870247304402200d6fafc20ec2d1a52b62bf6130bbb22678e14c11dec18eea997483cd1cf9340a02201b88b3508454ae7ccf1ccef245f6eafaee6209b5f55bb2c27b17de44d20db67a012103fe6fb3175dd133e95bb8312ef0ad87017cb12f5b62b59fc2dd4b2e39690b2bf3fa2f1700")
	assert.Nil(t, err)

	var req proto.QueryTransactionFromSignedDataRequest
	req.Chain = ChainName
	req.Symbol = Symbol
	req.SignedTxData = bz

	vins := []*proto.Vin{
		{
			Hash:    "3803c9d5c80e35dcdd76ecf059ed736439688ae9f884f4ab5fb3aaa3d8156726",
			Index:   0,
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
	req.SignedTxData = bz1
	vins = []*proto.Vin{
		{
			Hash:    "c94a4debf11bfef9316681f61fb663d15ed7a6ad8698063c4f1348575aec57a9",
			Index:   0,
			Address: "mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9",
			Amount:  32500000,
		},
	}
	testVinOnOff(t, (*ChainAdaptor).QueryUtxoTransactionFromSignedData, &req, vins, func(_reply interface{}, err error) {
		reply := _reply.(*proto.QueryUtxoTransactionReply)

		assert.Equal(t, "bc703215720998316f66833dcea3056842d5d0565ae38c0d078caf060cb7b64c", reply.TxHash)
		assert.Equal(t, "mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9", reply.Vins[0].Address)
		assert.Equal(t, "c94a4debf11bfef9316681f61fb663d15ed7a6ad8698063c4f1348575aec57a9", reply.Vins[0].Hash)
		assert.Equal(t, 1, len(reply.SignHashes))
		assert.Equal(t, "faa1e29f8ff3c19e6f307e9f2ce2f2f0ded93930796dbcf84ea59431136a0b6e", hex.EncodeToString(reply.SignHashes[0]))
	})

	// err because of too many outuput transactions
	bz2, err := hex.DecodeString("02000000000101266715d8a3abbbbbbbbbbab35fabf484f8e98a68396473ed59f0ec76dddc350ec8d5c9033800000000171600147e6e0170c81cf74bb9a433a2f905546a2766c98bfeffffff0210270000000000001976a914d6c331c38a8b4c966397c4862f86bcfe42cd924588ac6ccdae000000000017a914336b360ddbaf7dd99716bf1c2b92ad233ca9e2aa870247304402200d6fafc20ec2d1a52b62bf6130bbb22678e14c11dec18eea997483cd1cf9340a02201b88b3508454ae7ccf1ccef245f6eafaee6209b5f55bb2c27b17de44d20db67a012103fe6fb3175dd133e95bb8312ef0ad87017cb12f5b62b59fc2dd4b2e39690b2bf3fa2f17001d3476")
	assert.Nil(t, err)
	req.SignedTxData = bz2
	vins = []*proto.Vin{
		{
			Hash:    "3803c9d5c80e35dcdd76ecf059ed736439688ae9f884f4ab5fb3aaa3d8156726",
			Index:   0,
			Address: "2MyXNsXWUYmhVth3Rm6DWrDnpfiia79UsPk",
			Amount:  11466020,
		},
	}
	testVinOnOff(t, (*ChainAdaptor).QueryUtxoTransactionFromSignedData, &req, vins, func(_reply interface{}, err error) {
		reply := _reply.(*proto.QueryUtxoTransactionReply)

		assert.NotNil(t, err)
		assert.Equal(t, proto.ReturnCode_ERROR, reply.Code)
		assert.Contains(t, err.Error(), "MsgTx.BtcDecode: too many output transactions")
	})
}

func TestQueryTransactionFromDataNoFullNode(t *testing.T) {
	data, err := hex.DecodeString("0100000001a957ec5a5748134f3c069886ada6d75ed163b61ff6816631f9fe1bf1eb4d4ac90000000000ffffffff020048e801000000001976a91419064bda7eb5049f922a4bca4c24808c6aea948d88ac005307000000000017a91410080578e54a2a66efcb55e69b073100d0da47b98700000000")
	expectedHash := "faa1e29f8ff3c19e6f307e9f2ce2f2f0ded93930796dbcf84ea59431136a0b6e"
	assert.Nil(t, err)

	/*
		vout := []*proto.Vout{
			&proto.Vout{Address: "mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9", Amount: int64(32000000)},
			&proto.Vout{Address: "2MthzQgsQ8Rw8vPMtTsrTdqc9HsWiDHM9VY", Amount: int64(480000)},
		}
	*/

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

func TestVerifySignedTransactionFromDifferentAddressNoFullNode(t *testing.T) {
	bz, err := hex.DecodeString("02000000000101266715d8a3aab35fabf484f8e98a68396473ed59f0ec76dddc350ec8d5c9033800000000171600147e6e0170c81cf74bb9a433a2f905546a2766c98bfeffffff0210270000000000001976a914d6c331c38a8b4c966397c4862f86bcfe42cd924588ac6ccdae000000000017a914336b360ddbaf7dd99716bf1c2b92ad233ca9e2aa870247304402200d6fafc20ec2d1a52b62bf6130bbb22678e14c11dec18eea997483cd1cf9340a02201b88b3508454ae7ccf1ccef245f6eafaee6209b5f55bb2c27b17de44d20db67a012103fe6fb3175dd133e95bb8312ef0ad87017cb12f5b62b59fc2dd4b2e39690b2bf3fa2f1700")
	assert.Nil(t, err)

	var req proto.VerifySignedTransactionRequest
	req.Chain = ChainName
	req.Symbol = Symbol
	req.SignedTxData = bz
	vins := []*proto.Vin{
		{
			Hash:    "3803c9d5c80e35dcdd76ecf059ed736439688ae9f884f4ab5fb3aaa3d8156726",
			Index:   0,
			Amount:  11466020,
			Address: "2MyXNsXWUYmhVth3Rm6DWrDnpfiia79UsPk",
		},
	}

	testVinOnOff(t, (*ChainAdaptor).VerifyUtxoSignedTransaction, &req, vins, func(_reply interface{}, err error) {
		reply := _reply.(*proto.VerifySignedTransactionReply)

		assert.Nil(t, err)
		assert.Equal(t, true, reply.Verified)
	})

	bz, err = hex.DecodeString("02000000000101266715d8a3abbbbbbbbbbab35fabf484f8e98a68396473ed59f0ec76dddc350ec8d5c9033800000000171600147e6e0170c81cf74bb9a433a2f905546a2766c98bfeffffff0210270000000000001976a914d6c331c38a8b4c966397c4862f86bcfe42cd924588ac6ccdae000000000017a914336b360ddbaf7dd99716bf1c2b92ad233ca9e2aa870247304402200d6fafc20ec2d1a52b62bf6130bbb22678e14c11dec18eea997483cd1cf9340a02201b88b3508454ae7ccf1ccef245f6eafaee6209b5f55bb2c27b17de44d20db67a012103fe6fb3175dd133e95bb8312ef0ad87017cb12f5b62b59fc2dd4b2e39690b2bf3fa2f17001d3476")
	assert.Nil(t, err)
	req.SignedTxData = bz
	req.Addresses = []string{"2MyXNsXWUYmhVth3Rm6DWrDnpfiia79UsPk"}

	testVinOnOff(t, (*ChainAdaptor).VerifyUtxoSignedTransaction, &req, vins, func(_reply interface{}, err error) {
		reply := _reply.(*proto.VerifySignedTransactionReply)

		assert.NotNil(t, err)
		assert.Equal(t, proto.ReturnCode_ERROR, reply.Code)
	})

	bz1, err := hex.DecodeString("0100000001a957ec5a5748134f3c069886ada6d75ed163b61ff6816631f9fe1bf1eb4d4ac9000000008a4730440220486972701a1f11d72c575e0fec145c957c21a89df58a2c5878a4f62253eedaa1022065e13ca5d689c8b1c86bbcc5d30f05340948cc12c5e17c55cc434fca6f495ba10141043cd360fecac46da64c411c6b471d8e147504ed74c2cafd9a29329c63c4eaf1603fb5a230c1ba28d93bb6834989869259d4a4156d33fd5f99075e4b968cdbe8b8ffffffff020048e801000000001976a91419064bda7eb5049f922a4bca4c24808c6aea948d88ac005307000000000017a91410080578e54a2a66efcb55e69b073100d0da47b98700000000")
	assert.Nil(t, err)
	req.SignedTxData = bz1
	vins = []*proto.Vin{
		{
			Hash:    "c94a4debf11bfef9316681f61fb663d15ed7a6ad8698063c4f1348575aec57a9",
			Index:   0,
			Amount:  32500000,
			Address: "mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9",
		},
	}
	req.Addresses = []string{"mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9"}
	testVinOnOff(t, (*ChainAdaptor).VerifyUtxoSignedTransaction, &req, vins, func(_reply interface{}, err error) {
		reply := _reply.(*proto.VerifySignedTransactionReply)

		assert.Nil(t, err)
		assert.Equal(t, true, reply.Verified)
	})
}

func TestCreateTransactionAmountMismatchNoFullNode(t *testing.T) {
	btcChainAdaptorWithoutFullNode := newChainAdaptorWithClients([]*btcClient{newLocalBtcClient(config.TestNet)})

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

	reply, err := btcChainAdaptorWithoutFullNode.CreateUtxoTransaction(&req)
	assert.NotNil(t, err)
	assert.Equal(t, "CreateTransaction, total amount in != total amount out + fee", reply.Msg)
}
