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
			recaller := recall.New(recall.ContextWithLogger(ctx, withRequestID)).WithErrorFilter(func(err error) bool {
				// e.g. not found and bad argument must not be recalled
				switch connect.CodeOf(err) {
				case connect.CodeNotFound, connect.CodeInvalidArgument, connect.CodeUnauthenticated:
					return false
				}
				return true
			})
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
