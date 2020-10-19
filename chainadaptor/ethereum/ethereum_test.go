package ethereum

import (
	"context"
	"encoding/hex"
	"errors"
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/prometheus/common/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/hbtc-chain/chainnode/chainadaptor"
	"github.com/hbtc-chain/chainnode/config"
	"github.com/hbtc-chain/chainnode/proto"
)

var ethChainAdaptor chainadaptor.ChainAdaptor
var ethChainAdaptorWithoutFullNode chainadaptor.ChainAdaptor

func TestMain(m *testing.M) {
	conf, err := config.New("testnet.yaml")
	if err != nil {
		panic(err)
	}

	ethChainAdaptor, err = NewChainAdaptor(conf)
	if err != nil {
		panic(err)
	}
	ethChainAdaptorWithoutFullNode = NewLocalChainAdaptor(config.TestNet)
	os.Exit(m.Run())
}

func TestValidAddress(t *testing.T) {
	testdata := []struct {
		address       string
		isValid       bool
		canWithdrawal bool
		stdAddress    string
	}{
		{"0xA906Cb666198e528dB4E85E1908380C7325de115", true, true, "0xA906Cb666198e528dB4E85E1908380C7325de115"},
		{"0xACD6733fBC09FB95A2FF9e53C26e9B5D036D6EEb", true, true, "0xACD6733fBC09FB95A2FF9e53C26e9B5D036D6EEb"},
		{"0xACD6733fBC09FB95A2FF9e53C26e9B5D036D6EEB", true, true, "0xACD6733fBC09FB95A2FF9e53C26e9B5D036D6EEb"},
		{"0xacd6733fBC09FB95A2FF9e53C26e9B5D036D6EEb", true, true, "0xACD6733fBC09FB95A2FF9e53C26e9B5D036D6EEb"},
		{"0xc96d141c9110a8e61ed62caad8a7c858db15b82c", true, true, "0xc96d141c9110a8E61eD62caaD8A7c858dB15B82c"},
		{"0xacd6733fbc09fb95a2ff9e53c26e9b5d036d6eeb", true, true, "0xACD6733fBC09FB95A2FF9e53C26e9B5D036D6EEb"},
		{"0xc96d141c9110a8e61ed62caad8a7c858db15b82c", true, true, "0xc96d141c9110a8E61eD62caaD8A7c858dB15B82c"},
		{"0xacd6733fbc09fb95a2ff9e53c26e9b5d036d6ee", false, false, "0xACD6733fBC09FB95A2FF9e53C26e9B5D036D6EEb"}, // length is illegal
		{"0xd139e358ae9cb5424b2067da96f94cc938343446", true, true, "0xd139E358aE9cB5424B2067da96F94cC938343446"},
		{"0x81b7e08f65bdf5648606c89998a9cc8164397647", true, true, "0x81b7E08F65Bdf5648606c89998A9CC8164397647"},
		{"0xacd6733fbc09fb95a2ff9e53", false, false, "0xACD6733fBC09FB95A2FF9e53C26e9B5D036D6EEb"}, // length is illegal
	}

	for _, data := range testdata {
		req := &proto.ValidAddressRequest{
			Chain:   ChainName,
			Symbol:  Symbol,
			Address: data.address,
		}

		res, err := ethChainAdaptor.ValidAddress(req)
		assert.Nil(t, err)
		assert.Equal(t, data.isValid, res.Valid)
		if res.Valid {
			assert.Equal(t, data.stdAddress, res.CanonicalAddress)
			assert.Equal(t, data.canWithdrawal, res.CanWithdrawal)
		}
	}
}

func TestQueryTransaction(t *testing.T) {
	client := ethChainAdaptor.(*ChainAdaptor).client

	// correct hash
	txHash := common.HexToHash("0x33ac277a7e48a77fc6762660c5fa1372ca3395b9a59370ebbb0e7419ba4441bc")
	tx, pending, err := client.TransactionByHash(context.TODO(), txHash)
	assert.Nil(t, err)
	assert.False(t, pending)
	assert.Equal(t, uint64(24), tx.Nonce())
	assert.Equal(t, txHash, tx.Hash())

	// incorrect hash, case1
	txHash = common.HexToHash("0x33ac277a7e48a77fc6762660c5fa1372ca3395b9a59370ebbb0e7419ba4441b")
	tx, pending, err = client.TransactionByHash(context.TODO(), txHash)
	assert.NotNil(t, err)
	assert.False(t, pending)
	assert.Nil(t, tx)

	// incorrect hash, case2
	txHash = common.HexToHash("0x33")
	tx, pending, err = client.TransactionByHash(context.TODO(), txHash)
	assert.NotNil(t, err)
	assert.False(t, pending)
	assert.Nil(t, tx)
}

