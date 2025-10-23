# Context Utility
 
## Introduction
Context Utility provides functions to deal with context creation, callbacks.

## Structure

```go
	// DefaultTerminateSignals default signals to handle if no signals are provided
	DefaultTerminateSignals = []os.Signal{
		syscall.SIGTERM, syscall.SIGHUP, syscall.SIGINT,
		syscall.SIGQUIT,
	}
```

```go
// WithSignal return a context that is canceled if any of the specified signals was received
// if no signals are passed, default to DefaultTerminateSignals
func WithSignal(ctx context.Context, sig ...os.Signal) (context.Context, context.CancelFunc)


// OnDone registers a callback on a context when it's done
// The ctx.Err() is passed as is to the callback function
func OnDone(ctx context.Context, cb func(error)) 
```

## Usage

```go
	ctx, _ := utils.WithSignal(context.Background())
	utils.OnDone(ctx, func(_ error) {
		log.Info().Msg("shutting down")
	})
```
