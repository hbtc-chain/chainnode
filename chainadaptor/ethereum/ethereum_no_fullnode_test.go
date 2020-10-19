package ethereum

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hbtc-chain/chainnode/proto"
)

func TestValidAddressNoFullNode(t *testing.T) {
	testdata := []struct {
		address    string
		isValid    bool
		stdAddress string
	}{
		{"0xACD6733fBC09FB95A2FF9e53C26e9B5D036D6EEb", true, "0xACD6733fBC09FB95A2FF9e53C26e9B5D036D6EEb"},
		{"0xACD6733fBC09FB95A2FF9e53C26e9B5D036D6EEB", true, "0xACD6733fBC09FB95A2FF9e53C26e9B5D036D6EEb"},
		{"0xacd6733fBC09FB95A2FF9e53C26e9B5D036D6EEb", true, "0xACD6733fBC09FB95A2FF9e53C26e9B5D036D6EEb"},
		{"0xc96d141c9110a8e61ed62caad8a7c858db15b82c", true, "0xc96d141c9110a8E61eD62caaD8A7c858dB15B82c"},
		{"0xacd6733fbc09fb95a2ff9e53c26e9b5d036d6eeb", true, "0xACD6733fBC09FB95A2FF9e53C26e9B5D036D6EEb"},
		{"0xc96d141c9110a8e61ed62caad8a7c858db15b82c", true, "0xc96d141c9110a8E61eD62caaD8A7c858dB15B82c"},
		{"0xacd6733fbc09fb95a2ff9e53c26e9b5d036d6ee", false, "0xACD6733fBC09FB95A2FF9e53C26e9B5D036D6EEb"}, // length is illegal
		{"0xd139e358ae9cb5424b2067da96f94cc938343446", true, "0xd139E358aE9cB5424B2067da96F94cC938343446"},
		{"0x81b7e08f65bdf5648606c89998a9cc8164397647", true, "0x81b7E08F65Bdf5648606c89998A9CC8164397647"},
		{"0xacd6733fbc09fb95a2ff9e53", false, "0xACD6733fBC09FB95A2FF9e53C26e9B5D036D6EEb"}, // length is illegal
	}

	for _, data := range testdata {
		req := &proto.ValidAddressRequest{
			Chain:   ChainName,
			Symbol:  Symbol,
			Address: data.address,
		}

		res, err := ethChainAdaptorWithoutFullNode.ValidAddress(req)
		assert.Nil(t, err)
		assert.Equal(t, data.isValid, res.Valid)
		if res.Valid {
			assert.Equal(t, data.stdAddress, res.CanonicalAddress)
		}
	}
}

func TestQueryTransactionFromSignedDataNoFullNode(t *testing.T) {
	data, err := hex.DecodeString("f86b808504e3b2920082520894add42af7dd58b27e1e6ca5c4fdc01214b52d382f870bdccd84e7b000801ba0b86360f1c2d2b38421a80e71bf4cf54371bc9aa62f81c925484c6557b44b13f1a07b5690150c10a3947225fb612162c90ccfaefde99f7d363a8013e3eead0e55dd")
	expectedTxHash := "0xa88cca4dd97e028d7199028888156c4dad9936a2cbdfe8262fb12a252e16d4f1"
	assert.Nil(t, err)
	req := &proto.QueryTransactionFromSignedDataRequest{
		Chain:        ChainName,
		Symbol:       Symbol,
		SignedTxData: data,
	}
	res, err := ethChainAdaptorWithoutFullNode.QueryAccountTransactionFromSignedData(req)
	assert.Nil(t, err)
	assert.Equal(t, "0x7EA7eb1c8B0Fba77964C561f9B7494A87534Aa15", res.From)
	assert.Equal(t, "0xadd42AF7DD58B27e1E6cA5C4FdC01214b52d382f", res.To)
	assert.Equal(t, uint64(0), res.BlockHeight)
	assert.Equal(t, "0", res.CostFee)
	assert.Equal(t, "3339000000000000", res.Amount)
	assert.Equal(t, "", res.Memo)
	assert.Equal(t, "21000", res.GasLimit)
	assert.Equal(t, "21000000000", res.GasPrice)
	assert.Equal(t, proto.TxStatus_Success, res.TxStatus)
	assert.Equal(t, uint64(0), res.BlockTime)
	assert.Equal(t, proto.ReturnCode_SUCCESS, res.Code)
	assert.Equal(t, "", res.Msg)
	assert.Equal(t, expectedTxHash, res.TxHash)
	assert.Equal(t, uint64(0), res.Nonce)
	assert.Equal(t, "0f8c62511de16b0a2db7612e43ffa3ac1bc82c6066c982fca658b8c1e09fc109", hex.EncodeToString(res.SignHash))
	assert.Equal(t, "", res.ContractAddress)
}

func TestQueryTransactionFromDataNoFullNode(t *testing.T) {
	data, err := hex.DecodeString("ea59843b9aca0082520894c96d141c9110a8e61ed62caad8a7c858db15b82c870110d9316ec00080808080")
	assert.Nil(t, err)

	req := &proto.QueryTransactionFromDataRequest{
		Chain:   ChainName,
		Symbol:  Symbol,
		RawData: data,
	}

	reply, err := ethChainAdaptorWithoutFullNode.QueryAccountTransactionFromData(req)
	assert.Nil(t, err)
	assert.Equal(t, "", reply.From)
	assert.Equal(t, "0xc96d141c9110a8E61eD62caaD8A7c858dB15B82c", reply.To)
	assert.Equal(t, "300000000000000", reply.Amount)
	assert.Equal(t, "1000000000", reply.GasPrice)
	assert.Equal(t, "21000", reply.GasLimit)
	assert.Equal(t, "15a5fc99dd43dd89cd3632ecece65a88cf6226f97806decb397d930afafcb9e7", hex.EncodeToString(reply.SignHash))
	assert.Equal(t, "", reply.ContractAddress)
}

func TestVerifySignedTransactionNoFullNode(t *testing.T) {
	data, err := hex.DecodeString("f86a58843b9aca0082520894c96d141c9110a8e61ed62caad8a7c858db15b82c870110d9316ec0008029a0c7186fdf667ed65f4f0198dee58c49707917ce708111ddd568dfdabaed2c2fb7a0364475aa61ef1bab275c0fcd93265000485d8e0b70500abe1acd3d2e146d4b4d")
	assert.Nil(t, err)

	req := &proto.VerifySignedTransactionRequest{
		Chain:        ChainName,
		Symbol:       Symbol,
		Addresses:    []string{"0xd139E358aE9cB5424B2067da96F94cC938343446"},
		SignedTxData: data,
	}

	res, err := ethChainAdaptorWithoutFullNode.VerifyAccountSignedTransaction(req)
	assert.Nil(t, err)
	assert.Equal(t, true, res.Verified)

	req.Addresses = []string{"0xc96d141c9110a8E61eD62caaD8A7c858dB15B82c"}
	res, err = ethChainAdaptorWithoutFullNode.VerifyAccountSignedTransaction(req)
	assert.Nil(t, err)
	assert.Equal(t, false, res.Verified)

}