func TestQueryTransaction2(t *testing.T) {
	// correct hash
	txHash := "0x33ac277a7e48a77fc6762660c5fa1372ca3395b9a59370ebbb0e7419ba4441bc"
	req := &proto.QueryTransactionRequest{
		Chain:  ChainName,
		Symbol: Symbol,
		TxHash: txHash,
	}

	res, err := ethChainAdaptor.QueryAccountTransaction(req)
	assert.Nil(t, err)
	assert.Equal(t, proto.ReturnCode(0), res.Code)
	assert.Equal(t, "", res.Msg)
	assert.Equal(t, txHash, res.TxHash)
	assert.Equal(t, proto.TxStatus(3), res.TxStatus)
	assert.Equal(t, "0xc96d141c9110a8e61ed62caad8a7c858db15b82c", strings.ToLower(res.From))
	assert.Equal(t, "0x26fdc6b993d56e3f2a1b7f0fb2997750307bbae6", strings.ToLower(res.To))
	assert.Equal(t, "1000000000000000", res.Amount)
	assert.Equal(t, "", res.Memo)
	assert.Equal(t, uint64(24), res.Nonce)
	assert.Equal(t, "21000", res.GasLimit)
	assert.Equal(t, "16000000000", res.GasPrice)    // 0.000000016
	assert.Equal(t, "336000000000000", res.CostFee) // 0.000336eth
	assert.Equal(t, uint64(6453681), res.BlockHeight)
	assert.Equal(t, "d28e8836943d474034fe9e0c85fec05b21095e7c163ccacc8767c81ab9538b60", hex.EncodeToString(res.SignHash))
}

func TestQueryTransaction3(t *testing.T) {
	// correct hash
	txHash := "0xb32d7b4d93e0519594eba85ee02759ac7b57d43fb9dc4ec3218858f263e618bd"
	req := &proto.QueryTransactionRequest{
		Chain:  ChainName,
		Symbol: Symbol,
		TxHash: txHash,
	}

	res, err := ethChainAdaptor.QueryAccountTransaction(req)
	assert.Nil(t, err)
	assert.Equal(t, proto.ReturnCode(0), res.Code)
	assert.Equal(t, "", res.Msg)
	assert.Equal(t, txHash, res.TxHash)
	assert.Equal(t, proto.TxStatus(3), res.TxStatus)
	t.Logf("res:%v", res)
}

func TestQueryTransactionMock(t *testing.T) {
	testConfirmations := uint64(5)
	mockClient := &MockEthClient{}

	txHashErrorNotFoundAfterTimeout := common.HexToHash("0x12")
	req := &proto.QueryTransactionRequest{
		Chain:  ChainName,
		Symbol: Symbol,
		TxHash: "0x12",
	}
	signedTxHex := "f86f84019e1774843b9aca00825208949576e27257e0eceea565fce04ab1beedfc6f35e4880de0b6b3a7640000801ba0ebc2e281446bfd6c17b860156d977f32a4dd899382a54ae01aec2befff3f6948a07ffe30f29afb294344ffd2dbb5213fb54e94e159491efad324f8e2eb2bdb5c0e"
	signedTxBytes, err := hex.DecodeString(signedTxHex)
	assert.NoError(t, err)
	signedTx := &types.Transaction{}
	err = rlp.DecodeBytes(signedTxBytes, signedTx)
	assert.NoError(t, err)
	txBlockNumber := big.NewInt(4660)
	txBlockHash := common.HexToHash("0xaaaa")
	latestBlockNumber, _ := big.NewInt(0).SetString("0x1234", 0)
	latestBlockNumber = latestBlockNumber.Add(latestBlockNumber,
		big.NewInt(int64(testConfirmations)))

	// Times() does not need to be exact match
	mockClient.On("TransactionReceipt", mock.Anything,
		txHashErrorNotFoundAfterTimeout).Return(
		&types.Receipt{
			Status:      types.ReceiptStatusSuccessful,
			BlockNumber: txBlockNumber,
			BlockHash:   txBlockHash,
		}, nil)
	mockClient.On("TransactionByHash", mock.Anything,
		txHashErrorNotFoundAfterTimeout).Once().Return(signedTx, false, nil)

	mockClient.On("BlockByNumber", mock.Anything, (*big.Int)(nil)).Once().Return(
		types.NewBlockWithHeader(
			&types.Header{
				Number: latestBlockNumber,
			},
		), nil)
	mockClient.On("TransactionReceipt", mock.Anything,
		txHashErrorNotFoundAfterTimeout).Return(
		&types.Receipt{
			Status: types.ReceiptStatusSuccessful,
		}, nil)

	mockAdaptor := newChainAdaptor(newMockEthClient(mockClient))
	rep, err := mockAdaptor.QueryAccountTransaction(req)
	assert.NoError(t, err)
	assert.Equal(t, "0xb9feb6c136b3a76ce08e6da8c95bc25d9057b306a61a7db389f7d7ef843cbfd0", rep.TxHash)
	assert.Equal(t, "0x81b7E08F65Bdf5648606c89998A9CC8164397647", rep.From)
	assert.Equal(t, "0x9576e27257e0eceEA565fce04ab1bEedFC6F35E4", rep.To)
	assert.Equal(t, uint64(27137908), rep.Nonce)
}

