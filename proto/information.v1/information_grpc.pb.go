// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.20.0
// source: information.proto

package information_v1

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// Information_ServiceClient is the client API for Information_Service service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type Information_ServiceClient interface {
	List(ctx context.Context, in *ListRequest, opts ...grpc.CallOption) (*Information, error)
	Watch(ctx context.Context, in *WatchRequest, opts ...grpc.CallOption) (Information_Service_WatchClient, error)
}

type information_ServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewInformation_ServiceClient(cc grpc.ClientConnInterface) Information_ServiceClient {
	return &information_ServiceClient{cc}
}

func (c *information_ServiceClient) List(ctx context.Context, in *ListRequest, opts ...grpc.CallOption) (*Information, error) {
	out := new(Information)
	err := c.cc.Invoke(ctx, "/api.v1.Information_Service/List", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *information_ServiceClient) Watch(ctx context.Context, in *WatchRequest, opts ...grpc.CallOption) (Information_Service_WatchClient, error) {
	stream, err := c.cc.NewStream(ctx, &Information_Service_ServiceDesc.Streams[0], "/api.v1.Information_Service/Watch", opts...)
	if err != nil {
		return nil, err
	}
	x := &information_ServiceWatchClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type Information_Service_WatchClient interface {
	Recv() (*WatchResponse, error)
	grpc.ClientStream
}

type information_ServiceWatchClient struct {
	grpc.ClientStream
}

func (x *information_ServiceWatchClient) Recv() (*WatchResponse, error) {
	m := new(WatchResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// Information_ServiceServer is the server API for Information_Service service.
// All implementations must embed UnimplementedInformation_ServiceServer
// for forward compatibility
type Information_ServiceServer interface {
	List(context.Context, *ListRequest) (*Information, error)
	Watch(*WatchRequest, Information_Service_WatchServer) error
	mustEmbedUnimplementedInformation_ServiceServer()
}

// UnimplementedInformation_ServiceServer must be embedded to have forward compatible implementations.
type UnimplementedInformation_ServiceServer struct {
}

func (UnimplementedInformation_ServiceServer) List(context.Context, *ListRequest) (*Information, error) {
	return nil, status.Errorf(codes.Unimplemented, "method List not implemented")
}
func (UnimplementedInformation_ServiceServer) Watch(*WatchRequest, Information_Service_WatchServer) error {
	return status.Errorf(codes.Unimplemented, "method Watch not implemented")
}
func (UnimplementedInformation_ServiceServer) mustEmbedUnimplementedInformation_ServiceServer() {}

// UnsafeInformation_ServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to Information_ServiceServer will
// result in compilation errors.
type UnsafeInformation_ServiceServer interface {
	mustEmbedUnimplementedInformation_ServiceServer()
}

func RegisterInformation_ServiceServer(s grpc.ServiceRegistrar, srv Information_ServiceServer) {
	s.RegisterService(&Information_Service_ServiceDesc, srv)
}

func _Information_Service_List_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(Information_ServiceServer).List(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.v1.Information_Service/List",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(Information_ServiceServer).List(ctx, req.(*ListRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Information_Service_Watch_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(WatchRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(Information_ServiceServer).Watch(m, &information_ServiceWatchServer{stream})
}

type Information_Service_WatchServer interface {
	Send(*WatchResponse) error
	grpc.ServerStream
}

type information_ServiceWatchServer struct {
	grpc.ServerStream
}

func (x *information_ServiceWatchServer) Send(m *WatchResponse) error {
	return x.ServerStream.SendMsg(m)
}

// Information_Service_ServiceDesc is the grpc.ServiceDesc for Information_Service service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Information_Service_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "api.v1.Information_Service",
	HandlerType: (*Information_ServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "List",
			Handler:    _Information_Service_List_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Watch",
			Handler:       _Information_Service_Watch_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "information.proto",
}