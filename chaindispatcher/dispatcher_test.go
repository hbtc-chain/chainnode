package chaindispatcher

import (
	"context"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hbtc-chain/chainnode/chainadaptor/bitcoin"
	"github.com/hbtc-chain/chainnode/chainadaptor/ethereum"
	"github.com/hbtc-chain/chainnode/chainadaptor/tron"
	"github.com/hbtc-chain/chainnode/config"
	"github.com/hbtc-chain/chainnode/proto"

	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

var client proto.ChainnodeClient

const bufSize = 1024 * 1024

var listener *bufconn.Listener

func TestMain(m *testing.M) {
	setupServer()
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		panic("Failed to dial bufnet")
	}
	defer func() {
		err2 := conn.Close()
		fmt.Printf("error while close con %v\n", err2)
	}()
	client = proto.NewChainnodeClient(conn)

	os.Exit(m.Run())
}

func setupServer() {
	conf, err := config.New("./testnet.yaml")
	if err != nil {
		panic(err)
	}

	listener = bufconn.Listen(bufSize)

	dispatcher, err := New(conf)
	if err != nil {
		log.Error("Setup dispatcher failed", "err", err)
		panic(err)
	}
	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(dispatcher.Interceptor))

	proto.RegisterChainnodeServer(grpcServer, dispatcher)

	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			log.Error("grpc ChainDispatcher serve failed", "err", err)
			panic(err)
		}
	}()
}

func bufDialer(context.Context, string) (net.Conn, error) {
	return listener.Dial()
}

func TestSupportAsset(t *testing.T) {
	var req proto.SupportChainRequest

	req.Chain = bitcoin.ChainName
	reply, err := client.SupportChain(context.TODO(), &req)
	assert.Nil(t, err)
	assert.Equal(t, true, reply.Support)

	req.Chain = ethereum.ChainName
	reply, err = client.SupportChain(context.TODO(), &req)
	assert.Nil(t, err)
	assert.Equal(t, true, reply.Support)

	req.Chain = tron.ChainName
	reply, err = client.SupportChain(context.TODO(), &req)
	assert.Nil(t, err)
	assert.Equal(t, true, reply.Support)

	req.Chain = "btc"
	reply, err = client.SupportChain(context.TODO(), &req)
	assert.Nil(t, err)
	assert.Equal(t, true, reply.Support)

	req.Chain = "eth"
	reply, err = client.SupportChain(context.TODO(), &req)
	assert.Nil(t, err)
	assert.Equal(t, true, reply.Support)

	req.Chain = "trx"
	reply, err = client.SupportChain(context.TODO(), &req)
	assert.Nil(t, err)
	assert.Equal(t, true, reply.Support)

	req.Chain = "bhbtc"
	reply, err = client.SupportChain(context.TODO(), &req)
	assert.Nil(t, err)
	assert.Equal(t, false, reply.Support)

	req.Chain = "bheth"
	reply, err = client.SupportChain(context.TODO(), &req)
	assert.Nil(t, err)
	assert.Equal(t, false, reply.Support)

}

func clone(req *proto.QueryUtxoRequest) *proto.QueryUtxoRequest {
	newReq := new(proto.QueryUtxoRequest)
	*newReq = *req
	newReq.Vin = new(proto.Vin)
	*newReq.Vin = *req.Vin
	return newReq
}

func TestQueryUtxo(t *testing.T) {
	// todo use mock to prevent fail when utxo is spent
	normalReq := proto.QueryUtxoRequest{
		Chain: bitcoin.ChainName,
		Vin: &proto.Vin{
			Hash:    "9ae3c919d84f4b72802de6f4f4aa0d88abcc9fd57315ddf27b8e25f032e4a180",
			Index:   1,
			Amount:  85475551,
			Address: "mhoGjKn5xegDXL6u5LFSUQdm5ozdM6xao9",
		},
	}

	t.Run("normal", func(t *testing.T) {
		req := normalReq
		reply, err := client.QueryUtxo(context.TODO(), &req)
		assert.NoError(t, err)
		assert.Equal(t, proto.ReturnCode_SUCCESS, reply.Code)
		assert.True(t, reply.Unspent)
	})

	t.Run("wrong vin hash", func(t *testing.T) {
		req := clone(&normalReq)
		req.Vin.Hash = "a0495dfdd82069ec3b24bcb9d1cf420e85b931fba32e8d567c0cc8c3ee4ce1ab"
		var _, err = client.QueryUtxo(context.TODO(), req)
		assert.Error(t, err)
	})

	t.Run("wrong index", func(t *testing.T) {
		req := clone(&normalReq)
		req.Vin.Index = 0
		_, err := client.QueryUtxo(context.TODO(), req)
		assert.Error(t, err)
	})

	t.Run("wrong amount", func(t *testing.T) {
		req := clone(&normalReq)
		req.Vin.Amount = 7087994
		_, err := client.QueryUtxo(context.TODO(), req)
		assert.Error(t, err)
	})

	t.Run("wrong address", func(t *testing.T) {
		req := clone(&normalReq)
		req.Vin.Address = "n17KByno4FbTenvfQkp3rMjaSSfTWUKUKb"
		_, err := client.QueryUtxo(context.TODO(), req)
		assert.Error(t, err)
	})
}

