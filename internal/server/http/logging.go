package gophmarkthttpserver

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

type (
	responseData struct {
		statusCode int
		answerSize int
		answerBody string
	}

	loggingResponseWriter struct {
		http.ResponseWriter
		responseData *responseData
	}
)

func (w *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := w.ResponseWriter.Write(b)
	w.responseData.answerSize = size
	w.responseData.answerBody = string(b)
	return size, err
}

func (w *loggingResponseWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
	w.responseData.statusCode = statusCode
}

func Logging(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rID, ok := r.Context().Value(middleware.RequestIDKey).(string)
			if !ok {
				logger.Error("undefined request id")
			}

			method := r.Method
			uri := r.URL.Path
			cType := r.Header.Get("Content-Type")
			// read request body
			bodyBytes, _ := io.ReadAll(r.Body)
			requestBody := string(bodyBytes)
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

			lw := &loggingResponseWriter{
				ResponseWriter: w,
				responseData:   &responseData{},
			}

			next.ServeHTTP(lw, r)

			responseContentType := lw.Header()["Content-Type"][0]

			rDuration := time.Since(start).Nanoseconds()
			logger.Info(
				fmt.Sprintf("Request %v", rID),
				zap.String("Method", method),
				zap.String("URI", uri),
				zap.String("Content-Type", cType),       // for debug
				zap.String("Request Body", requestBody), // for debug
				zap.String("Duration", fmt.Sprintf("%d ns", rDuration)),
				zap.Int("Response Code", lw.responseData.statusCode),
				zap.Int("Response Length", lw.responseData.answerSize),
				zap.String("Response Content-Type", responseContentType),
				zap.String("Response Body", lw.responseData.answerBody), // for debug
			)
		})
	}
}
