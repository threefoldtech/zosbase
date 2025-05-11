package receiver

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zosbase/pkg/api"
	"github.com/threefoldtech/zosbase/pkg/environment"
	"github.com/threefoldtech/zosbase/pkg/network/namespace"
)

const (
	retryInterval = 100 * time.Millisecond

	// JSON-RPC error codes
	ErrCodeParseError     = -32700
	ErrCodeInvalidRequest = -32600
	ErrCodeMethodNotFound = -32601
	ErrCodeInvalidParams  = -32602
	ErrCodeInternalError  = -32000
)

type Receiver struct {
	api       *api.API
	handlers  map[string]HandlerFunc
	namespace string // network namespace where mycelium is running
	stopCh    chan struct{}
	wg        sync.WaitGroup
	running   bool
}

func NewReceiver(api *api.API, namespace string) *Receiver {
	receiver := &Receiver{
		api:       api,
		namespace: namespace,
		handlers:  make(map[string]HandlerFunc),
		stopCh:    make(chan struct{}),
	}

	receiver.registerHandlers()

	return receiver
}

func (r *Receiver) Run(ctx context.Context) error {
	log.Info().Str("namespace", r.namespace).Msg("starting receiver...")
	r.running = true

	r.wg.Add(1)
	defer r.wg.Done()

	if r.namespace == "" {
		r.receiveLoop(ctx)
		return nil
	}

	netns, err := namespace.GetByName(r.namespace)
	if err != nil {
		return fmt.Errorf("failed to get network namespace %s: %w", r.namespace, err)
	}
	defer netns.Close()

	return netns.Do(func(_ ns.NetNS) error {
		r.receiveLoop(ctx)
		return nil
	})
}

func (r *Receiver) Stop() {
	log.Info().Msg("stopping receiver...")
	r.wg.Wait()

	close(r.stopCh)
}

func (r *Receiver) receiveLoop(ctx context.Context) {
	for {
		select {
		case <-r.stopCh:
			return
		case <-ctx.Done():
			return
		default:
			if err := r.receiveAndProcessMessage(ctx); err != nil {
				log.Error().Err(err).Msg("error processing message")
				time.Sleep(retryInterval)
			}
		}
	}
}

func (r *Receiver) receiveAndProcessMessage(ctx context.Context) error {
	log.Debug().Msg("ready to receive messages...")
	cmd := exec.CommandContext(ctx, "mycelium", "message", "receive")
	raw, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to receive message: %w", err)
	}

	var message Message
	if err := json.Unmarshal(raw, &message); err != nil {
		return fmt.Errorf("failed to parse message: %w", err)
	}
	log.Debug().
		Str("src_ip", message.SrcIP).
		Str("message_id", message.ID).
		Msg("received message")

	response := r.processRequest(ctx, message)
	return r.sendResponse(message.SrcPK, message.ID, response)
}

func (r *Receiver) processRequest(ctx context.Context, message Message) JSONRPCResponse {
	response := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      message.ID,
	}

	var request JSONRPCRequest
	if err := json.Unmarshal([]byte(message.Payload), &request); err != nil {
		response.Error = &RPCError{
			Code:    ErrCodeParseError,
			Message: "Parse error",
			Data:    err.Error(),
		}
		return response
	}

	log.Debug().
		Str("method", request.Method).
		Msg("executing handler")

	if request.JSONRPC != "2.0" {
		response.Error = &RPCError{
			Code:    ErrCodeInvalidRequest,
			Message: "Invalid Request",
			Data:    "jsonrpc version must be 2.0",
		}
		return response
	}

	response.ID = request.ID

	handler, ok := r.handlers[request.Method]
	if !ok {
		response.Error = &RPCError{
			Code:    ErrCodeMethodNotFound,
			Message: "Method not found",
			Data:    request.Method,
		}
		return response
	}

	params := json.RawMessage("{}")
	if request.Params != nil {
		var err error
		if params, err = json.Marshal(request.Params); err != nil {
			response.Error = &RPCError{
				Code:    ErrCodeInvalidParams,
				Message: "Invalid params",
				Data:    err.Error(),
			}
			return response
		}
	}

	twin, err := getTwinFromPublicKey(ctx, message.SrcPK)
	if err != nil || twin == 0 {
		response.Error = &RPCError{
			Code:    ErrCodeInternalError,
			Message: "Failed to get twin from chain",
			Data:    err.Error(),
		}
		return response
	}
	ctx = context.WithValue(ctx, "twin_id", twin)

	result, err := handler(ctx, params)
	if err != nil {
		response.Error = &RPCError{
			Code:    ErrCodeInternalError,
			Message: err.Error(),
		}
		return response
	}

	response.Result = result
	return response
}

func (r *Receiver) sendResponse(recipient string, replyTo string, response JSONRPCResponse) error {
	responseBytes, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	args := []string{"message", "send", recipient, string(responseBytes)}
	if replyTo != "" {
		args = append(args, "--reply-to", replyTo)
	}

	cmd := exec.Command("mycelium", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to send response: %w, output: %s", err, string(output))
	}

	return nil
}

func getTwinFromPublicKey(ctx context.Context, publicKey string) (uint32, error) {
	// needs mycelium identity on the chain
	// to get the twin id from the public key
	manager, err := environment.GetSubstrate()
	if err != nil {
		return 0, fmt.Errorf("failed to get substrate manager: %w", err)
	}

	sub, err := manager.Substrate()
	if err != nil {
		return 0, fmt.Errorf("failed to get substrate: %w", err)
	}

	return sub.GetTwinByPubKey([]byte(publicKey))
}
