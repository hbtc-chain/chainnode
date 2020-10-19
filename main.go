//go:generate protoc --go_out=plugins=grpc:. proto/chainnode.proto

package main

import (
	"flag"
	"net"

	"github.com/hbtc-chain/chainnode/chaindispatcher"
	"github.com/hbtc-chain/chainnode/config"
	"github.com/hbtc-chain/chainnode/proto"

	"github.com/ethereum/go-ethereum/log"
	"google.golang.org/grpc"
)

func main() {
	var f = flag.String("c", "config.yml", "config path")
	flag.Parse()
	conf, err := config.New(*f)
	if err != nil {
		panic(err)
	}

	dispatcher, err := chaindispatcher.New(conf)
	if err != nil {
		log.Error("Setup dispatcher failed", "err", err)
		panic(err)
	}

	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(dispatcher.Interceptor))
	defer grpcServer.GracefulStop()

	proto.RegisterChainnodeServer(grpcServer, dispatcher)

	listen, err := net.Listen("tcp", ":"+conf.Server.Port)
	if err != nil {
		log.Error("net listen failed", "err", err)
		panic(err)
	}
	log.Info("chainnode start success", "port", conf.Server.Port)

	if err := grpcServer.Serve(listen); err != nil {
		log.Error("grpc server serve failed", "err", err)
		panic(err)
	}
}
