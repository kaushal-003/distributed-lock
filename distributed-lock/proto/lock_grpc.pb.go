// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v3.12.4
// source: proto/lock.proto

package proto

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	DistributedLock_InitConnection_FullMethodName = "/distributed_lock.DistributedLock/InitConnection"
	DistributedLock_LockAcquire_FullMethodName    = "/distributed_lock.DistributedLock/LockAcquire"
	DistributedLock_LockRelease_FullMethodName    = "/distributed_lock.DistributedLock/LockRelease"
	DistributedLock_AppendFile_FullMethodName     = "/distributed_lock.DistributedLock/AppendFile"
	DistributedLock_GetQueueIndex_FullMethodName  = "/distributed_lock.DistributedLock/GetQueueIndex"
	DistributedLock_UpdateLeader_FullMethodName   = "/distributed_lock.DistributedLock/UpdateLeader"
	DistributedLock_Heartbeat_FullMethodName      = "/distributed_lock.DistributedLock/Heartbeat"
	DistributedLock_AddQueue_FullMethodName       = "/distributed_lock.DistributedLock/AddQueue"
	DistributedLock_RemoveQueue_FullMethodName    = "/distributed_lock.DistributedLock/RemoveQueue"
	DistributedLock_GetQueueState_FullMethodName  = "/distributed_lock.DistributedLock/GetQueueState"
)

// DistributedLockClient is the client API for DistributedLock service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type DistributedLockClient interface {
	InitConnection(ctx context.Context, in *InitRequest, opts ...grpc.CallOption) (*InitResponse, error)
	LockAcquire(ctx context.Context, in *LockRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[LockResponse], error)
	LockRelease(ctx context.Context, in *LockRequest, opts ...grpc.CallOption) (*LockResponse, error)
	AppendFile(ctx context.Context, in *AppendRequest, opts ...grpc.CallOption) (*AppendResponse, error)
	GetQueueIndex(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*GetQueueIndexResponse, error)
	UpdateLeader(ctx context.Context, in *UpdateLeaderRequest, opts ...grpc.CallOption) (*Empty, error)
	Heartbeat(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*Empty, error)
	AddQueue(ctx context.Context, in *AddQueueRequest, opts ...grpc.CallOption) (*Empty, error)
	RemoveQueue(ctx context.Context, in *RemoveQueueRequest, opts ...grpc.CallOption) (*Empty, error)
	GetQueueState(ctx context.Context, in *GetQueueRequest, opts ...grpc.CallOption) (*GetQueueResponse, error)
}

type distributedLockClient struct {
	cc grpc.ClientConnInterface
}

func NewDistributedLockClient(cc grpc.ClientConnInterface) DistributedLockClient {
	return &distributedLockClient{cc}
}

