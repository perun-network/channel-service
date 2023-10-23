// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v4.24.3
// source: perun-wallet.proto

package proto

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

// ChannelServiceClient is the client API for ChannelService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ChannelServiceClient interface {
	// Initiate channel opening.
	OpenChannel(ctx context.Context, in *ChannelOpenRequest, opts ...grpc.CallOption) (*ChannelOpenResponse, error)
	// Initiate some channel update.
	UpdateChannel(ctx context.Context, in *ChannelUpdateRequest, opts ...grpc.CallOption) (*ChannelUpdateResponse, error)
	// Initiate channel closing.
	CloseChannel(ctx context.Context, in *ChannelCloseRequest, opts ...grpc.CallOption) (*ChannelCloseResponse, error)
}

type channelServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewChannelServiceClient(cc grpc.ClientConnInterface) ChannelServiceClient {
	return &channelServiceClient{cc}
}

func (c *channelServiceClient) OpenChannel(ctx context.Context, in *ChannelOpenRequest, opts ...grpc.CallOption) (*ChannelOpenResponse, error) {
	out := new(ChannelOpenResponse)
	err := c.cc.Invoke(ctx, "/perunservice.ChannelService/OpenChannel", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *channelServiceClient) UpdateChannel(ctx context.Context, in *ChannelUpdateRequest, opts ...grpc.CallOption) (*ChannelUpdateResponse, error) {
	out := new(ChannelUpdateResponse)
	err := c.cc.Invoke(ctx, "/perunservice.ChannelService/UpdateChannel", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *channelServiceClient) CloseChannel(ctx context.Context, in *ChannelCloseRequest, opts ...grpc.CallOption) (*ChannelCloseResponse, error) {
	out := new(ChannelCloseResponse)
	err := c.cc.Invoke(ctx, "/perunservice.ChannelService/CloseChannel", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ChannelServiceServer is the server API for ChannelService service.
// All implementations must embed UnimplementedChannelServiceServer
// for forward compatibility
type ChannelServiceServer interface {
	// Initiate channel opening.
	OpenChannel(context.Context, *ChannelOpenRequest) (*ChannelOpenResponse, error)
	// Initiate some channel update.
	UpdateChannel(context.Context, *ChannelUpdateRequest) (*ChannelUpdateResponse, error)
	// Initiate channel closing.
	CloseChannel(context.Context, *ChannelCloseRequest) (*ChannelCloseResponse, error)
	mustEmbedUnimplementedChannelServiceServer()
}

// UnimplementedChannelServiceServer must be embedded to have forward compatible implementations.
type UnimplementedChannelServiceServer struct {
}

func (UnimplementedChannelServiceServer) OpenChannel(context.Context, *ChannelOpenRequest) (*ChannelOpenResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method OpenChannel not implemented")
}
func (UnimplementedChannelServiceServer) UpdateChannel(context.Context, *ChannelUpdateRequest) (*ChannelUpdateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateChannel not implemented")
}
func (UnimplementedChannelServiceServer) CloseChannel(context.Context, *ChannelCloseRequest) (*ChannelCloseResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CloseChannel not implemented")
}
func (UnimplementedChannelServiceServer) mustEmbedUnimplementedChannelServiceServer() {}

// UnsafeChannelServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ChannelServiceServer will
// result in compilation errors.
type UnsafeChannelServiceServer interface {
	mustEmbedUnimplementedChannelServiceServer()
}

func RegisterChannelServiceServer(s grpc.ServiceRegistrar, srv ChannelServiceServer) {
	s.RegisterService(&ChannelService_ServiceDesc, srv)
}

func _ChannelService_OpenChannel_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ChannelOpenRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ChannelServiceServer).OpenChannel(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/perunservice.ChannelService/OpenChannel",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ChannelServiceServer).OpenChannel(ctx, req.(*ChannelOpenRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ChannelService_UpdateChannel_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ChannelUpdateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ChannelServiceServer).UpdateChannel(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/perunservice.ChannelService/UpdateChannel",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ChannelServiceServer).UpdateChannel(ctx, req.(*ChannelUpdateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ChannelService_CloseChannel_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ChannelCloseRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ChannelServiceServer).CloseChannel(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/perunservice.ChannelService/CloseChannel",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ChannelServiceServer).CloseChannel(ctx, req.(*ChannelCloseRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// ChannelService_ServiceDesc is the grpc.ServiceDesc for ChannelService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var ChannelService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "perunservice.ChannelService",
	HandlerType: (*ChannelServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "OpenChannel",
			Handler:    _ChannelService_OpenChannel_Handler,
		},
		{
			MethodName: "UpdateChannel",
			Handler:    _ChannelService_UpdateChannel_Handler,
		},
		{
			MethodName: "CloseChannel",
			Handler:    _ChannelService_CloseChannel_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "perun-wallet.proto",
}

// WalletServiceClient is the client API for WalletService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type WalletServiceClient interface {
	// Requesting a channel opening from the wallet. This happens if the Perun
	// channel service received a channel opening request from another peer.
	// This method lets the wallet know that it should ask the user whether or
	// not to accept the channel opening request.
	OpenChannel(ctx context.Context, in *OpenChannelRequest, opts ...grpc.CallOption) (*OpenChannelResponse, error)
	// The Perun channel service calls this method if it received a channel
	// update request from another peer. The wallet might use this channel update
	// request containing the proposed/new channel state to shown it in the
	// front-end. The wallet might use this update event to query the user
	// whether or not to accept the channel update.
	UpdateNotification(ctx context.Context, in *UpdateNotificationRequest, opts ...grpc.CallOption) (*UpdateNotificationResponse, error)
	// Request a signature on the given message by some wallet.
	SignMessage(ctx context.Context, in *SignMessageRequest, opts ...grpc.CallOption) (*SignMessageResponse, error)
	// Request a signature on the given transaction by some wallet.
	SignTransaction(ctx context.Context, in *SignTransactionRequest, opts ...grpc.CallOption) (*SignTransactionResponse, error)
	// Request a list outpoints from a wallet at least matching the requested
	// amount of possibly different assets. This can be called by the Perun
	// channel backend if it builds transactions.
	GetAssets(ctx context.Context, in *GetAssetsRequest, opts ...grpc.CallOption) (*GetAssetsResponse, error)
}

type walletServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewWalletServiceClient(cc grpc.ClientConnInterface) WalletServiceClient {
	return &walletServiceClient{cc}
}

func (c *walletServiceClient) OpenChannel(ctx context.Context, in *OpenChannelRequest, opts ...grpc.CallOption) (*OpenChannelResponse, error) {
	out := new(OpenChannelResponse)
	err := c.cc.Invoke(ctx, "/perunservice.WalletService/OpenChannel", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *walletServiceClient) UpdateNotification(ctx context.Context, in *UpdateNotificationRequest, opts ...grpc.CallOption) (*UpdateNotificationResponse, error) {
	out := new(UpdateNotificationResponse)
	err := c.cc.Invoke(ctx, "/perunservice.WalletService/UpdateNotification", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *walletServiceClient) SignMessage(ctx context.Context, in *SignMessageRequest, opts ...grpc.CallOption) (*SignMessageResponse, error) {
	out := new(SignMessageResponse)
	err := c.cc.Invoke(ctx, "/perunservice.WalletService/SignMessage", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *walletServiceClient) SignTransaction(ctx context.Context, in *SignTransactionRequest, opts ...grpc.CallOption) (*SignTransactionResponse, error) {
	out := new(SignTransactionResponse)
	err := c.cc.Invoke(ctx, "/perunservice.WalletService/SignTransaction", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *walletServiceClient) GetAssets(ctx context.Context, in *GetAssetsRequest, opts ...grpc.CallOption) (*GetAssetsResponse, error) {
	out := new(GetAssetsResponse)
	err := c.cc.Invoke(ctx, "/perunservice.WalletService/GetAssets", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// WalletServiceServer is the server API for WalletService service.
// All implementations must embed UnimplementedWalletServiceServer
// for forward compatibility
type WalletServiceServer interface {
	// Requesting a channel opening from the wallet. This happens if the Perun
	// channel service received a channel opening request from another peer.
	// This method lets the wallet know that it should ask the user whether or
	// not to accept the channel opening request.
	OpenChannel(context.Context, *OpenChannelRequest) (*OpenChannelResponse, error)
	// The Perun channel service calls this method if it received a channel
	// update request from another peer. The wallet might use this channel update
	// request containing the proposed/new channel state to shown it in the
	// front-end. The wallet might use this update event to query the user
	// whether or not to accept the channel update.
	UpdateNotification(context.Context, *UpdateNotificationRequest) (*UpdateNotificationResponse, error)
	// Request a signature on the given message by some wallet.
	SignMessage(context.Context, *SignMessageRequest) (*SignMessageResponse, error)
	// Request a signature on the given transaction by some wallet.
	SignTransaction(context.Context, *SignTransactionRequest) (*SignTransactionResponse, error)
	// Request a list outpoints from a wallet at least matching the requested
	// amount of possibly different assets. This can be called by the Perun
	// channel backend if it builds transactions.
	GetAssets(context.Context, *GetAssetsRequest) (*GetAssetsResponse, error)
	mustEmbedUnimplementedWalletServiceServer()
}

// UnimplementedWalletServiceServer must be embedded to have forward compatible implementations.
type UnimplementedWalletServiceServer struct {
}

func (UnimplementedWalletServiceServer) OpenChannel(context.Context, *OpenChannelRequest) (*OpenChannelResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method OpenChannel not implemented")
}
func (UnimplementedWalletServiceServer) UpdateNotification(context.Context, *UpdateNotificationRequest) (*UpdateNotificationResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateNotification not implemented")
}
func (UnimplementedWalletServiceServer) SignMessage(context.Context, *SignMessageRequest) (*SignMessageResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SignMessage not implemented")
}
func (UnimplementedWalletServiceServer) SignTransaction(context.Context, *SignTransactionRequest) (*SignTransactionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SignTransaction not implemented")
}
func (UnimplementedWalletServiceServer) GetAssets(context.Context, *GetAssetsRequest) (*GetAssetsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetAssets not implemented")
}
func (UnimplementedWalletServiceServer) mustEmbedUnimplementedWalletServiceServer() {}

// UnsafeWalletServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to WalletServiceServer will
// result in compilation errors.
type UnsafeWalletServiceServer interface {
	mustEmbedUnimplementedWalletServiceServer()
}

func RegisterWalletServiceServer(s grpc.ServiceRegistrar, srv WalletServiceServer) {
	s.RegisterService(&WalletService_ServiceDesc, srv)
}

func _WalletService_OpenChannel_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(OpenChannelRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WalletServiceServer).OpenChannel(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/perunservice.WalletService/OpenChannel",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WalletServiceServer).OpenChannel(ctx, req.(*OpenChannelRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WalletService_UpdateNotification_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateNotificationRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WalletServiceServer).UpdateNotification(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/perunservice.WalletService/UpdateNotification",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WalletServiceServer).UpdateNotification(ctx, req.(*UpdateNotificationRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WalletService_SignMessage_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SignMessageRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WalletServiceServer).SignMessage(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/perunservice.WalletService/SignMessage",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WalletServiceServer).SignMessage(ctx, req.(*SignMessageRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WalletService_SignTransaction_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SignTransactionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WalletServiceServer).SignTransaction(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/perunservice.WalletService/SignTransaction",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WalletServiceServer).SignTransaction(ctx, req.(*SignTransactionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WalletService_GetAssets_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetAssetsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WalletServiceServer).GetAssets(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/perunservice.WalletService/GetAssets",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WalletServiceServer).GetAssets(ctx, req.(*GetAssetsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// WalletService_ServiceDesc is the grpc.ServiceDesc for WalletService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var WalletService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "perunservice.WalletService",
	HandlerType: (*WalletServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "OpenChannel",
			Handler:    _WalletService_OpenChannel_Handler,
		},
		{
			MethodName: "UpdateNotification",
			Handler:    _WalletService_UpdateNotification_Handler,
		},
		{
			MethodName: "SignMessage",
			Handler:    _WalletService_SignMessage_Handler,
		},
		{
			MethodName: "SignTransaction",
			Handler:    _WalletService_SignTransaction_Handler,
		},
		{
			MethodName: "GetAssets",
			Handler:    _WalletService_GetAssets_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "perun-wallet.proto",
}