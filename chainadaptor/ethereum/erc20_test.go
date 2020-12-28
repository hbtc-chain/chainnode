package ethereum

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hbtc-chain/chainnode/proto"
)

const (
	tbtcContractAddress = "0x50802B3E32748a75696e7124B25D6113468958b1"
	tbtcSymbol          = "tbtc"
)

func TestBalanceOf(t *testing.T) {
	t.Skip("can'nt access to archive state")

	client := ethChainAdaptor.(*ChainAdaptor).getClient()

	address := "0x00Cb32D3C9c0040E117158AaBBa7ACEE6f7Be307"
	balance, err := client.erc20BalanceOf(tbtcContractAddress, address, big.NewInt(6981577))
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), balance.Uint64())
	balance, err = client.erc20BalanceOf(tbtcContractAddress, address, big.NewInt(6981578))
	assert.NoError(t, err)
	expected, _ := big.NewFloat(10e8).Int(big.NewInt(0))
	assert.Equal(t, 0, balance.Cmp(expected))
	balance, err = client.erc20BalanceOf(tbtcContractAddress, address, big.NewInt(6981583))
	assert.NoError(t, err)
	expected, _ = big.NewFloat(25e8).Int(big.NewInt(0))
	assert.Equal(t, 0, balance.Cmp(expected))
}

func TestDecimal(t *testing.T) {
	client := ethChainAdaptor.(*ChainAdaptor).getClient()
	decimals, err := client.erc20Decimals(tbtcContractAddress)
	assert.NoError(t, err)
	assert.Equal(t, uint8(8), decimals)
}

func TestQueryBalanceForERC20(t *testing.T) {
	t.Skip("can'nt access to archive state")

	req := &proto.QueryBalanceRequest{
		Chain:           ChainName,
		Symbol:          tbtcSymbol,
		Address:         "0x00Cb32D3C9c0040E117158AaBBa7ACEE6f7Be307",
		ContractAddress: tbtcContractAddress,
		BlockHeight:     6981577,
	}

	res, err := ethChainAdaptor.QueryBalance(req)
	assert.Nil(t, err)
	assert.Equal(t, "0", res.Balance)

	req.BlockHeight = 6981578
	res, err = ethChainAdaptor.QueryBalance(req)
	assert.Nil(t, err)
	assert.Equal(t, "1000000000", res.Balance)

	req.BlockHeight = 6981583
	res, err = ethChainAdaptor.QueryBalance(req)
	assert.Nil(t, err)
	assert.Equal(t, "2500000000", res.Balance)
}

func TestCreateERC20Transaction(t *testing.T) {
	expectedData := "f86959843b9aca00830186a09450802b3e32748a75696e7124b25d6113468958b180b844a9059cbb000000000000000000000000c96d141c9110a8e61ed62caad8a7c858db15b82c00000000000000000000000000000000000000000000000000000006fc23ac00808080"
	expectedHash := "74c5e2d36c3c66905fa64be1824db6fe0b61329381dd34dfbd43a6cb8e4e3189"

	req := &proto.CreateAccountTransactionRequest{
		Chain:           ChainName,
		Symbol:          Symbol,
		From:            "0xd139E358aE9cB5424B2067da96F94cC938343446",
		To:              "0xc96d141c9110a8E61eD62caaD8A7c858dB15B82c",
		Amount:          "30000000000",
		Nonce:           89,
		GasPrice:        "1000000000", // 1Gwei = 10^9Wei
		GasLimit:        "100000",     // Greater than normal send
		ContractAddress: tbtcContractAddress,
	}

	res3, err := ethChainAdaptor.CreateAccountTransaction(req)
	assert.Nil(t, err)
	assert.Equal(t, expectedData, hex.EncodeToString(res3.TxData))
	assert.Equal(t, expectedHash, hex.EncodeToString(res3.SignHash))
}

