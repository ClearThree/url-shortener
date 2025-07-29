// Package proto stores all the required files for GRPC server, including .proto file, interfaces and structures and
// implementation of GRPC-service, which is a facade over ShortURLService
package proto

import (
	"context"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/clearthree/url-shortener/internal/app/config"
	"github.com/clearthree/url-shortener/internal/app/handlers"
	"github.com/clearthree/url-shortener/internal/app/models"
	"github.com/clearthree/url-shortener/internal/app/service"
)

// ShortenerGRPCServer Supports all the service methods
type ShortenerGRPCServer struct {
	UnimplementedURLShortenerServiceServer

	service service.ShortURLServiceInterface
}

// NewShortenerGRPCServer creates the ShortenerGRPCServer structure and returns a pointer to freshly created struct.
func NewShortenerGRPCServer(service service.ShortURLServiceInterface) *ShortenerGRPCServer {
	return &ShortenerGRPCServer{
		service: service,
	}
}

// CreateShortURL - RPC handler to create the shortURL from the given original URL.
func (s ShortenerGRPCServer) CreateShortURL(ctx context.Context, request *ShortenRequest) (*ShortenResponse, error) {
	if request.Url == "" {
		return nil, status.Error(codes.InvalidArgument, "URL is required")
	}
	if request.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "UserId is required")
	}
	if !handlers.IsURL(request.Url) {
		return nil, status.Error(codes.InvalidArgument, "URL is invalid")
	}
	var response ShortenResponse
	result, err := s.service.Create(ctx, request.Url, request.UserId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	response.Result = result
	return &response, nil
}

// BatchCreateShortURL - RPC handler to create a batch of shortURLs from the given batch of original URLs.
func (s ShortenerGRPCServer) BatchCreateShortURL(ctx context.Context, request *BatchShortenRequest) (*BatchShortenResponse, error) {
	if len(request.Items) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Items required")
	}
	if request.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "UserId required")
	}
	requestData := make([]models.ShortenBatchItemRequest, len(request.Items))
	for i, item := range request.Items {
		requestData[i] = models.ShortenBatchItemRequest{
			CorrelationID: item.CorrelationId,
			OriginalURL:   item.OriginalUrl,
		}
	}
	result, err := s.service.BatchCreate(ctx, requestData, request.UserId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	var response BatchShortenResponse
	for _, item := range result {
		response.Items = append(response.Items, &BatchShortenResponse_Item{
			CorrelationId: item.CorrelationID, ShortUrl: item.ShortURL})
	}
	return &response, nil
}

// GetUserURLs - RPC handler that returns all the URLs created by user.
func (s ShortenerGRPCServer) GetUserURLs(ctx context.Context, request *GetUserURLsRequest) (*GetUserURLsResponse, error) {
	if request.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "UserID is required")
	}
	result, err := s.service.ReadByUserID(ctx, request.UserId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if len(result) == 0 {
		return &GetUserURLsResponse{}, nil
	}
	var response GetUserURLsResponse
	for _, item := range result {
		response.Urls = append(response.Urls, &GetUserURLsResponse_URL{
			ShortUrl: item.ShortURL, OriginalUrl: item.OriginalURL})
	}
	return &response, nil
}

// DeleteBatchURLs - RPC handler that schedules the deletion of the URL batch (if they belong to the current user).
func (s ShortenerGRPCServer) DeleteBatchURLs(ctx context.Context, request *DeleteBatchRequest) (*emptypb.Empty, error) {
	if request.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "UserID is required")
	}
	if len(request.ShortUrls) == 0 {
		return nil, status.Error(codes.InvalidArgument, "ShortUrls required")
	}
	requestPrepared := make([]models.ShortURLChannelMessage, len(request.ShortUrls))
	for i, requestItem := range request.ShortUrls {
		requestPrepared[i] = models.ShortURLChannelMessage{
			Ctx:      ctx,
			ShortURL: requestItem,
			UserID:   request.UserId,
		}
	}
	s.service.ScheduleDeletionOfBatch(requestPrepared)
	return &emptypb.Empty{}, nil
}

// GetServiceStats - RPC handler that returns the statistics of the service.
func (s ShortenerGRPCServer) GetServiceStats(ctx context.Context, _ *ServiceStatsRequest) (*ServiceStatsResponse, error) {
	result, err := s.service.GetStats(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	response := &ServiceStatsResponse{
		Users: uint32(result.Users),
		Urls:  uint32(result.URLs),
	}
	return response, nil
}

// Ping RPC handler to ping the service.
func (s ShortenerGRPCServer) Ping(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	err := s.service.Ping(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &emptypb.Empty{}, nil
}

// AuthFn is a custom auth-function that checks the header presence.
func AuthFn(ctx context.Context) (context.Context, error) {
	token, err := auth.AuthFromMD(ctx, "bearer")
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}
	if token != config.Settings.GRPCToken {
		return nil, status.Error(codes.Unauthenticated, "invalid auth token")
	}
	return ctx, nil
}
