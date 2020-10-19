package ethereum

import (
	"context"
	"math/big"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/hbtc-chain/chainnode/config"
)

var (
	blockNumberCacheTime int64 = 10 // seconds
)

type ethClient struct {
	Client
	chainConfig      *params.ChainConfig
	cacheBlockNumber *big.Int
	cacheTime        int64
	rw               sync.RWMutex
	confirmations    uint64
	local            bool
}

type Client interface {
	bind.ContractBackend
	BalanceAt(context.Context, common.Address, *big.Int) (*big.Int, error)
	TransactionByHash(context.Context, common.Hash) (*types.Transaction, bool, error)
	BlockByNumber(context.Context, *big.Int) (*types.Block, error)
	TransactionReceipt(context.Context, common.Hash) (*types.Receipt, error)
	NonceAt(context.Context, common.Address, *big.Int) (uint64, error)
}

// newEthClient init the eth client
func newEthClient(conf *config.Config) (*ethClient, error) {
	var client ethClient
	client.chainConfig = params.RopstenChainConfig
	if conf.NetWork == "mainnet" {
		client.chainConfig = params.MainnetChainConfig
	} else if conf.NetWork == "regtest" {
		client.chainConfig = params.AllCliqueProtocolChanges
	}
	log.Info("eth client setup", "chain_id", client.chainConfig.ChainID.Int64(), "network", conf.NetWork)

	var err error
	var rpcURL string

	client.confirmations = conf.Fullnode.Eth.Confirmations

	domain := strings.TrimPrefix(conf.Fullnode.Eth.RPCURL, "http://")
	domain = strings.TrimPrefix(domain, "https://")
	if strings.Contains(domain, ":") {
		words := strings.Split(domain, ":")

		var ipAddr *net.IPAddr
		ipAddr, err = net.ResolveIPAddr("ip", words[0])
		if err != nil {
			log.Error("resolve eth domain failed", "url", conf.Fullnode.Eth.RPCURL)
			return nil, err
		}
		log.Info("ethclient setup client", "ip", ipAddr)

		rpcURL = strings.Replace(conf.Fullnode.Eth.RPCURL, words[0], ipAddr.String(), 1)
	} else {
		rpcURL = conf.Fullnode.Eth.RPCURL
	}

	client.Client, err = ethclient.Dial(rpcURL)
	if err != nil {
		log.Error("ethclient dial failed", "err", err)
		return nil, err
	}

	return &client, nil
}

func newLocalEthClient(network config.NetWorkType) *ethClient {
	var para *params.ChainConfig
	switch network {
	case config.MainNet:
		para = params.MainnetChainConfig
	case config.TestNet:
		para = params.RopstenChainConfig
	case config.RegTest:
		para = params.AllCliqueProtocolChanges
	default:
		panic("unsupported network type")
	}
	return &ethClient{
		Client:           &ethclient.Client{},
		chainConfig:      para,
		cacheBlockNumber: nil,
		local:            true,
	}
}

func (client *ethClient) blockNumber() *big.Int {
	now := time.Now().Unix()
	client.rw.RLock()
	if now-client.cacheTime < blockNumberCacheTime {
		number := client.cacheBlockNumber
		client.rw.RUnlock()
		return number
	}
	client.rw.RUnlock()

	client.rw.Lock()
	defer client.rw.Unlock()
	if now-client.cacheTime < blockNumberCacheTime {
		return client.cacheBlockNumber
	}
	latestBlock, err := client.BlockByNumber(context.Background(), nil)
	if err != nil {
		log.Error("get BlockByNumber failed", "error", err)
		return nil
	}
	client.cacheBlockNumber = latestBlock.Number()
	client.cacheTime = now
	return client.cacheBlockNumber
}

func (client *ethClient) isContractAddress(address common.Address) bool {
	code, err := client.CodeAt(context.Background(), address, nil)
	return err == nil && len(code) > 0
}