func TestQueryTransactionNotFound(t *testing.T) {
	txHash := "0xb32d7b4d93e0519594eba85ee02759ac7b57d43fb9dc4ec3218858f263e618b0"
	req := &proto.QueryTransactionRequest{
		Chain:  ChainName,
		Symbol: Symbol,
		TxHash: txHash,
	}

	res, err := ethChainAdaptor.QueryAccountTransaction(req)
	assert.Nil(t, err)
	assert.Equal(t, proto.ReturnCode(0), res.Code)
	assert.Equal(t, proto.TxStatus_NotFound, res.TxStatus)
}

func TestQueryTransactionPending(t *testing.T) {
	testConfirmations := uint64(5)
	mockClient := &MockEthClient{}
	mockEthClient := newMockEthClient(mockClient)
	mockAdaptor := newChainAdaptor(mockEthClient)

	symbol := Symbol

	txHashPending := common.HexToHash("0x01")
	req := &proto.QueryTransactionRequest{
		Chain:  ChainName,
		Symbol: symbol,
		TxHash: "0x01",
	}
	mockClient.On("TransactionByHash", mock.Anything,
		txHashPending).Return(
		&types.Transaction{}, true, nil)
	rep, err := mockAdaptor.QueryAccountTransaction(req)
	assert.NoError(t, err)
	assert.Equal(t, proto.ReturnCode_SUCCESS, rep.Code)
	assert.Equal(t, proto.TxStatus_Pending, rep.TxStatus)

	txHashPendingWithoutBlockNumber := common.HexToHash("0x02")
	req = &proto.QueryTransactionRequest{
		Chain:  ChainName,
		Symbol: symbol,
		TxHash: "0x02",
	}
	mockClient.On("TransactionReceipt", mock.Anything,
		txHashPendingWithoutBlockNumber).Return(
		&types.Receipt{
			Status:      types.ReceiptStatusSuccessful,
			BlockNumber: nil,
		}, nil)
	mockClient.On("TransactionByHash", mock.Anything,
		txHashPendingWithoutBlockNumber).Return(&types.Transaction{}, false, nil)
	rep, err = mockAdaptor.QueryAccountTransaction(req)
	assert.NoError(t, err)
	assert.Equal(t, proto.ReturnCode_SUCCESS, rep.Code)
	assert.Equal(t, proto.TxStatus_Pending, rep.TxStatus)

	txHashPendingWithInvalidLatestBlockNumber := common.HexToHash("0x04")
	txBlockNumber := big.NewInt(4660)
	req = &proto.QueryTransactionRequest{
		Chain:  ChainName,
		Symbol: symbol,
		TxHash: "0x04",
	}
	mockClient.On("BlockByNumber", mock.Anything, (*big.Int)(nil)).Once().Return(
		(*types.Block)(nil), errors.New("error"))
	mockClient.On("TransactionReceipt", mock.Anything,
		txHashPendingWithInvalidLatestBlockNumber).Return(
		&types.Receipt{
			Status:      types.ReceiptStatusSuccessful,
			BlockNumber: txBlockNumber,
		}, nil)
	mockClient.On("TransactionByHash", mock.Anything,
		txHashPendingWithInvalidLatestBlockNumber).Return(&types.Transaction{}, false, nil)
	rep, err = mockAdaptor.QueryAccountTransaction(req)
	assert.NoError(t, err)
	assert.Equal(t, proto.ReturnCode_ERROR, rep.Code)

	for i := int64(-5); i < int64(testConfirmations); i++ {
		txHashPendingWithUnconfirmedTx := common.HexToHash("0x05")
		req = &proto.QueryTransactionRequest{
			Chain:  ChainName,
			Symbol: symbol,
			TxHash: "0x05",
		}
		latestBlockNumber, _ := big.NewInt(0).SetString("0x1234", 0)
		latestBlockNumber = latestBlockNumber.Add(latestBlockNumber, big.NewInt(i))
		mockEthClient.cacheTime = 0 // Always invalidate block number cache
		mockClient.On("BlockByNumber", mock.Anything, (*big.Int)(nil)).Once().Return(
			types.NewBlockWithHeader(
				&types.Header{
					Number: latestBlockNumber,
				},
			), nil)
		mockClient.On("TransactionReceipt", mock.Anything,
			txHashPendingWithUnconfirmedTx).Return(
			&types.Receipt{
				Status:      types.ReceiptStatusSuccessful,
				BlockNumber: txBlockNumber,
			}, nil)
		mockClient.On("TransactionByHash", mock.Anything,
			txHashPendingWithUnconfirmedTx).Once().Return(
			&types.Transaction{}, false, nil)
		rep, err = mockAdaptor.QueryAccountTransaction(req)
		assert.NoError(t, err)
		assert.Equal(t, proto.ReturnCode_SUCCESS, rep.Code)
		assert.Equal(t, proto.TxStatus_Pending, rep.TxStatus)
	}

	signedTxHex := "f86f84019e1774843b9aca00825208949576e27257e0eceea565fce04ab1beedfc6f35e4880de0b6b3a7640000801ba0ebc2e281446bfd6c17b860156d977f32a4dd899382a54ae01aec2befff3f6948a07ffe30f29afb294344ffd2dbb5213fb54e94e159491efad324f8e2eb2bdb5c0e"
	signedTxBytes, err := hex.DecodeString(signedTxHex)
	assert.NoError(t, err)
	signedTx := &types.Transaction{}
	err = rlp.DecodeBytes(signedTxBytes, signedTx)
	assert.NoError(t, err)

	txHashPendingWithConfirmedTx := signedTx.Hash()
	req = &proto.QueryTransactionRequest{
		Chain:  ChainName,
		Symbol: symbol,
		TxHash: signedTx.Hash().Hex(),
	}
	latestBlockNumber, _ := big.NewInt(0).SetString("0x1234", 0)
	latestBlockNumber = latestBlockNumber.Add(latestBlockNumber,
		big.NewInt(int64(testConfirmations)))
	txBlockHash := common.HexToHash("0xaaaa")
	mockEthClient.cacheTime = 0 // Always invalidate block number cache
	mockClient.On("BlockByNumber", mock.Anything, (*big.Int)(nil)).Once().Return(
		types.NewBlockWithHeader(
			&types.Header{
				Number: latestBlockNumber,
			},
		), nil)
	mockClient.On("TransactionReceipt", mock.Anything,
		txHashPendingWithConfirmedTx).Return(
		&types.Receipt{
			Status:      types.ReceiptStatusSuccessful,
			BlockNumber: txBlockNumber,
			BlockHash:   txBlockHash,
		}, nil)
	mockClient.On("TransactionByHash", mock.Anything,
		txHashPendingWithConfirmedTx).Return(signedTx, false, nil)
	rep, err = ethChainAdaptor.QueryAccountTransaction(req)
	assert.NoError(t, err)
	assert.Equal(t, proto.ReturnCode_SUCCESS, rep.Code)
	assert.Equal(t, proto.TxStatus_Success, rep.TxStatus)
}

