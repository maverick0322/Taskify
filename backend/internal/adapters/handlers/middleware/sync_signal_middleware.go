package middleware

import (
	"net/http"
	"strings"

	"github.com/maverick0322/taskify/backend/internal/core/services"
)

type responseStatusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (recorder *responseStatusRecorder) WriteHeader(statusCode int) {
	recorder.statusCode = statusCode
	recorder.ResponseWriter.WriteHeader(statusCode)
}

func (recorder *responseStatusRecorder) StatusCode() int {
	if recorder.statusCode == 0 {
		return http.StatusOK
	}
	return recorder.statusCode
}

func NotifySyncOnSuccessfulMutation(signalBus *services.SyncSignalBus) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			recorder := &responseStatusRecorder{ResponseWriter: response}
			next.ServeHTTP(recorder, request)

			if signalBus == nil || !isSyncMutationRequest(request) || !isSuccessfulStatus(recorder.StatusCode()) {
				return
			}
			signalBus.Notify(services.SyncSignalLocalMutation)
		})
	}
}

func isSyncMutationRequest(request *http.Request) bool {
	switch request.Method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
	default:
		return false
	}

	path := request.URL.Path
	return !strings.HasPrefix(path, "/sync/") && !strings.HasPrefix(path, "/system/")
}

func isSuccessfulStatus(statusCode int) bool {
	return statusCode >= 200 && statusCode < 400
}
