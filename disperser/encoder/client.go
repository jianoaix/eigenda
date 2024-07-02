package encoder

import (
	"context"
	"fmt"
	"time"

	"github.com/Layr-Labs/eigenda/disperser"
	pb "github.com/Layr-Labs/eigenda/disperser/api/grpc/encoder"
	"github.com/Layr-Labs/eigenda/encoding"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func getServiceEndpoints(serviceName, namespace string) ([]string, error) {
	// Create the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	// Create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	// Get the endpoints
	endpoints, err := clientset.CoreV1().Endpoints(namespace).Get(context.TODO(), serviceName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	var addresses []string
	for _, subset := range endpoints.Subsets {
		for _, address := range subset.Addresses {
			addresses = append(addresses, address.IP)
		}
	}

	return addresses, nil
}

type client struct {
	addr    string
	timeout time.Duration
}

func NewEncoderClient(addr string, timeout time.Duration) (disperser.EncoderClient, error) {
	return client{
		addr:    addr,
		timeout: timeout,
	}, nil
}

func (c client) EncodeBlob(ctx context.Context, data []byte, encodingParams encoding.EncodingParams) (*encoding.BlobCommitments, []*encoding.Frame, error) {
	epts, err := getServiceEndpoints("encoder", "encoder")
	if err != nil {
		fmt.Println("XDEB service endpints err", err)
	} else {
		fmt.Println("XDEB service endpoints:", epts)
	}

	conn, err := grpc.Dial(
		c.addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024*1024*1024)), // 1 GiB
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to dial encoder: %w", err)
	}
	defer conn.Close()

	encoder := pb.NewEncoderClient(conn)
	reply, err := encoder.EncodeBlob(ctx, &pb.EncodeBlobRequest{
		Data: data,
		EncodingParams: &pb.EncodingParams{
			ChunkLength: uint32(encodingParams.ChunkLength),
			NumChunks:   uint32(encodingParams.NumChunks),
		},
	})
	if err != nil {
		return nil, nil, err
	}

	commitment, err := new(encoding.G1Commitment).Deserialize(reply.GetCommitment().GetCommitment())
	if err != nil {
		return nil, nil, err
	}
	lengthCommitment, err := new(encoding.G2Commitment).Deserialize(reply.GetCommitment().GetLengthCommitment())
	if err != nil {
		return nil, nil, err
	}
	lengthProof, err := new(encoding.LengthProof).Deserialize(reply.GetCommitment().GetLengthProof())
	if err != nil {
		return nil, nil, err
	}
	chunks := make([]*encoding.Frame, len(reply.GetChunks()))
	for i, chunk := range reply.GetChunks() {
		deserialized, err := new(encoding.Frame).Deserialize(chunk)
		if err != nil {
			return nil, nil, err
		}
		chunks[i] = deserialized
	}
	return &encoding.BlobCommitments{
		Commitment:       commitment,
		LengthCommitment: lengthCommitment,
		LengthProof:      lengthProof,
		Length:           uint(reply.GetCommitment().GetLength()),
	}, chunks, nil
}