func TestQueryBalance(t *testing.T) {
	t.Skip("can'nt access to archive state")
	req := &proto.QueryBalanceRequest{
		Chain:       ChainName,
		Symbol:      Symbol,
		Address:     "0x02e48c5ae584f718f77a2165855994b254685cc1",
		BlockHeight: 6593704,
	}

	res, err := ethChainAdaptor.QueryBalance(req)
	assert.Nil(t, err)
	assert.Equal(t, "2000000000000000", res.Balance)

	req.BlockHeight = 6419356
	res, err = ethChainAdaptor.QueryBalance(req)
	assert.Nil(t, err)
	assert.Equal(t, "1000000000000000", res.Balance)

}

func TestQueryTransactionFromSignedData(t *testing.T) {
	data, err := hex.DecodeString("f86b808504e3b2920082520894add42af7dd58b27e1e6ca5c4fdc01214b52d382f870bdccd84e7b000801ba0b86360f1c2d2b38421a80e71bf4cf54371bc9aa62f81c925484c6557b44b13f1a07b5690150c10a3947225fb612162c90ccfaefde99f7d363a8013e3eead0e55dd")
	expectedTxHash := "0xa88cca4dd97e028d7199028888156c4dad9936a2cbdfe8262fb12a252e16d4f1"
	assert.Nil(t, err)
	req := &proto.QueryTransactionFromSignedDataRequest{
		Chain:        ChainName,
		Symbol:       Symbol,
		SignedTxData: data,
	}
	res, err := ethChainAdaptor.QueryAccountTransactionFromSignedData(req)
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

// 0xf8690683cb3da282520894c96d141c9110a8e61ed62caad8a7c858db15b82c872386f26fc100008029a060d7493646d54ea95c4201bd309cd5132d21edf9991399bedf334ac2476b7fd4a071b5495e6f4a27ef58abd318bc7b6c2f65abf4af7c609d2d8e3c95f3da9fc06e

func TestQueryTransactionFromSignedData4(t *testing.T) {
	data, err := hex.DecodeString("f8690683cb3da282520894c96d141c9110a8e61ed62caad8a7c858db15b82c872386f26fc100008029a060d7493646d54ea95c4201bd309cd5132d21edf9991399bedf334ac2476b7fd4a071b5495e6f4a27ef58abd318bc7b6c2f65abf4af7c609d2d8e3c95f3da9fc06e")
	assert.Nil(t, err)
	req := &proto.QueryTransactionFromSignedDataRequest{
		Chain:        ChainName,
		Symbol:       Symbol,
		SignedTxData: data,
	}
	res, err := ethChainAdaptor.QueryAccountTransactionFromSignedData(req)
	require.NoError(t, err)
	t.Logf("res:%v\n", res)
}
func TestQueryTransactionFromSignedData2(t *testing.T) {
	data, err := hex.DecodeString("f86a58843b9aca0082520894c96d141c9110a8e61ed62caad8a7c858db15b82c870110d9316ec0008029a0c7186fdf667ed65f4f0198dee58c49707917ce708111ddd568dfdabaed2c2fb7a0364475aa61ef1bab275c0fcd93265000485d8e0b70500abe1acd3d2e146d4b4d")
	expectedSignHash := "3b443eafae95b0e81ff3880fab7e41233c71a82cd07384db2d22a162e7935ff4"
	expectedTxHash := "0x84ed75bfad4b6d1c405a123990a1750974aa1f053394d442dfbc76090eeed44a"

	assert.Nil(t, err)
	req := &proto.QueryTransactionFromSignedDataRequest{
		Chain:        ChainName,
		Symbol:       Symbol,
		SignedTxData: data,
	}
	res, err := ethChainAdaptor.QueryAccountTransactionFromSignedData(req)
	assert.Nil(t, err)
	assert.Equal(t, "0xd139E358aE9cB5424B2067da96F94cC938343446", res.From)
	assert.Equal(t, "0xc96d141c9110a8E61eD62caaD8A7c858dB15B82c", res.To)
	assert.Equal(t, uint64(0), res.BlockHeight)
	assert.Equal(t, "0", res.CostFee)
	assert.Equal(t, "300000000000000", res.Amount)
	assert.Equal(t, "", res.Memo)
	assert.Equal(t, "21000", res.GasLimit)
	assert.Equal(t, "1000000000", res.GasPrice)
	assert.Equal(t, proto.TxStatus_Success, res.TxStatus)
	assert.Equal(t, uint64(0), res.BlockTime)
	assert.Equal(t, proto.ReturnCode_SUCCESS, res.Code)
	assert.Equal(t, "", res.Msg)
	assert.Equal(t, expectedTxHash, res.TxHash)
	assert.Equal(t, uint64(88), res.Nonce)
	assert.Equal(t, expectedSignHash, hex.EncodeToString(res.SignHash))
	assert.Equal(t, "", res.ContractAddress)
}

func TestQueryTransactionFromSignedData3(t *testing.T) {
	data, err := hex.DecodeString("f86959843b9aca0082520894c96d141c9110a8e61ed62caad8a7c858db15b82c86886c98b760008029a06672c856f120b4d526bcce103df2ebbf707c96da1b22d4c110e80c4d8ba35868a051ce420a54662be2ece89e57f30797f6aedb06b02e6cfc0e3e2f12ce48c303dc")
	expectedSignHash := "0ed98835ad4e49e83e5e7a84added408bfdc2169fce48a8268ad2faaf86906a4"
	expectedTxHash := "0xd9e17b7907043970b2258c70e4f76c792fbca28e65fa1bd5a04dde2380164fa5"

	assert.Nil(t, err)
	req := &proto.QueryTransactionFromSignedDataRequest{
		Chain:        ChainName,
		Symbol:       Symbol,
		SignedTxData: data,
	}
	res, err := ethChainAdaptor.QueryAccountTransactionFromSignedData(req)
	assert.Nil(t, err)
	assert.Equal(t, "0xd139E358aE9cB5424B2067da96F94cC938343446", res.From)
	assert.Equal(t, "0xc96d141c9110a8E61eD62caaD8A7c858dB15B82c", res.To)
	assert.Equal(t, uint64(0), res.BlockHeight)
	assert.Equal(t, "0", res.CostFee)
	assert.Equal(t, "150000000000000", res.Amount)
	assert.Equal(t, "", res.Memo)
	assert.Equal(t, "21000", res.GasLimit)
	assert.Equal(t, "1000000000", res.GasPrice)
	assert.Equal(t, proto.TxStatus_Success, res.TxStatus)
	assert.Equal(t, uint64(0), res.BlockTime)
	assert.Equal(t, proto.ReturnCode_SUCCESS, res.Code)
	assert.Equal(t, "", res.Msg)
	assert.Equal(t, expectedTxHash, res.TxHash)
	assert.Equal(t, uint64(89), res.Nonce)
	assert.Equal(t, expectedSignHash, hex.EncodeToString(res.SignHash))
	assert.Equal(t, "", res.ContractAddress)
}

func TestQueryTransactionFromData(t *testing.T) {
	data, err := hex.DecodeString("ea59843b9aca0082520894c96d141c9110a8e61ed62caad8a7c858db15b82c870110d9316ec00080808080")
	assert.Nil(t, err)

	req := &proto.QueryTransactionFromDataRequest{
		Chain:   ChainName,
		Symbol:  Symbol,
		RawData: data,
	}

	reply, err := ethChainAdaptor.QueryAccountTransactionFromData(req)
	assert.Nil(t, err)
	assert.Equal(t, "", reply.From)
	assert.Equal(t, "0xc96d141c9110a8E61eD62caaD8A7c858dB15B82c", reply.To)
	assert.Equal(t, "300000000000000", reply.Amount)
	assert.Equal(t, "1000000000", reply.GasPrice)
	assert.Equal(t, "21000", reply.GasLimit)
	assert.Equal(t, "15a5fc99dd43dd89cd3632ecece65a88cf6226f97806decb397d930afafcb9e7", hex.EncodeToString(reply.SignHash))
	assert.Equal(t, "", reply.ContractAddress)
}

func TestCreateTransaction(t *testing.T) {
	expectedData := "ea59843b9aca0082520894c96d141c9110a8e61ed62caad8a7c858db15b82c870110d9316ec00080808080"
	expectedHash := "15a5fc99dd43dd89cd3632ecece65a88cf6226f97806decb397d930afafcb9e7"
	// res1, err := QueryGasPrice(context.TODO(), &proto.QueryGasPriceRequest{
	//	ChainName: chain,
	// })
	// assert.Nil(t, err)
	// assert.NotNil(t, res1.GasPrice)
	// t.Logf("gas Price:%v", res1.GasPrice)

	// res2, err := QueryNonce(context.TODO(), &proto.QueryNonceRequest{
	//	ChainName:   chain,
	//	Address: "0xd139E358aE9cB5424B2067da96F94cC938343446",
	// })
	//
	// assert.Less(t, uint64(87), res2.Nonce)

	req := &proto.CreateAccountTransactionRequest{
		Chain:    ChainName,
		Symbol:   Symbol,
		From:     "0xd139E358aE9cB5424B2067da96F94cC938343446",
		To:       "0xc96d141c9110a8E61eD62caaD8A7c858dB15B82c",
		Amount:   "300000000000000",
		Nonce:    89,
		GasPrice: "1000000000", // 1Gwei = 10^9Wei
		GasLimit: "21000",
	}

	res3, err := ethChainAdaptor.CreateAccountTransaction(req)
	assert.Nil(t, err)
	assert.Equal(t, expectedData, hex.EncodeToString(res3.TxData))
	assert.Equal(t, expectedHash, hex.EncodeToString(res3.SignHash))
}

func TestQueryGasPrice(t *testing.T) {
	res1, err := ethChainAdaptor.QueryGasPrice(&proto.QueryGasPriceRequest{
		Chain: ChainName,
	})
	assert.Nil(t, err)
	assert.NotNil(t, res1.GasPrice)
}

func TestQueryNonce(t *testing.T) {
	res2, err := ethChainAdaptor.QueryNonce(&proto.QueryNonceRequest{
		Chain:   ChainName,
		Address: "0xd139E358aE9cB5424B2067da96F94cC938343446",
	})
	assert.Nil(t, err)
	assert.Less(t, uint64(87), res2.Nonce)
}

func TestCreateSignedTransaction(t *testing.T) {
	privKey := "106cbc245ee55b36810fcf54a13c30aa4b79df18f5df0095fb9a2a4ab4ba5e42"
	expectedData := "ea58843b9aca0082520894c96d141c9110a8e61ed62caad8a7c858db15b82c870110d9316ec00080808080"
	expectedSignHash := "3b443eafae95b0e81ff3880fab7e41233c71a82cd07384db2d22a162e7935ff4"
	expectedTxHash, _ := hex.DecodeString("84ed75bfad4b6d1c405a123990a1750974aa1f053394d442dfbc76090eeed44a")
	expectedSignedData, _ := hex.DecodeString("f86a58843b9aca0082520894c96d141c9110a8e61ed62caad8a7c858db15b82c870110d9316ec0008029a0c7186fdf667ed65f4f0198dee58c49707917ce708111ddd568dfdabaed2c2fb7a0364475aa61ef1bab275c0fcd93265000485d8e0b70500abe1acd3d2e146d4b4d")

	req1 := &proto.CreateAccountTransactionRequest{
		Chain:    ChainName,
		Symbol:   Symbol,
		From:     "0xd139E358aE9cB5424B2067da96F94cC938343446",
		To:       "0xc96d141c9110a8E61eD62caaD8A7c858dB15B82c",
		Amount:   "300000000000000",
		Nonce:    88,
		GasPrice: "1000000000",
		GasLimit: "21000",
	}

	res1, err := ethChainAdaptor.CreateAccountTransaction(req1)
	assert.Nil(t, err)
	assert.Equal(t, expectedData, hex.EncodeToString(res1.TxData))
	assert.Equal(t, expectedSignHash, hex.EncodeToString(res1.SignHash))

	sig, pub, err := sign(privKey, res1.SignHash)
	require.NoError(t, err)

	req2 := &proto.CreateAccountSignedTransactionRequest{
		Chain:     ChainName,
		Symbol:    Symbol,
		TxData:    res1.TxData,
		Signature: sig,
		PublicKey: pub,
	}

	res2, err := ethChainAdaptor.CreateAccountSignedTransaction(req2)
	assert.Nil(t, err)
	assert.Equal(t, expectedTxHash, res2.Hash)
	assert.Equal(t, expectedSignedData, res2.SignedTxData)
}

func TestBroadcastTransaction(t *testing.T) {
	data, err := hex.DecodeString("f86a58843b9aca0082520894c96d141c9110a8e61ed62caad8a7c858db15b82c870110d9316ec0008029a0c7186fdf667ed65f4f0198dee58c49707917ce708111ddd568dfdabaed2c2fb7a0364475aa61ef1bab275c0fcd93265000485d8e0b70500abe1acd3d2e146d4b4d")
	assert.Nil(t, err)

	req := &proto.BroadcastTransactionRequest{
		Chain:        ChainName,
		Symbol:       Symbol,
		SignedTxData: data,
	}

	res, err := ethChainAdaptor.BroadcastTransaction(req)
	// assert.Nil(t, err)
	// assert.Equal(t, proto.ReturnCode_SUCCESS, res.Code)
	assert.NotNil(t, err)
	assert.Equal(t, proto.ReturnCode_ERROR, res.Code)
	assert.Equal(t, "nonce too low", res.Msg)
}

func TestCreateSignedTransaction2(t *testing.T) {
	privKey := "106cbc245ee55b36810fcf54a13c30aa4b79df18f5df0095fb9a2a4ab4ba5e42"
	expectedData := "e959843b9aca0082520894c96d141c9110a8e61ed62caad8a7c858db15b82c86886c98b7600080808080"
	expectedSignHash := "0ed98835ad4e49e83e5e7a84added408bfdc2169fce48a8268ad2faaf86906a4"
	expectedTxHash := "d9e17b7907043970b2258c70e4f76c792fbca28e65fa1bd5a04dde2380164fa5"
	expectedSignedData := "f86959843b9aca0082520894c96d141c9110a8e61ed62caad8a7c858db15b82c86886c98b760008029a06672c856f120b4d526bcce103df2ebbf707c96da1b22d4c110e80c4d8ba35868a051ce420a54662be2ece89e57f30797f6aedb06b02e6cfc0e3e2f12ce48c303dc"

	req1 := &proto.CreateAccountTransactionRequest{
		Chain:    ChainName,
		Symbol:   Symbol,
		From:     "0xd139E358aE9cB5424B2067da96F94cC938343446",
		To:       "0xc96d141c9110a8E61eD62caaD8A7c858dB15B82c",
		Amount:   "150000000000000",
		Nonce:    89,
		GasPrice: "1000000000",
		GasLimit: "21000",
	}

	res1, err := ethChainAdaptor.CreateAccountTransaction(req1)
	assert.Nil(t, err)
	assert.Equal(t, expectedData, hex.EncodeToString(res1.TxData))
	assert.Equal(t, expectedSignHash, hex.EncodeToString(res1.SignHash))

	sig, pub, err := sign(privKey, res1.SignHash)
	require.NoError(t, err)

	req2 := &proto.CreateAccountSignedTransactionRequest{
		Chain:     ChainName,
		Symbol:    Symbol,
		TxData:    res1.TxData,
		Signature: sig,
		PublicKey: pub,
	}

	res2, err := ethChainAdaptor.CreateAccountSignedTransaction(req2)
	assert.Nil(t, err)
	// t.Logf("hash:%v", hex.EncodeToString(res2.Hash))
	// t.Logf("signedData:%v", hex.EncodeToString(res2.SignedTxData))
	assert.Equal(t, expectedTxHash, hex.EncodeToString(res2.Hash))
	assert.Equal(t, expectedSignedData, hex.EncodeToString(res2.SignedTxData))
}

func TestBroadcastTransaction2(t *testing.T) {
	data, err := hex.DecodeString("f86959843b9aca0082520894c96d141c9110a8e61ed62caad8a7c858db15b82c86886c98b760008029a06672c856f120b4d526bcce103df2ebbf707c96da1b22d4c110e80c4d8ba35868a051ce420a54662be2ece89e57f30797f6aedb06b02e6cfc0e3e2f12ce48c303dc")
	assert.Nil(t, err)

	req := &proto.BroadcastTransactionRequest{
		Chain:        ChainName,
		Symbol:       Symbol,
		SignedTxData: data,
	}

	res, err := ethChainAdaptor.BroadcastTransaction(req)
	// assert.Nil(t, err)
	// assert.Equal(t, proto.ReturnCode_SUCCESS, res.Code)
	assert.NotNil(t, err)
	assert.Equal(t, proto.ReturnCode_ERROR, res.Code)
	assert.Equal(t, "nonce too low", res.Msg)
}

func TestBroadcastTransaction4(t *testing.T) {
	data, err := hex.DecodeString("f86a03843b9aca00825208943fc3aaa0b7e3cc21265ce94aca413fb9a06c8b1c8702d79883d20000802aa0ffbd3170116e93da55f642966199becf818eb5d12a6a565664ff7b4bdb5531eca02da27f1c9cbeab2dbefe2ce9ba2433949f8927eba9bac07ebc3aa4c1fd13212a")
	assert.Nil(t, err)

	req1 := &proto.BroadcastTransactionRequest{
		Chain:        ChainName,
		Symbol:       Symbol,
		SignedTxData: data,
	}

	res1, err := ethChainAdaptor.BroadcastTransaction(req1)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "known transaction")
	t.Logf("res:%v", res1.Msg)

}

