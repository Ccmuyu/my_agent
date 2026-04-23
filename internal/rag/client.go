package rag

import (
	"context"
	"fmt"
	"time"

	pb "github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"github.com/Ccmuyu/my_agent/internal/config"
)

type VectorStore interface {
	Init(ctx context.Context) error
	Insert(ctx context.Context, vectors [][]float64, payloads []map[string]any) error
	Search(ctx context.Context, query []float64, topK int) ([]VectorSearchResult, error)
	Delete(ctx context.Context, ids []string) error
	Close() error
}

type VectorSearchResult struct {
	ID      string
	Score   float64
	Payload map[string]any
}

type QdrantClient struct {
	conn        *grpc.ClientConn
	collections pb.CollectionsClient
	points      pb.PointsClient
	collection  string
	vectorSize  int
}

func NewQdrantClient(cfg *config.VectorDBConfig, vectorSize int) (VectorStore, error) {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	conn, err := grpc.DialContext(context.Background(), addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to qdrant: %w", err)
	}

	return &QdrantClient{
		conn:        conn,
		collections: pb.NewCollectionsClient(conn),
		points:      pb.NewPointsClient(conn),
		collection:  cfg.Collection,
		vectorSize:  vectorSize,
	}, nil
}

func (q *QdrantClient) Init(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	exists, err := q.collections.CollectionExists(ctx, &pb.CollectionExistsRequest{
		CollectionName: q.collection,
	})
	if err != nil {
		return fmt.Errorf("failed to check collection: %w", err)
	}

	if exists.GetResult().GetExists() {
		return nil
	}

	_, err = q.collections.Create(ctx, &pb.CreateCollection{
		CollectionName: q.collection,
		VectorsConfig: &pb.VectorsConfig{
			Config: &pb.VectorsConfig_Params{
				Params: &pb.VectorParams{
					Size:     uint64(q.vectorSize),
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

func (q *QdrantClient) Insert(ctx context.Context, vectors [][]float64, payloads []map[string]any) error {
	if len(vectors) != len(payloads) {
		return fmt.Errorf("vectors and payloads length mismatch")
	}

	points := make([]*pb.PointStruct, len(vectors))
	for i := range vectors {
		points[i] = &pb.PointStruct{
			Id: &pb.PointId{
				PointIdOptions: &pb.PointId_Num{Num: uint64(time.Now().UnixNano() + int64(i))},
			},
			Vectors: &pb.Vectors{
				VectorsOptions: &pb.Vectors_Vector{
					Vector: &pb.Vector{Data: toFloat32Slice(vectors[i])},
				},
			},
			Payload: q.payloadToProto(payloads[i]),
		}
	}

	wait := true
	_, err := q.points.Upsert(ctx, &pb.UpsertPoints{
		CollectionName: q.collection,
		Points:         points,
		Wait:           &wait,
	})
	if err != nil {
		return fmt.Errorf("failed to upsert points: %w", err)
	}

	return nil
}

func (q *QdrantClient) Search(ctx context.Context, query []float64, topK int) ([]VectorSearchResult, error) {
	queryF32 := toFloat32Slice(query)
	searchResult, err := q.points.Search(ctx, &pb.SearchPoints{
		CollectionName: q.collection,
		Vector:         queryF32,
		Limit:          uint64(topK),
		WithPayload:    &pb.WithPayloadSelector{SelectorOptions: &pb.WithPayloadSelector_Enable{Enable: true}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	results := make([]VectorSearchResult, len(searchResult.Result))
	for i, r := range searchResult.Result {
		pointID := fmt.Sprintf("%d", r.GetId().GetNum())

		results[i] = VectorSearchResult{
			ID:      pointID,
			Score:   float64(r.Score),
			Payload: q.payloadFromProto(r.Payload),
		}
	}

	return results, nil
}

func (q *QdrantClient) Delete(ctx context.Context, ids []string) error {
	pointIDs := make([]*pb.PointId, len(ids))
	for i := range ids {
		pointIDs[i] = &pb.PointId{PointIdOptions: &pb.PointId_Num{Num: uint64(i + 1)}}
	}

	wait := true
	_, err := q.points.Delete(ctx, &pb.DeletePoints{
		CollectionName: q.collection,
		Points:         &pb.PointsSelector{PointsSelectorOneOf: &pb.PointsSelector_Points{Points: &pb.PointsIdsList{Ids: pointIDs}}},
		Wait:           &wait,
	})
	if err != nil {
		return fmt.Errorf("failed to delete points: %w", err)
	}

	return nil
}

func (q *QdrantClient) Close() error {
	return q.conn.Close()
}

func (q *QdrantClient) payloadToProto(payload map[string]any) map[string]*pb.Value {
	if payload == nil {
		return nil
	}

	fields := make(map[string]*pb.Value)
	for k, v := range payload {
		fields[k] = &pb.Value{Kind: &pb.Value_StringValue{StringValue: fmt.Sprintf("%v", v)}}
	}
	return fields
}

func (q *QdrantClient) payloadFromProto(payload map[string]*pb.Value) map[string]any {
	if payload == nil {
		return nil
	}
	result := make(map[string]any)
	for k, v := range payload {
		if v != nil && v.Kind != nil {
			switch val := v.Kind.(type) {
			case *pb.Value_StringValue:
				result[k] = val.StringValue
			case *pb.Value_IntegerValue:
				result[k] = val.IntegerValue
			case *pb.Value_DoubleValue:
				result[k] = val.DoubleValue
			case *pb.Value_BoolValue:
				result[k] = val.BoolValue
			default:
				result[k] = fmt.Sprintf("%v", v.Kind)
			}
		}
	}
	return result
}

func toFloat32Slice(f64 []float64) []float32 {
	f32 := make([]float32, len(f64))
	for i, v := range f64 {
		f32[i] = float32(v)
	}
	return f32
}