func TestVerifyAccountSignedTx(t *testing.T) {
	data, err := hex.DecodeString("f86c01850ba43b74008275309446feb3a2309789e1cc3b2fe434a42efaa9c845ab88016345785d8a0000801ba0a47277caebb9024e7021b183a6c8b463f107b7d1fa0aaffaeba4407e083f812aa0051e640773b781111b55f4b1ddb79ef02ecbe8dbbb5c23d2629ae8152a33daeb")
	req := &proto.VerifySignedTransactionRequest{
		Chain:        "eth",
		SignedTxData: data,
		Sender:       "0xd15f9b493ae32238a4d96a2766905bfd1d20d54a",
		Height:       1140000,
	}

	t.Run("normal frontier tx", func(t *testing.T) {
		_, err = client.VerifyAccountSignedTransaction(context.TODO(), req)
		assert.NoError(t, err)
	})
	data, err = hex.DecodeString("f86f8401cc5431843b9aca008252089441458aa770ab50f79387379ebec329dd70075167880de0b6b3a7640000801ba025d3293b1e22ff436fe6441831bb12b4103b2024b1d8b142893707e1370bd52fa0422a4cc6fda13a47bc7f9ecb5ca0e85ef3114b6841c8764281aec5e344567ba0")
	req = &proto.VerifySignedTransactionRequest{
		Chain:        "eth",
		SignedTxData: data,
		Sender:       "0x81b7e08f65bdf5648606c89998a9cc8164397647",
	}

	t.Run("normal tx", func(t *testing.T) {
		_, err = client.VerifyAccountSignedTransaction(context.TODO(), req)
		assert.NoError(t, err)
	})

	req.Sender = "0x81b7e08f65bdf5648606c89998a9cc8164397648"

	t.Run("sender differs", func(t *testing.T) {
		reply, err := client.VerifyAccountSignedTransaction(context.TODO(), req)
		assert.NoError(t, err)
		assert.Equal(t, false, reply.Verified)
	})

}
func TestVerifyUtxoSignedTx(t *testing.T) {
	data, err := hex.DecodeString("02000000000102524bcade1687c8c063cab88e740602e067eedbdce18dbd0d2d5145c54eb94c490100000017160014ccf09dd85dc58cc24d9dccd3e84d1e7f1f689b58feffffffa5bf04be9017989d4e22d00c0ce5f7de90db7b34e9cc2c06c4e3dcbb8a050896000000001716001471808fc591655b1b6c554257e6903f82cff3c76afeffffff0212b506000000000017a914b260ba7e0c231b3fc18b9e173406c701dd73167b87097c0f000000000017a9142ce6bdfa9e82b591387f73d3d76c89f0daa58719870247304402203badae513e7f6fde74e6f4c2318cf41e140d0eb715ea91de771cf37a8104d47002205dcce64635c7cffc5a166728d442050de26da3c56d68e24832d7980147a5181d012102ce996b00874f354e79b4780f6a3d820b63c6ee41b6742d4df01c34cf6602546a024730440220779ae1ceb8a2da3e0c58b285d415bf037294c2d1fffffd1f85039cb53bd541f902205f9fad62b0a569092378751680e5a8b2285d1cb0e66e16b8a6252a4f5fa49f0801210283fce2a8de14bccdc4157fe6d9f53c9276be2ef4ef7e0a905b71ae6becb9a77eea7c0900")
	reqWithWitnessData := &proto.VerifySignedTransactionRequest{
		Chain:        "btc",
		SignedTxData: data,
		Vins: []*proto.Vin{
			{
				Amount:  1040233,
				Address: "31jbNXRfZPQvFm5pyzjwh8ofQ31bXY63Mq",
			},
			{
				Amount:  414386,
				Address: "34XgJPbLueTLHRXN4TcEg5EQx5E9QBpLCg",
			},
		},
	}

	t.Run("normal tx with witness data", func(t *testing.T) {
		_, err = client.VerifyUtxoSignedTransaction(context.TODO(), reqWithWitnessData)
		assert.NoError(t, err)
	})

	req := new(proto.VerifySignedTransactionRequest)
	*req = *reqWithWitnessData
	req.Vins[0].Amount = 1040232
	t.Run("wrong amount in segwit tx", func(t *testing.T) {
		_, err = client.VerifyUtxoSignedTransaction(context.TODO(), req)
		assert.Error(t, err)
	})

	*req = *reqWithWitnessData
	req.Vins[0].Address = "34XgJPbLueTLHRXN4TcEg5EQx5E9QBpLCg"
	t.Run("wrong address in segwit tx", func(t *testing.T) {
		_, err = client.VerifyUtxoSignedTransaction(context.TODO(), req)
		assert.Error(t, err)
	})

	data, err = hex.DecodeString("010000000140a6f95a972a9957a019c2741d7a391facec0cccb7def50f795ee4c94d527ec0000000006b483045022100f8155f8131dc301018963add1ad463ab90baf4d17dbc5cde602493163f7841ff02204645c2ef30db666e3a1a35e801e01c5a8b25f73601ec02c2b130777af03365ee012102258a470425a782183a46a3a95c648d9ca9114ca96c17c138f2187a7212688b17ffffffff029fbc01000000000017a914419d06580e87ab775ae791e24d0540bc8a81fd498756082100000000001976a9145a058a02183a0dd5ffe45d5e4123b053226a9d9a88ac00000000")
	reqWithoutWitnessData := &proto.VerifySignedTransactionRequest{
		Chain:        "btc",
		SignedTxData: data,
		Vins: []*proto.Vin{
			{
				Amount:  2279212,
				Address: "1JhXLWzqHyUdGts3S1vHtCAhB4UXKYutHc",
			},
		},
	}
	t.Run("normal legacy tx", func(t *testing.T) {
		_, err = client.VerifyUtxoSignedTransaction(context.TODO(), reqWithoutWitnessData)
		assert.NoError(t, err)
	})

	*req = *reqWithoutWitnessData
	req.Vins[0].Amount = 1040232
	t.Run("wrong amount in legacy tx", func(t *testing.T) {
		_, err = client.VerifyUtxoSignedTransaction(context.TODO(), req)
		assert.NoError(t, err)
	})

	*req = *reqWithoutWitnessData
	req.Vins[0].Address = "34XgJPbLueTLHRXN4TcEg5EQx5E9QBpLCg"
	t.Run("wrong address in legacy tx", func(t *testing.T) {
		_, err = client.VerifyUtxoSignedTransaction(context.TODO(), req)
		assert.Error(t, err)
	})

	data, err = hex.DecodeString("010000000001011057b543b42c8eeb9d237f5de3298c18d924ec9ebc79fe9d6b886f7f65f6ddef0100000000feffffff02e8030000000000002200204364063fac8829a931a752ecd9049e43425e2cebf83681876c556002bf389b0088900000000000001600144cdb88fcab2b39e572df9a51ff0f13c63b9ada9d02473044022070cc588fd54acdbf74161d68ff0a9e78547f4be22443d4c0f677be7d42cd80cb022058c78b9c894e6a067b9ab252b1a0bbeeb76bb160d14eb8e335a5ed53fc84a9020121020a8164c643f7589848fefae9385c4bacd7cc1c96f370ae53f39dbebe27ecd6e100000000")
	reqWithBcAddress := &proto.VerifySignedTransactionRequest{
		Chain:        "btc",
		SignedTxData: data,
		Vins: []*proto.Vin{
			{
				Amount:  42746,
				Address: "bc1qfndc3l9t9vu72uklnfgl7rcnccae4k5a3zrust",
			},
		},
	}
	t.Run("normal tx with bc address", func(t *testing.T) {
		_, err = client.VerifyUtxoSignedTransaction(context.TODO(), reqWithBcAddress)
		assert.NoError(t, err)
	})

}

