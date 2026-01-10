package qdrant

import (
	"context"
	"fmt"

	"github.com/jllopis/kairos/pkg/memory"
	pb "github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Store struct {
	client      pb.PointsClient
	collections pb.CollectionsClient // Add collections client
}

func New(addr string) (*Store, error) {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("did not connect: %v", err)
	}

	return &Store{
		client:      pb.NewPointsClient(conn),
		collections: pb.NewCollectionsClient(conn),
	}, nil
}

func (s *Store) CreateCollection(ctx context.Context, name string, vectorSize uint64) error {
	_, err := s.collections.Create(ctx, &pb.CreateCollection{
		CollectionName: name,
		VectorsConfig: &pb.VectorsConfig{
			Config: &pb.VectorsConfig_Params{
				Params: &pb.VectorParams{
					Size:     vectorSize,
					Distance: pb.Distance_Cosine,
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}
	return nil
}

func (s *Store) Upsert(ctx context.Context, collection string, points []memory.Point) error {
	qPoints := make([]*pb.PointStruct, len(points))
	for i, p := range points {
		// Convert payload map[string]interface{} to map[string]*pb.Value
		payload := make(map[string]*pb.Value)
		for k, v := range p.Payload {
			// This is a simplified conversion. Real implementation needs robust type checking.
			switch val := v.(type) {
			case string:
				payload[k] = &pb.Value{Kind: &pb.Value_StringValue{StringValue: val}}
			case int:
				payload[k] = &pb.Value{Kind: &pb.Value_IntegerValue{IntegerValue: int64(val)}}
			case int64:
				payload[k] = &pb.Value{Kind: &pb.Value_IntegerValue{IntegerValue: val}}
			case float64:
				payload[k] = &pb.Value{Kind: &pb.Value_DoubleValue{DoubleValue: val}}
				// Add more types as needed
			}
		}

		qPoints[i] = &pb.PointStruct{
			Id: &pb.PointId{
				PointIdOptions: &pb.PointId_Uuid{Uuid: p.ID},
			},
			Vectors: &pb.Vectors{
				VectorsOptions: &pb.Vectors_Vector{
					Vector: &pb.Vector{Data: p.Vector},
				},
			},
			Payload: payload,
		}
	}

	_, err := s.client.Upsert(ctx, &pb.UpsertPoints{
		CollectionName: collection,
		Points:         qPoints,
	})
	if err != nil {
		return fmt.Errorf("failed to upsert points: %w", err)
	}

	return nil
}

func (s *Store) Search(ctx context.Context, collection string, vector []float32, limit int, scoreThreshold float32) ([]memory.SearchResult, error) {
	resp, err := s.client.Search(ctx, &pb.SearchPoints{
		CollectionName: collection,
		Vector:         vector,
		Limit:          uint64(limit),
		ScoreThreshold: &scoreThreshold,
		WithPayload:    &pb.WithPayloadSelector{SelectorOptions: &pb.WithPayloadSelector_Enable{Enable: true}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search points: %w", err)
	}

	results := make([]memory.SearchResult, len(resp.Result))
	for i, r := range resp.Result {
		// Convert back payload
		payload := make(map[string]interface{})
		for k, v := range r.Payload {
			if v.GetKind() != nil {
				// Simplified payload extraction
				switch knd := v.GetKind().(type) {
				case *pb.Value_StringValue:
					payload[k] = knd.StringValue
				case *pb.Value_IntegerValue:
					payload[k] = knd.IntegerValue
				case *pb.Value_DoubleValue:
					payload[k] = knd.DoubleValue
				}
			}
		}

		var id string
		if r.Id.GetUuid() != "" {
			id = r.Id.GetUuid()
		} else {
			// Handle uint64 IDs if necessary or just convert to string
			id = fmt.Sprintf("%d", r.Id.GetNum())
		}

		results[i] = memory.SearchResult{
			ID:    id,
			Score: r.Score,
			Point: memory.Point{
				ID:      id,
				Vector:  nil, // usually don't need vector back
				Payload: payload,
			},
		}
	}

	return results, nil
}
