package iperf

import (
	"context"

	"github.com/threefoldtech/zosbase/pkg/perf/graphql"
)

// GraphQLClient interface for mocking GraphQL operations
type GraphQLClient interface {
	GetUpNodes(ctx context.Context, nodesNum int, farmID, excludeFarmID uint32, ipv4, ipv6 bool) ([]graphql.Node, error)
}

// GraphQLClientWrapper wraps the real GraphQL client to implement the interface
type GraphQLClientWrapper struct {
	client graphql.GraphQl
}

// NewGraphQLClientWrapper creates a new wrapper for the GraphQL client
func NewGraphQLClientWrapper(client graphql.GraphQl) GraphQLClient {
	return &GraphQLClientWrapper{client: client}
}

// GetUpNodes calls the underlying GraphQL client's GetUpNodes method
func (g *GraphQLClientWrapper) GetUpNodes(ctx context.Context, nodesNum int, farmID, excludeFarmID uint32, ipv4, ipv6 bool) ([]graphql.Node, error) {
	return g.client.GetUpNodes(ctx, nodesNum, farmID, excludeFarmID, ipv4, ipv6)
}