func TestGetLatestBlockHeight(t *testing.T) {
	conf, err := config.New("./testnet.yaml")
	require.Nil(t, err)

	dispatcher, err := New(conf)
	require.Nil(t, err)

	height, err := dispatcher.GetLatestBlockHeight("btc")
	require.Nil(t, err)
	assert.Greater(t, height, int64(1692742))
	t.Log(height)

	height, err = dispatcher.GetLatestBlockHeight("eth")
	require.Nil(t, err)
	assert.Greater(t, height, int64(7626364))
	t.Log(height)

	height, err = dispatcher.GetLatestBlockHeight("tron")
	require.Nil(t, err)
	assert.Greater(t, height, int64(7244298))
	t.Log(height)

}

func TestGetUtxoTransactionByHeight(t *testing.T) {
	txs := GetUtxoTransactionByHeight(t, 1692700)
	require.Len(t, txs, 41)

	for i := 1692700 - 1000; i <= 1692700; i++ {
		j := i
		t.Run(strconv.Itoa(j), func(t2 *testing.T) {
			GetUtxoTransactionByHeight(t2, int64(j))
		})
	}
}

func GetUtxoTransactionByHeight(t *testing.T, height int64) []*proto.QueryUtxoTransactionReply {
	conf, err := config.New("./testnet.yaml")
	require.Nil(t, err)

	dispatcher, err := New(conf)
	require.Nil(t, err)

	txCh, errCh := dispatcher.GetUtxoTransactionByHeight("btc", height)
	txs := make([]*proto.QueryUtxoTransactionReply, 0)

loop:
	for {
		select {
		case tx, ok := <-txCh:
			if !ok {
				break loop
			}
			txs = append(txs, tx)
		case err, ok := <-errCh:
			if !ok {
				break loop
			}
			t.Error(err)
			t.FailNow()
		}

	}

	return txs
}

