package main
 
import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"
 
	pb "grpc-lesson/gen/search"
 
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func singleCrawl(ctx context.Context, client pb.CrawlServiceClient) {
	resp, err := client.Crawl(ctx, &pb.CrawlRequest{
		Url: "https://go.dev/doc/effective_go",
		Javascript: false,
		TimeoutMs: 5000,
	})
	if err != nil {
		slog.Error("Crawl failed", "error", err)
		return 
	}
	fmt.Printf("Url: %s\n", resp.Url)
	fmt.Printf("Status: %d\n", resp.StatusCode)
	fmt.Printf("Token count: %d\n", resp.TokenCount)
	fmt.Printf("Content: %s\n", resp.Markdown[:80])
}

func batchCrawl(ctx context.Context, client pb.CrawlServiceClient) {
	resp, err := client.BatchCrawl(ctx, &pb.BatchCrawlRequest{
		Requests: []*pb.CrawlRequest{
			{Url: "https://go.dev/tour", Javascript: false, TimeoutMs: 5000},
			{Url: "https://pkg.go.dev/net/http", Javascript: false, TimeoutMs: 5000},
			{Url: "https://grpc.io/docs", Javascript: true, TimeoutMs: 8000},
		},
	})
	if err != nil {
		slog.Error("batch crawl failed", "error", err)
		return
	}
	for _, result := range resp.Results {
		fmt.Printf("URL: %s — tokens: %d — status: %d\n", result.Url, result.TokenCount, result.StatusCode)
	}
}

// streamCrawl demonstrates server-side streaming.
// Results arrive one by one as each URL completes —
// we process each immediately rather than waiting for all three.
// This is how OpenSearch streams partial results to agents.
func streamCrawl(ctx context.Context, client pb.CrawlServiceClient) {
	stream, err := client.StreamCrawl(ctx, &pb.BatchCrawlRequest{
		Requests: []*pb.CrawlRequest{
			{Url: "https://searxng.org", Javascript: false, TimeoutMs: 5000},
			{Url: "https://spider.cloud", Javascript: true, TimeoutMs: 8000},
			{Url: "https://redis.io/docs", Javascript: false, TimeoutMs: 5000},
		},
	})
	if err != nil {
		slog.Error("stream crawl failed", "error", err)
		return
	}
	
	// Recv() blocks until the next result arrives from the server.
	// io.EOF means the server finished streaming — not an error.
	for {
		result, err := stream.Recv()
		if err == io.EOF {
			fmt.Println("stream complete")
			break
		}
		if err != nil {
			slog.Error("stream error", "error", err)
			break
		}
		// Process each result the moment it arrives
		fmt.Printf("received: %s — tokens: %d\n", result.Url, result.TokenCount)
	}

}

func main() {
	// Dial opens a connection to the gRPC server.
	// insecure.NewCredentials() disables TLS — fine for local development.
	// In production you use real TLS certificates.
	conn, err := grpc.NewClient("localhost: 50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		slog.Error("failed to connect", "error", err)
		return
	}
	defer conn.Close()

	// Create the client stub — this is the generated code that
	// gives you a Go function for every RPC method in the proto.
	client := pb.NewCrawlServiceClient(conn)

	// Context with timeout — every gRPC call needs one.
	// Without a timeout a slow server blocks your goroutine forever.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fmt.Println("=== Single Crawl ===")
	singleCrawl(ctx, client)
	
	fmt.Println("\n=== Batch Crawl ===")
	batchCrawl(ctx, client)

	fmt.Println("\n=== Stream Crawl ===")
	streamCrawl(ctx, client)
}