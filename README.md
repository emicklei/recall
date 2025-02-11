## recall

[![GoDoc](https://pkg.go.dev/badge/github.com/emicklei/recall)](https://pkg.go.dev/github.com/emicklei/recall)

In service-oriented architectures, request processing failures often require detailed logging for effective troubleshooting.
Ideally, log records should contain comprehensive context.
It can be beneficial to also have access to all debug-level logs **leading up to the point** of failure.
However, enabling debug logging in production environments can result in excessive log volumes and increased CPU utilization.

So, you ideally want debug logging information only when a failure occurs. A failure situation is an exceptional occurrence, so it is assumed that reprocessing the same request for the purpose of collecting information is not overly costly. Additionally, it is assumed that processing the same request leads to the same failure.

The `recall` package builds on this idea by encapsulating a function that can return an error. If an error is detected, a Recaller will call that same function again, but this time with a different logger configured to capture all debug logging.

### Usage

	recaller := recall.New(context.Background())

	err := recaller.Call(func(ctx context.Context) error {
		slog := recall.LoggerFromContext(ctx)
		slog.Info("begin")
		slog.Debug("this will show up on error")
		return errors.New("something went wrong")
	})

will output

    2025/02/11 18:55:23 INFO begin
    2025/02/11 18:55:23 INFO [RECALL] begin
    2025/02/11 18:55:23 INFO [RECALL] this will show up on error
    2025/02/11 18:55:23 ERROR bummer err="something went wrong"

See [examples](https://github.com/emicklei/recall/tree/main/examples) for other usage.

(c) 2025, https://ernestmicklei.com. MIT License.