func TestGetAccountTransactionByHeightEth(t *testing.T) {
	txs := GetAccountTransactionByHeightEth(t, 7629980)
	require.Len(t, txs, 39)
	for i := 7631082 - 1000; i <= 7631082; i++ {
		j := i
		t.Run(strconv.Itoa(j), func(t2 *testing.T) {
			GetAccountTransactionByHeightEth(t2, int64(j))
		})
	}
}

func GetAccountTransactionByHeightEth(t *testing.T, height int64) []*proto.QueryAccountTransactionReply {
	conf, err := config.New("./testnet.yaml")
	require.Nil(t, err)

	dispatcher, err := New(conf)
	require.Nil(t, err)

	txCh, errCh := dispatcher.GetAccountTransactionByHeight("eth", height)
	txs := make([]*proto.QueryAccountTransactionReply, 0)
loop:
	for {
		select {
		case tx, ok := <-txCh:
			if !ok {
				break loop
			}
			txs = append(txs, tx)
		case err, ok := <-errCh:
			if !ok {
				break loop
			}
			t.Error(err)
			t.FailNow()
		}
	}

	return txs
}

func TestGetAccountTransactionByHeightTron(t *testing.T) {
	txs := GetAccountTransactionByHeightTron(t, 7239271)
	require.Len(t, txs, 2)

	total := 0
	for i := 7239271; i <= 7239271+50; i++ {
		j := i
		t.Run(strconv.Itoa(j), func(t2 *testing.T) {
			txs = GetAccountTransactionByHeightTron(t2, int64(j))
		})
		t.Logf("height:%v, len(txs):%v", i, len(txs))
		total += len(txs)
	}
	require.Equal(t, 18, total)
}

func GetAccountTransactionByHeightTron(t *testing.T, height int64) []*proto.QueryAccountTransactionReply {
	conf, err := config.New("./testnet.yaml")
	require.Nil(t, err)

	dispatcher, err := New(conf)
	require.Nil(t, err)

	txCh, errCh := dispatcher.GetAccountTransactionByHeight("trx", height)
	txs := make([]*proto.QueryAccountTransactionReply, 0)
loop:
	for {
		select {
		case tx, ok := <-txCh:
			if !ok {
				break loop
			}
			txs = append(txs, tx)
		case err, ok := <-errCh:
			if !ok {
				break loop
			}
			t.Log(err)
			//	t.FailNow()
		}
	}

	return txs
}
