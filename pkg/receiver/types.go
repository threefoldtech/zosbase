package receiver

import (
	"context"
	"encoding/json"
)

type Message struct {
	ID      string `json:"id"`
	SrcIP   string `json:"srcIp"`
	SrcPK   string `json:"srcPk"`
	DstIP   string `json:"dstIp"`
	DstPK   string `json:"dstPk"`
	Topic   string `json:"topic"`
	Payload string `json:"payload"`
}

type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
	ID      string      `json:"id"`
}

type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
	ID      string      `json:"id"`
}

type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type HandlerFunc func(ctx context.Context, params json.RawMessage) (interface{}, error)
