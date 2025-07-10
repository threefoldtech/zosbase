package graphql

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	realEndpoint = "https://graphql.dev.threefold.me/graphql"
)

func mockGraphQLServer(t *testing.T, statusCode int, responseBody string) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		fmt.Fprintln(w, responseBody)
	}))
	t.Cleanup(server.Close)
	return server
}

func TestGetUpNodes_Mock(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		body := make([]byte, r.ContentLength)
		r.Body.Read(body)
		requestBody := string(body)

		var response string
		if strings.Contains(requestBody, "nodesConnection") {
			// This is the count query
			response = `{
				"data": {
					"items": {
						"count": 10
					}
				}
			}`
		} else {
			// This is the nodes query
			response = `{
				"data": {
					"nodes": [
						{
							"nodeID": 1,
							"publicConfig": {
								"ipv4": "192.168.1.1",
								"ipv6": "2001:db8::1"
							}
						},
						{
							"nodeID": 2,
							"publicConfig": {
								"ipv4": "192.168.1.2",
								"ipv6": "2001:db8::2"
							}
						}
					]
				}
			}`
		}

		fmt.Fprintln(w, response)
	}))
	t.Cleanup(server.Close)

	t.Run("basic query", func(t *testing.T) {
		gql, err := NewGraphQl(server.URL)
		require.NoError(t, err)

		ctx := context.Background()
		nodes, err := gql.GetUpNodes(ctx, 0, 0, 0, false, false)

		require.NoError(t, err)
		require.Len(t, nodes, 2)
		assert.Equal(t, uint32(1), nodes[0].NodeID)
		assert.Equal(t, "192.168.1.1", nodes[0].PublicConfig.Ipv4)
		assert.Equal(t, "2001:db8::1", nodes[0].PublicConfig.Ipv6)
	})

	failServer := mockGraphQLServer(t, http.StatusInternalServerError, `{"errors":[{"message":"internal error"}]}`)

	t.Run("server error", func(t *testing.T) {
		gql, err := NewGraphQl(failServer.URL)
		require.NoError(t, err)

		ctx := context.Background()
		_, err = gql.GetUpNodes(ctx, 0, 0, 0, false, false)
		require.Error(t, err)
	})

	t.Run("fallback behavior", func(t *testing.T) {
		gql, err := NewGraphQl(failServer.URL, server.URL)
		require.NoError(t, err)

		ctx := context.Background()
		nodes, err := gql.GetUpNodes(ctx, 0, 0, 0, false, false)
		require.NoError(t, err)
		require.Len(t, nodes, 2)
	})
}

func TestGetUpNodes_WithFilters(t *testing.T) {

	gql, err := NewGraphQl(realEndpoint)
	require.NoError(t, err)
	ctx := context.Background()

	t.Run("with node limit", func(t *testing.T) {
		nodes, err := gql.GetUpNodes(ctx, 5, 0, 0, false, false)
		require.NoError(t, err)

		assert.LessOrEqual(t, len(nodes), 5)
	})

	t.Run("with ipv4 filter", func(t *testing.T) {
		nodes, err := gql.GetUpNodes(ctx, 0, 0, 0, true, false)
		require.NoError(t, err)

		for _, node := range nodes {
			assert.NotEmpty(t, node.PublicConfig.Ipv4)
		}
	})

	t.Run("with ipv6 filter", func(t *testing.T) {
		nodes, err := gql.GetUpNodes(ctx, 0, 0, 0, false, true)
		require.NoError(t, err)

		for _, node := range nodes {
			assert.NotEmpty(t, node.PublicConfig.Ipv6)
		}
	})
}

func TestGetItemTotalCount(t *testing.T) {
	successResponse := `{
		"data": {
			"items": {
				"count": 42
			}
		}
	}`

	server := mockGraphQLServer(t, http.StatusOK, successResponse)

	t.Run("get count", func(t *testing.T) {
		gql, err := NewGraphQl(server.URL)
		require.NoError(t, err)

		ctx := context.Background()
		count, err := gql.getItemTotalCount(ctx, "nodes", "where: {}")

		require.NoError(t, err)
		assert.Equal(t, 42, count)
	})
}

