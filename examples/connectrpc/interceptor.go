package connectrpc

import (
	"context"
	"log/slog"
	"strconv"
	"sync/atomic"

	"connectrpc.com/connect"
	"github.com/emicklei/recall"
)

const requestIDHeader = "x-request-id"

var localRequestID atomic.Int64

func NewRecallInterceptor() connect.UnaryInterceptorFunc {
	interceptor := func(next connect.UnaryFunc) connect.UnaryFunc {
		return connect.UnaryFunc(func(
			ctx context.Context,
			req connect.AnyRequest,
		) (connect.AnyResponse, error) {
			requestID := req.Header().Get(requestIDHeader)
			if requestID == "" {
				requestID = strconv.FormatInt(localRequestID.Add(1), 10)
			}
			withRequestID := slog.Default().With(slog.String(requestIDHeader, requestID))
			recaller := recall.New(recall.ContextWithLogger(ctx, withRequestID))
			var resp connect.AnyResponse
			err := recaller.Call(func(callCtx context.Context) error {
				response, nextErr := next(callCtx, req)
				resp = response
				return nextErr
			})
			return resp, err
		})
	}
	return connect.UnaryInterceptorFunc(interceptor)
}
