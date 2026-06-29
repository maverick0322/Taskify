package middleware

import (
	"net/http"

	"github.com/maverick0322/taskify/backend/internal/core/services"
)

func NotifyRealtimeOnSuccessfulMutation(realtimeHub *services.UserRealtimeHub) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			recorder := &responseStatusRecorder{ResponseWriter: response}
			next.ServeHTTP(recorder, request)

			if realtimeHub == nil || !isSyncMutationRequest(request) || !isSuccessfulStatus(recorder.StatusCode()) {
				return
			}

			userID, ok := UserIDFromContext(request.Context())
			if !ok {
				return
			}

			realtimeHub.Publish(userID, services.RealtimeEvent{
				Type:   services.RealtimeSyncUpdateEvent,
				UserID: userID,
				Source: services.RealtimeSourceHTTPMutation,
			})
		})
	}
}
