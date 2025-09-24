package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

func RateLimiter(client *redis.Client, keyPrefix string, limit int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// In a real app behind a proxy, you'd use r.Header.Get("X-Forwarded-For") or similar
			ip := r.RemoteAddr
			key := fmt.Sprintf("%s:%s", keyPrefix, ip)

			ctx := context.Background()

			// Use a pipeline to execute commands atomically and reduce round-trips
			pipe := client.Pipeline()
			countCmd := pipe.Incr(ctx, key)
			pipe.Expire(ctx, key, window)
			_, err := pipe.Exec(ctx)

			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			count := countCmd.Val()
			if count > int64(limit) {
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}

			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(limit-int(count)))

			next.ServeHTTP(w, r)
		})
	}
}
```