func sign(privKey string, hash []byte) ([]byte, []byte, error) {
	// convert private string to ecdsa privatekey
	prv, err := crypto.HexToECDSA(privKey)
	if err != nil {
		log.Errorf("Parse private key err:%v", err)
		return nil, nil, err
	}

	h := common.BytesToHash(hash)
	sig, err := crypto.Sign(h[:], prv)
	if err != nil {
		return nil, nil, err
	}

	pub := crypto.FromECDSAPub(&prv.PublicKey)

	return sig, pub, err
}

func TestVerifySignedTransaction(t *testing.T) {
	data, err := hex.DecodeString("f86a58843b9aca0082520894c96d141c9110a8e61ed62caad8a7c858db15b82c870110d9316ec0008029a0c7186fdf667ed65f4f0198dee58c49707917ce708111ddd568dfdabaed2c2fb7a0364475aa61ef1bab275c0fcd93265000485d8e0b70500abe1acd3d2e146d4b4d")
	assert.Nil(t, err)

	req := &proto.VerifySignedTransactionRequest{
		Chain:        ChainName,
		Symbol:       Symbol,
		Addresses:    []string{"0xd139E358aE9cB5424B2067da96F94cC938343446"},
		SignedTxData: data,
	}

	res, err := ethChainAdaptor.VerifyAccountSignedTransaction(req)
	assert.Nil(t, err)
	assert.Equal(t, true, res.Verified)

	req.Addresses = []string{"0xc96d141c9110a8E61eD62caaD8A7c858dB15B82c"}
	res, err = ethChainAdaptor.VerifyAccountSignedTransaction(req)
	assert.Nil(t, err)
	assert.Equal(t, false, res.Verified)

}
