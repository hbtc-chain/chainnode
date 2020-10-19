package ethereum

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetTxHash(t *testing.T) {
	client := ethChainAdaptor.(*ChainAdaptor).client
	txHash := common.HexToHash("0x33ac277a7e48a77fc6762660c5fa1372ca3395b9a59370ebbb0e7419ba4441bc")
	tx, pending, err := client.TransactionByHash(context.TODO(), txHash)
	assert.Nil(t, err)
	assert.False(t, pending)
	assert.Equal(t, uint64(24), tx.Nonce())
	assert.Equal(t, txHash, tx.Hash())
}

func TestGetBlockByNumber(t *testing.T) {
	client := ethChainAdaptor.(*ChainAdaptor).client

	num := big.NewInt(6552057)
	expectedBlockHash := common.HexToHash("0x30442f7e23d048f2e84e9b001a6210e03e7be55a13efe6713f46dfaee138c282")
	expectedParentHash := common.HexToHash("0x3300aabac1203db6b5b60ed7b06bc0928fff1a30166163c28ee4ea2aadde2722")

	block, err := client.BlockByNumber(context.TODO(), num)
	assert.NoError(t, err)
	assert.Equal(t, expectedBlockHash, block.Hash())
	assert.Equal(t, expectedParentHash, block.ParentHash())
	assert.Equal(t, int64(6552057), block.Number().Int64())
}

func newMockEthClient(client *MockEthClient) *ethClient {
	return &ethClient{
		Client:           client,
		chainConfig:      params.RopstenChainConfig,
		cacheBlockNumber: nil,
		confirmations:    5,
	}
}

type MockEthClient struct {
	mock.Mock
}

func (c *MockEthClient) BalanceAt(context.Context, common.Address, *big.Int) (
	*big.Int, error) {
	panic("Impelement me")
}

func (c *MockEthClient) TransactionByHash(ctx context.Context, hash common.Hash) (
	*types.Transaction, bool, error) {
	m := c.Called(ctx, hash)
	return m.Get(0).(*types.Transaction), m.Bool(1), m.Error(2)
}

func (c *MockEthClient) BlockByNumber(ctx context.Context, blockNumber *big.Int) (
	*types.Block, error) {
	m := c.Called(ctx, blockNumber)
	return m.Get(0).(*types.Block), m.Error(1)
}

func (c *MockEthClient) TransactionReceipt(ctx context.Context, hash common.Hash) (
	*types.Receipt, error) {
	m := c.Called(ctx, hash)
	return m.Get(0).(*types.Receipt), m.Error(1)
}

func (c *MockEthClient) NonceAt(context.Context, common.Address, *big.Int) (
	uint64, error) {
	panic("Impelement me")
}

func (c *MockEthClient) SendTransaction(context.Context, *types.Transaction) error {
	panic("Impelement me")
}

func (c *MockEthClient) CodeAt(context.Context, common.Address, *big.Int) (
	[]byte, error) {
	panic("Impelement me")
}

func (c *MockEthClient) SuggestGasPrice(context.Context) (*big.Int, error) {
	panic("Impelement me")
}

func (c *MockEthClient) CallContract(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error) {
	panic("implement me")
}

func (c *MockEthClient) PendingCodeAt(context.Context, common.Address) ([]byte, error) {
	panic("implement me")
}

func (c *MockEthClient) PendingNonceAt(context.Context, common.Address) (uint64, error) {
	panic("implement me")
}

func (c *MockEthClient) EstimateGas(context.Context, ethereum.CallMsg) (gas uint64, err error) {
	panic("implement me")
}

func (c *MockEthClient) FilterLogs(context.Context, ethereum.FilterQuery) ([]types.Log, error) {
	panic("implement me")
}

func (c *MockEthClient) SubscribeFilterLogs(context.Context, ethereum.FilterQuery, chan<- types.Log) (ethereum.Subscription, error) {
	panic("implement me")
}
