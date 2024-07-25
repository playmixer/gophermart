package rest

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var (
	errUnauthorize = errors.New("unauthorize")
)

func (s *Server) Authentication() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := s.checkAuth(c)
		if err != nil {
			if !errors.Is(err, errUnauthorize) {
				c.Writer.WriteHeader(http.StatusInternalServerError)
			} else {
				c.Writer.WriteHeader(http.StatusUnauthorized)
			}
			c.Abort()
		}

		if err == nil && userID == 0 {
			c.Writer.WriteHeader(http.StatusUnauthorized)
			c.Abort()
		}

		c.Next()
	}
}

func (s *Server) Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		s.log.Info(
			"Request",
			zap.String("uri", c.Request.RequestURI),
			zap.Duration("duration", time.Since(start)),
			zap.String("method", c.Request.Method),
			zap.Int("status", c.Writer.Status()),
			zap.Int("size", c.Writer.Size()),
		)
	}
}

func (s *Server) GzipDecompress() gin.HandlerFunc {
	return func(c *gin.Context) {
		if ok := strings.Contains(c.Request.Header.Get("Content-Encoding"), "gzip"); ok {
			gr, err := NewGzipReader(c.Request.Body)
			if err != nil {
				c.Writer.WriteHeader(http.StatusBadRequest)
				c.Abort()
				return
			}
			c.Request.Body = gr
			defer func() {
				if err := gr.Close(); err != nil {
					s.log.Info("failed close gzip reader", zap.Error(err))
				}
			}()
		}
		c.Next()
	}
}

func (s *Server) GzipCompress() gin.HandlerFunc {
	return func(c *gin.Context) {
		if ok := strings.Contains(c.Request.Header.Get("Accept-Encoding"), "gzip"); ok {
			c.Writer.Header().Set("Content-Encoding", "gzip")

			cw := NewGzipWriter(c)
			c.Writer = cw

			defer func() {
				if err := cw.writer.Close(); err != nil {
					s.log.Info("failed close gzip writer", zap.Error(err))
				}
			}()
		}
		c.Next()
	}
}
