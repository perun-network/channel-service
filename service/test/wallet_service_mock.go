package test

import (
	"context"

	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

type MockWalletServiceClient struct {
	mock.Mock
}

func (m *MockWalletServiceClient) OpenChannel(ctx context.Context, in *OpenChannelRequest, opts ...grpc.CallOption) (*OpenChannelResponse, error) {
	args := m.Called(ctx, in, opts)
	return args.Get(0).(*OpenChannelResponse), args.Error(1)
}

func (m *MockWalletServiceClient) UpdateNotification(ctx context.Context, in *UpdateNotificationRequest, opts ...grpc.CallOption) (*UpdateNotificationResponse, error) {
	args := m.Called(ctx, in, opts)
	return args.Get(0).(*UpdateNotificationResponse), args.Error(1)
}

func (m *MockWalletServiceClient) SignMessage(ctx context.Context, in *SignMessageRequest, opts ...grpc.CallOption) (*SignMessageResponse, error) {
	args := m.Called(ctx, in, opts)
	return args.Get(0).(*SignMessageResponse), args.Error(1)
}

func (m *MockWalletServiceClient) SignTransaction(ctx context.Context, in *SignTransactionRequest, opts ...grpc.CallOption) (*SignTransactionResponse, error) {
	args := m.Called(ctx, in, opts)
	return args.Get(0).(*SignTransactionResponse), args.Error(1)
}

func (m *MockWalletServiceClient) GetAssets(ctx context.Context, in *GetAssetsRequest, opts ...grpc.CallOption) (*GetAssetsResponse, error) {
	args := m.Called(ctx, in, opts)
	return args.Get(0).(*GetAssetsResponse), args.Error(1)
}