// TestExec tests the exec method directly
func TestExec(t *testing.T) {
	successResponse := `{
		"data": {
			"test": "success"
		}
	}`

	failureResponse := `{
		"errors": [
			{"message": "something went wrong"}
		]
	}`

	successServer := mockGraphQLServer(t, http.StatusOK, successResponse)
	failureServer := mockGraphQLServer(t, http.StatusInternalServerError, failureResponse)

	t.Run("success on first url", func(t *testing.T) {
		gql, err := NewGraphQl(successServer.URL)
		require.NoError(t, err)

		result := struct {
			Test string
		}{}

		err = gql.exec(context.Background(), "query { test }", &result, nil)
		require.NoError(t, err)
		assert.Equal(t, "success", result.Test)
	})

	t.Run("fallback to second url", func(t *testing.T) {
		gql, err := NewGraphQl(failureServer.URL, successServer.URL)
		require.NoError(t, err)

		result := struct {
			Test string
		}{}

		err = gql.exec(context.Background(), "query { test }", &result, nil)
		require.NoError(t, err)
		assert.Equal(t, "success", result.Test)
	})

	t.Run("all urls fail", func(t *testing.T) {
		gql, err := NewGraphQl(failureServer.URL, failureServer.URL)
		require.NoError(t, err)

		result := struct {
			Test string
		}{}

		err = gql.exec(context.Background(), "query { test }", &result, nil)
		require.Error(t, err)
	})
}



func TestIntegration_NodeFiltering(t *testing.T) {

	gql, err := NewGraphQl(realEndpoint)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("nodes with IPv4 only", func(t *testing.T) {
		nodes, err := gql.GetUpNodes(ctx, 20, 0, 0, true, false)
		require.NoError(t, err)
		for i, node := range nodes {
			assert.NotEmpty(t, node.PublicConfig.Ipv4, "Node %d should have IPv4", i)
		}
	})

	t.Run("nodes with IPv6 only", func(t *testing.T) {
		nodes, err := gql.GetUpNodes(ctx, 20, 0, 0, false, true)
		require.NoError(t, err)
		for i, node := range nodes {
			assert.NotEmpty(t, node.PublicConfig.Ipv6, "Node %d should have IPv6", i)
		}
	})

	t.Run("nodes with both IPv4 and IPv6", func(t *testing.T) {
		nodes, err := gql.GetUpNodes(ctx, 10, 0, 0, true, true)
		require.NoError(t, err)
		for i, node := range nodes {
			assert.NotEmpty(t, node.PublicConfig.Ipv4, "Node %d should have IPv4", i)
			assert.NotEmpty(t, node.PublicConfig.Ipv6, "Node %d should have IPv6", i)
		}
	})
}


func TestIntegration_PaginationAndLimits(t *testing.T) {

	gql, err := NewGraphQl(realEndpoint)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("different_limits", func(t *testing.T) {
		limits := []int{1, 5, 10, 20, 50}

		for _, limit := range limits {
			nodes, err := gql.GetUpNodes(ctx, limit, 0, 0, false, false)
			require.NoError(t, err)
			assert.LessOrEqual(t, len(nodes), limit,
				"Returned %d nodes but limit was %d", len(nodes), limit)
		}
	})
}

func TestIntegration_ErrorHandling(t *testing.T) {

	t.Run("invalid_endpoint", func(t *testing.T) {
		gql, err := NewGraphQl("https://invalid-endpoint-that-does-not-exist.com/graphql")
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err = gql.GetUpNodes(ctx, 5, 0, 0, false, false)
		require.Error(t, err)
	})

	t.Run("timeout_handling", func(t *testing.T) {
		gql, err := NewGraphQl(realEndpoint)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Microsecond)
		defer cancel()

		_, err = gql.GetUpNodes(ctx, 5, 0, 0, false, false)
		require.Error(t, err)
	})

	t.Run("fallback_to_valid_endpoint", func(t *testing.T) {
		gql, err := NewGraphQl(
			"https://invalid-endpoint.com/graphql",
			realEndpoint,
		)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		nodes, err := gql.GetUpNodes(ctx, 5, 0, 0, false, false)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(nodes), 0)
	})
}

func TestIntegration_DataValidation(t *testing.T) {

	gql, err := NewGraphQl(realEndpoint)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	nodes, err := gql.GetUpNodes(ctx, 20, 0, 0, false, false)
	require.NoError(t, err)

	t.Run("node_id_validation", func(t *testing.T) {
		nodeIDs := make(map[uint32]bool)

		for i, node := range nodes {
			assert.Greater(t, node.NodeID, uint32(0), "Node %d has invalid ID", i)

			assert.False(t, nodeIDs[node.NodeID], "Duplicate node ID %d found", node.NodeID)
			nodeIDs[node.NodeID] = true
		}

	})

	t.Run("public_config_validation", func(t *testing.T) {
		var ipv4Count, ipv6Count int

		for i, node := range nodes {
			hasIPv4 := node.PublicConfig.Ipv4 != ""
			hasIPv6 := node.PublicConfig.Ipv6 != ""

			if hasIPv4 {
				ipv4Count++
				assert.Contains(t, node.PublicConfig.Ipv4, ".",
					"Node %d IPv4 doesn't look like an IP: %s", i, node.PublicConfig.Ipv4)
			}

			if hasIPv6 {
				ipv6Count++
				assert.Contains(t, node.PublicConfig.Ipv6, ":",
					"Node %d IPv6 doesn't look like an IP: %s", i, node.PublicConfig.Ipv6)
			}
		}

	})
}
