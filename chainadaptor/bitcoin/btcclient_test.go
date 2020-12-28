package bitcoin

import (
	"testing"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/stretchr/testify/assert"
)

const (
	hash = "c2247fb66cf44652f27552b052a7d359d48a1c8e90a50651f6104a441041963f"
)

func TestGetTxHash(t *testing.T) {
	btcChainAdaptor := newChainAdaptorWithConfig(conf)
	txHash, err := chainhash.NewHashFromStr(hash)
	assert.Nil(t, err)

	tx, err := btcChainAdaptor.getClient().GetRawTransactionVerbose(txHash)
	assert.Nil(t, err)
	assert.Equal(t, hash, tx.Txid)
}