func TestQueryERC20TransactionFromData(t *testing.T) {
	txData, _ := hex.DecodeString("f86959843b9aca00830186a09450802b3e32748a75696e7124b25d6113468958b180b844a9059cbb000000000000000000000000c96d141c9110a8e61ed62caad8a7c858db15b82c00000000000000000000000000000000000000000000000000000006fc23ac00808080")

	req := &proto.QueryTransactionFromDataRequest{
		Chain:   ChainName,
		Symbol:  tbtcSymbol,
		RawData: txData,
	}

	res, err := ethChainAdaptor.QueryAccountTransactionFromData(req)

	assert.Nil(t, err)
	assert.Equal(t, "", res.From)
	assert.Equal(t, "0xc96d141c9110a8E61eD62caaD8A7c858dB15B82c", res.To)
	assert.Equal(t, uint64(89), res.Nonce)
	assert.Equal(t, uint64(0), res.BlockHeight)
	assert.Equal(t, "", res.CostFee)
	assert.Equal(t, "30000000000", res.Amount)
	assert.Equal(t, "", res.Memo)
	assert.Equal(t, "100000", res.GasLimit)
	assert.Equal(t, "1000000000", res.GasPrice)
	assert.Equal(t, proto.TxStatus_NotFound, res.TxStatus)
	assert.Equal(t, uint64(0), res.BlockTime)
	assert.Equal(t, proto.ReturnCode_SUCCESS, res.Code)
	assert.Equal(t, "", res.Msg)
	assert.Equal(t, "", res.TxHash)
	assert.Equal(t, tbtcContractAddress, res.ContractAddress)
}

func TestQueryERC20TransactionFromSignedData(t *testing.T) {
	signedTxData, _ := hex.DecodeString("f8a901843b9aca00830186a09450802b3e32748a75696e7124b25d6113468958b180b844a9059cbb0000000000000000000000007801e6fc30c77852f82272c17f853bff20a70c4200000000000000000000000000000000000000000000000000000000000027102aa0796fc79bc923e3bcab68b79e237be2a2ed270e511821fc8ae48b385ea75de0a8a02b3acc092454bb1f7c1ddabd56b4c8693226b3fa551b1daecec81448e77e59bb")

	req := &proto.QueryTransactionFromSignedDataRequest{
		Chain:        ChainName,
		Symbol:       tbtcSymbol,
		SignedTxData: signedTxData,
	}

	res, err := ethChainAdaptor.QueryAccountTransactionFromSignedData(req)

	assert.Nil(t, err)
	assert.Equal(t, "0x68c6d35f7b63cAc3814f521F43c121daa59E5233", res.From)
	assert.Equal(t, "0x7801E6fC30c77852F82272c17F853bFf20A70c42", res.To)
	assert.Equal(t, uint64(1), res.Nonce)
	assert.Equal(t, uint64(0), res.BlockHeight)
	assert.Equal(t, "0", res.CostFee)
	assert.Equal(t, "10000", res.Amount)
	assert.Equal(t, "", res.Memo)
	assert.Equal(t, "100000", res.GasLimit)
	assert.Equal(t, "1000000000", res.GasPrice)
	assert.Equal(t, proto.TxStatus_Success, res.TxStatus)
	assert.Equal(t, uint64(0), res.BlockTime)
	assert.Equal(t, proto.ReturnCode_SUCCESS, res.Code)
	assert.Equal(t, "", res.Msg)
	assert.Equal(t, "0x1509e8fec49276ae38f503213e5f49eaef904ad042e35c3edbed4e45ecb18f43", res.TxHash)
	assert.Equal(t, tbtcContractAddress, res.ContractAddress)
}

func TestQueryERC20Transaction(t *testing.T) {
	hash := "0x1509e8fec49276ae38f503213e5f49eaef904ad042e35c3edbed4e45ecb18f43"
	req := &proto.QueryTransactionRequest{
		Chain:  ChainName,
		Symbol: tbtcSymbol,
		TxHash: hash,
	}

	res, err := ethChainAdaptor.QueryAccountTransaction(req)

	assert.Nil(t, err)
	assert.Equal(t, "0x68c6d35f7b63cAc3814f521F43c121daa59E5233", res.From)
	assert.Equal(t, "0x7801E6fC30c77852F82272c17F853bFf20A70c42", res.To)
	assert.Equal(t, uint64(1), res.Nonce)
	assert.Equal(t, uint64(6995599), res.BlockHeight)
	assert.Equal(t,
		// 36280 * 10^9 wei
		new(big.Int).Mul(big.NewInt(36280),
			new(big.Int).Exp(big.NewInt(10), big.NewInt(9), nil)).String(),
		res.CostFee)
	assert.Equal(t, "10000", res.Amount)
	assert.Equal(t, "", res.Memo)
	assert.Equal(t, "100000", res.GasLimit)
	assert.Equal(t, "1000000000", res.GasPrice)
	assert.Equal(t, proto.TxStatus_Success, res.TxStatus)
	assert.Equal(t, uint64(0), res.BlockTime)
	assert.Equal(t, proto.ReturnCode_SUCCESS, res.Code)
	assert.Equal(t, "", res.Msg)
	assert.Equal(t, hash, res.TxHash)
	assert.Equal(t, tbtcContractAddress, res.ContractAddress)
}
