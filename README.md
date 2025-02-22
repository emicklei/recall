## recall

[![Go](https://github.com/emicklei/recall/actions/workflows/go.yaml/badge.svg)](https://github.com/emicklei/recall/actions/workflows/go.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/emicklei/recall)](https://goreportcard.com/report/github.com/emicklei/recall)
[![GoDoc](https://pkg.go.dev/badge/github.com/emicklei/recall)](https://pkg.go.dev/github.com/emicklei/recall)
[![codecov](https://codecov.io/gh/emicklei/recall/branch/main/graph/badge.svg)](https://codecov.io/gh/emicklei/recall)

In service-oriented architectures, request processing failures often require detailed logging for effective troubleshooting.
Ideally, log records should contain comprehensive context.
It can be beneficial to also have access to all debug-level logs **leading up to the point** of failure.
However, enabling debug logging in production environments can result in excessive log volumes and increased CPU utilization.

So, you ideally want debug logging information only when a failure occurs. 
In general, a failure situation is an exceptional occurrence, so collecting information in that case is not overly costly.

The `recall` package builds on this idea by encapsulating a function that can return an error. 
The following strategies are available to capture debug (slog) logging:

#### RecallOnErrorStrategy

If an error is detected, a Recaller will call that same function again, but this time with a different logger configured to capture all debug logging. 
This strategy requires that your function has no side-effects ; idempotency.
This is the default strategy.

#### RecordingStrategy

Debug logging is recorded by the Recaller directly and only if an error is detected, the log records are replayed from memory using the default logger. 
This strategy can result in a higher memory consumption (and GC time) because all Debug records are recorded on every function call. 
The function is not called a second time so no idempotency in processing is required.

### Usage

	recaller := recall.New(context.Background())

	err := recaller.Call(func(ctx context.Context) error {
		rlog := recall.Slog(ctx)
		
		rlog.Info("begin")
		rlog.Debug("this will show up on error")
		return errors.New("something went wrong")
	})
	
	slog.Error("bummer", "err", err)
	
will output

    2025/02/11 18:55:23 INFO begin
    2025/02/11 18:55:23 INFO [RECALL] begin
    2025/02/11 18:55:23 INFO [RECALL] this will show up on error
    2025/02/11 18:55:23 ERROR bummer err="something went wrong"

This example uses RecallOnErrorStrategy by default.

See [examples](https://github.com/emicklei/recall/tree/main/examples) for other usage.

### Panic

By default, a Recaller will recover from a panic and writes an Error message with stack information, before returning an error with the panic message. You can disable panic recovery using `WithPanicRecovery(false)`.

### Not all errors are equal

If your function can return an error for which it makes no sense to retry it then you can set a `filter` function to check the error before applying the strategy. Use the `WithErrorFilter(...)` to set the function for the Recaller.

### Other work

A different approach in both capturing and visualising logging is offered by the [Nanny](https://github.com/emicklei/nanny) package.

&copy; 2025, https://ernestmicklei.com. MIT License.
