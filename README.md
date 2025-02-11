## recall

In service-oriented architectures, request processing failures often require detailed logging for effective troubleshooting.
Ideally, log records should contain comprehensive context.
It can be beneficial to have access to all debug-level logs leading up to the point of failure.
However, enabling debug logging in production environments can result in excessive log volumes and increased CPU utilization.

So, you ideally want debug logging information only when a failure occurs. A failure situation is an exceptional occurrence, so it is assumed that reprocessing the same request for the purpose of collecting information is not overly costly. Additionally, it is assumed that processing the same request leads to the same failure.

The `recall` package builds on this idea by encapsulating a function that can return an error. If an error is detected, a Recaller will call that same function again, but this time with a different logger configured to capture all debug logging.
