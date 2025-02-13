## recall

[![GoDoc](https://pkg.go.dev/badge/github.com/emicklei/recall)](https://pkg.go.dev/github.com/emicklei/recall)

In service-oriented architectures, request processing failures often require detailed logging for effective troubleshooting.
Ideally, log records should contain comprehensive context.
It can be beneficial to also have access to all debug-level logs **leading up to the point** of failure.
However, enabling debug logging in production environments can result in excessive log volumes and increased CPU utilization.

So, you ideally want debug logging information only when a failure occurs. 
In general, a failure situation is an exceptional occurrence, so collecting information in that case is not overly costly.

The `recall` package builds on this idea by encapsulating a function that can return an error. 
The following strategies are available to capture debug logging:

#### RecallOnErrorStrategy

If an error is detected, a Recaller will call that same function again, but this time with a different logger configured to capture all debug logging. 
This strategy requires that your function has no side-effects ; idempotent.
This is the default strategy.

#### RecordingStrategy

Debug logging is recorded by the Recaller directly and only if an error is detected, the log records are replayed from memory using the default logger. 
This strategy can result in a higher memory consumption (and GC time) because all Debug records are recorded on every function call. 
The function is not called a second time so no idempotency in processing is required.

### Usage

	recaller := recall.New(context.Background())

	err := recaller.Call(func(ctx context.Context) error {
		rlog := recall.LoggerFromContext(ctx)
		
		rlog.Info("begin")
		rlog.Debug("this will show up on error")
		return errors.New("something went wrong")
	})

will output

    2025/02/11 18:55:23 INFO begin
    2025/02/11 18:55:23 INFO [RECALL] begin
    2025/02/11 18:55:23 INFO [RECALL] this will show up on error
    2025/02/11 18:55:23 ERROR bummer err="something went wrong"

See [examples](https://github.com/emicklei/recall/tree/main/examples) for other usage.

A different approach in both capturing and visualising logging is the [Nanny](https://github.com/emicklei/nanny) package.

(c) 2025, https://ernestmicklei.com. MIT License.