func (c *distributedLockClient) InitConnection(ctx context.Context, in *InitRequest, opts ...grpc.CallOption) (*InitResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(InitResponse)
	err := c.cc.Invoke(ctx, DistributedLock_InitConnection_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *distributedLockClient) LockAcquire(ctx context.Context, in *LockRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[LockResponse], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &DistributedLock_ServiceDesc.Streams[0], DistributedLock_LockAcquire_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[LockRequest, LockResponse]{ClientStream: stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type DistributedLock_LockAcquireClient = grpc.ServerStreamingClient[LockResponse]

func (c *distributedLockClient) LockRelease(ctx context.Context, in *LockRequest, opts ...grpc.CallOption) (*LockResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(LockResponse)
	err := c.cc.Invoke(ctx, DistributedLock_LockRelease_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *distributedLockClient) AppendFile(ctx context.Context, in *AppendRequest, opts ...grpc.CallOption) (*AppendResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(AppendResponse)
	err := c.cc.Invoke(ctx, DistributedLock_AppendFile_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *distributedLockClient) GetQueueIndex(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*GetQueueIndexResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetQueueIndexResponse)
	err := c.cc.Invoke(ctx, DistributedLock_GetQueueIndex_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *distributedLockClient) UpdateLeader(ctx context.Context, in *UpdateLeaderRequest, opts ...grpc.CallOption) (*Empty, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Empty)
	err := c.cc.Invoke(ctx, DistributedLock_UpdateLeader_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *distributedLockClient) Heartbeat(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*Empty, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Empty)
	err := c.cc.Invoke(ctx, DistributedLock_Heartbeat_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *distributedLockClient) AddQueue(ctx context.Context, in *AddQueueRequest, opts ...grpc.CallOption) (*Empty, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Empty)
	err := c.cc.Invoke(ctx, DistributedLock_AddQueue_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *distributedLockClient) RemoveQueue(ctx context.Context, in *RemoveQueueRequest, opts ...grpc.CallOption) (*Empty, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Empty)
	err := c.cc.Invoke(ctx, DistributedLock_RemoveQueue_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *distributedLockClient) GetQueueState(ctx context.Context, in *GetQueueRequest, opts ...grpc.CallOption) (*GetQueueResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetQueueResponse)
	err := c.cc.Invoke(ctx, DistributedLock_GetQueueState_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// DistributedLockServer is the server API for DistributedLock service.
// All implementations must embed UnimplementedDistributedLockServer
// for forward compatibility.
type DistributedLockServer interface {
	InitConnection(context.Context, *InitRequest) (*InitResponse, error)
	LockAcquire(*LockRequest, grpc.ServerStreamingServer[LockResponse]) error
	LockRelease(context.Context, *LockRequest) (*LockResponse, error)
	AppendFile(context.Context, *AppendRequest) (*AppendResponse, error)
	GetQueueIndex(context.Context, *Empty) (*GetQueueIndexResponse, error)
	UpdateLeader(context.Context, *UpdateLeaderRequest) (*Empty, error)
	Heartbeat(context.Context, *Empty) (*Empty, error)
	AddQueue(context.Context, *AddQueueRequest) (*Empty, error)
	RemoveQueue(context.Context, *RemoveQueueRequest) (*Empty, error)
	GetQueueState(context.Context, *GetQueueRequest) (*GetQueueResponse, error)
	mustEmbedUnimplementedDistributedLockServer()
}

// UnimplementedDistributedLockServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedDistributedLockServer struct{}

func (UnimplementedDistributedLockServer) InitConnection(context.Context, *InitRequest) (*InitResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method InitConnection not implemented")
}
func (UnimplementedDistributedLockServer) LockAcquire(*LockRequest, grpc.ServerStreamingServer[LockResponse]) error {
	return status.Errorf(codes.Unimplemented, "method LockAcquire not implemented")
}
func (UnimplementedDistributedLockServer) LockRelease(context.Context, *LockRequest) (*LockResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method LockRelease not implemented")
}
func (UnimplementedDistributedLockServer) AppendFile(context.Context, *AppendRequest) (*AppendResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AppendFile not implemented")
}
func (UnimplementedDistributedLockServer) GetQueueIndex(context.Context, *Empty) (*GetQueueIndexResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetQueueIndex not implemented")
}
func (UnimplementedDistributedLockServer) UpdateLeader(context.Context, *UpdateLeaderRequest) (*Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateLeader not implemented")
}
func (UnimplementedDistributedLockServer) Heartbeat(context.Context, *Empty) (*Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Heartbeat not implemented")
}
func (UnimplementedDistributedLockServer) AddQueue(context.Context, *AddQueueRequest) (*Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AddQueue not implemented")
}
func (UnimplementedDistributedLockServer) RemoveQueue(context.Context, *RemoveQueueRequest) (*Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RemoveQueue not implemented")
}
func (UnimplementedDistributedLockServer) GetQueueState(context.Context, *GetQueueRequest) (*GetQueueResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetQueueState not implemented")
}
func (UnimplementedDistributedLockServer) mustEmbedUnimplementedDistributedLockServer() {}
func (UnimplementedDistributedLockServer) testEmbeddedByValue()                         {}

// UnsafeDistributedLockServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to DistributedLockServer will
// result in compilation errors.
type UnsafeDistributedLockServer interface {
	mustEmbedUnimplementedDistributedLockServer()
}

func RegisterDistributedLockServer(s grpc.ServiceRegistrar, srv DistributedLockServer) {
	// If the following call pancis, it indicates UnimplementedDistributedLockServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&DistributedLock_ServiceDesc, srv)
}

func _DistributedLock_InitConnection_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(InitRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DistributedLockServer).InitConnection(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: DistributedLock_InitConnection_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DistributedLockServer).InitConnection(ctx, req.(*InitRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DistributedLock_LockAcquire_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(LockRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(DistributedLockServer).LockAcquire(m, &grpc.GenericServerStream[LockRequest, LockResponse]{ServerStream: stream})
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type DistributedLock_LockAcquireServer = grpc.ServerStreamingServer[LockResponse]

func _DistributedLock_LockRelease_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(LockRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DistributedLockServer).LockRelease(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: DistributedLock_LockRelease_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DistributedLockServer).LockRelease(ctx, req.(*LockRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DistributedLock_AppendFile_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AppendRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DistributedLockServer).AppendFile(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: DistributedLock_AppendFile_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DistributedLockServer).AppendFile(ctx, req.(*AppendRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DistributedLock_GetQueueIndex_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DistributedLockServer).GetQueueIndex(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: DistributedLock_GetQueueIndex_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DistributedLockServer).GetQueueIndex(ctx, req.(*Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _DistributedLock_UpdateLeader_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateLeaderRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DistributedLockServer).UpdateLeader(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: DistributedLock_UpdateLeader_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DistributedLockServer).UpdateLeader(ctx, req.(*UpdateLeaderRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DistributedLock_Heartbeat_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DistributedLockServer).Heartbeat(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: DistributedLock_Heartbeat_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DistributedLockServer).Heartbeat(ctx, req.(*Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _DistributedLock_AddQueue_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AddQueueRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DistributedLockServer).AddQueue(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: DistributedLock_AddQueue_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DistributedLockServer).AddQueue(ctx, req.(*AddQueueRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DistributedLock_RemoveQueue_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RemoveQueueRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DistributedLockServer).RemoveQueue(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: DistributedLock_RemoveQueue_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DistributedLockServer).RemoveQueue(ctx, req.(*RemoveQueueRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DistributedLock_GetQueueState_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetQueueRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DistributedLockServer).GetQueueState(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: DistributedLock_GetQueueState_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DistributedLockServer).GetQueueState(ctx, req.(*GetQueueRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// DistributedLock_ServiceDesc is the grpc.ServiceDesc for DistributedLock service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var DistributedLock_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "distributed_lock.DistributedLock",
	HandlerType: (*DistributedLockServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "InitConnection",
			Handler:    _DistributedLock_InitConnection_Handler,
		},
		{
			MethodName: "LockRelease",
			Handler:    _DistributedLock_LockRelease_Handler,
		},
		{
			MethodName: "AppendFile",
			Handler:    _DistributedLock_AppendFile_Handler,
		},
		{
			MethodName: "GetQueueIndex",
			Handler:    _DistributedLock_GetQueueIndex_Handler,
		},
		{
			MethodName: "UpdateLeader",
			Handler:    _DistributedLock_UpdateLeader_Handler,
		},
		{
			MethodName: "Heartbeat",
			Handler:    _DistributedLock_Heartbeat_Handler,
		},
		{
			MethodName: "AddQueue",
			Handler:    _DistributedLock_AddQueue_Handler,
		},
		{
			MethodName: "RemoveQueue",
			Handler:    _DistributedLock_RemoveQueue_Handler,
		},
		{
			MethodName: "GetQueueState",
			Handler:    _DistributedLock_GetQueueState_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "LockAcquire",
			Handler:       _DistributedLock_LockAcquire_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "proto/lock.proto",
}
