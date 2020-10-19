package tron

import (
	"math/big"
	"net"
	"strings"
	"sync"

	"github.com/hbtc-chain/chainnode/config"
	tclient "github.com/hbtc-chain/gotron-sdk/pkg/client"
	"github.com/ethereum/go-ethereum/log"
)

var (
	blockNumberCacheTime int64 = 10 // seconds
)

const (
	ChainIDMain = 0x41
	//ChainIDTest = 0xa0
	ChainIDTest = 0x41
)

type tronClient struct {
	grpcClient       *tclient.GrpcClient
	chainID          byte
	cacheBlockNumber *big.Int
	cacheTime        int64
	rw               sync.RWMutex
	confirmations    uint64
	local            bool
}

// newTronClient init the tron client
func newTronClient(conf *config.Config) (*tronClient, error) {
	var client tronClient
	log.Info("tron client setup", "network", conf.NetWork)

	var err error
	var rpcURL string

	client.confirmations = conf.Fullnode.Trx.Confirmations

	domain := strings.TrimPrefix(conf.Fullnode.Trx.RPCURL, "http://")
	domain = strings.TrimPrefix(domain, "https://")
	if strings.Contains(domain, ":") {
		words := strings.Split(domain, ":")

		var ipAddr *net.IPAddr
		ipAddr, err = net.ResolveIPAddr("ip", words[0])
		if err != nil {
			log.Error("resolve eth domain failed", "url", conf.Fullnode.Trx.RPCURL)
			return nil, err
		}
		log.Info("tronclient setup client", "ip", ipAddr)

		rpcURL = strings.Replace(conf.Fullnode.Trx.RPCURL, words[0], ipAddr.String(), 1)
	} else {
		rpcURL = conf.Fullnode.Trx.RPCURL
	}
	c := tclient.NewGrpcClient(rpcURL)
	if err := c.Start(); err != nil {
		return nil, err
	}

	client.chainID = ChainIDTest
	if conf.NetWork == "mainnet" {
		client.chainID = ChainIDMain
	}
	client.grpcClient = c

	return &client, nil
}

func (t *tronClient) Close() {
	t.grpcClient.Stop()
}

func newLocalTronClient(network config.NetWorkType) *tronClient {
	var chainID byte
	switch network {
	case config.MainNet:
		chainID = ChainIDMain
	case config.TestNet:
		chainID = ChainIDTest
	case config.RegTest:
		chainID = ChainIDTest
	}

	return &tronClient{
		grpcClient:       &tclient.GrpcClient{},
		cacheBlockNumber: nil,
		chainID:          chainID,
		local:            true,
	}
}
