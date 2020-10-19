package tron

import (
	"github.com/hbtc-chain/chainnode/config"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGetLatestBlock(t *testing.T) {
	client := tronChainAdaptor.(*ChainAdaptor).client

	block, err := client.grpcClient.GetNowBlock()
	require.Nil(t, err)
	t.Logf("block:%+v", block)

}

func TestGetLatestBlock2(t *testing.T) {
	conf, err := config.New("./testnet.yaml")
	require.Nil(t, err)

	t.Logf("conf:%v", conf)

	client, err := newTronClient(conf)
	require.Nil(t, err)

	block, err := client.grpcClient.GetBlockByNum(1)

	t.Logf("block:%v", block)
}

func TestGetBalance(t *testing.T) {
	conf, err := config.New("./testnet.yaml")
	require.Nil(t, err)

	t.Logf("conf:%v", conf)

	client, err := newTronClient(conf)
	require.Nil(t, err)

	acc, err := client.grpcClient.GetAccount("TYbcQrwHHjcd3n4pKGkxmCnjtw3nPoBs8b")
	require.Nil(t, err)
	require.NotNil(t, acc)
	//t.Logf("acc:%v", acc)
}
