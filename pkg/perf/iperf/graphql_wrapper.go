package iperf

import (
	"context"

	"github.com/threefoldtech/zosbase/pkg/perf/graphql"
)

// GraphQLClient interface for mocking GraphQL operations
type GraphQLClient interface {
	GetUpNodes(ctx context.Context, nodesNum int, farmID, excludeFarmID uint32, ipv4, ipv6 bool) ([]graphql.Node, error)
}
