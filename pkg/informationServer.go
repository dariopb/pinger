package pinger

import (
	"context"
	"fmt"
	"math"
	"net"
	"os"
	"strings"
	"syscall"
	"time"

	information_v1 "github.com/dariopb/pinger/proto/information.v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/reflection"

	log "github.com/sirupsen/logrus"
)

type InformationGrpcServer struct {
	information_v1.UnimplementedInformation_ServiceServer
	grpcServer *grpc.Server
}

func NewGRPCServer(grpcaddress string) (*InformationGrpcServer, error) {
	log.Info("Starting grpc server on address: ", grpcaddress)

	var lis net.Listener
	var err error

	if strings.HasPrefix(grpcaddress, "unix:///") {
		path := grpcaddress[7:]

		// always remove the named socket from the fs if its there...
		err = syscall.Unlink(path)
		if err != nil {
			log.Error("Unlink()", err)
		}

		lis, err = net.Listen("unix", path)
	} else {
		lis, err = net.Listen("tcp", grpcaddress)
	}

	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	inf := &InformationGrpcServer{}
	inf.grpcServer = grpc.NewServer()

	information_v1.RegisterInformation_ServiceServer(inf.grpcServer, inf)

	// Register reflection service on gRPC server.
	reflection.Register(inf.grpcServer)

	go func() {
		if err := inf.grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve grpc: %v", err)
		}
	}()

	return inf, nil
}

func (s *InformationGrpcServer) List(ctx context.Context, req *information_v1.ListRequest) (*information_v1.Information, error) {
	p, ok := peer.FromContext(ctx)
	if ok {
		log.Info("Got grpc List call from: ", p.Addr)
	}

	info := &information_v1.Information{
		Description:  "Pinger Information",
		EnvVariables: []*information_v1.KV{},
	}

	for _, ev := range os.Environ() {
		kv := &information_v1.KV{
			Key:   ev[:strings.Index(ev, "=")],
			Value: ev[strings.Index(ev, "=")+1:],
		}
		info.EnvVariables = append(info.EnvVariables, kv)
	}
	return info, nil
}

func (s *InformationGrpcServer) Watch(req *information_v1.WatchRequest, ws information_v1.Information_Service_WatchServer) error {
	p, ok := peer.FromContext(ws.Context())
	if ok {
		log.Info("Got grpc Watch call from: ", p.Addr)
	}

	var count, until int32 = 0, math.MaxInt32
	if req.StopAfterCount != 0 {
		until = req.StopAfterCount
	}

	for count = 0; count < until; count++ {
		r := &information_v1.WatchResponse{
			Name: "pinger",
			Id:   count,
		}
		if err := ws.Send(r); err != nil {
			log.Info(fmt.Sprintf("finishing grpc Watch call from: %s, got: [%v]", p.Addr, err))
			return err

		}
		time.Sleep(time.Second * 1)
	}

	return nil
}
