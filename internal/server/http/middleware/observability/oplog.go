package observability

import (
	"bytes"
	"encoding/json"
	"io"
	"net/url"
	"strings"
	"time"

	"go-apiadmin/internal/mq/kafka"

	"github.com/gin-gonic/gin"
)

var skipOpLogPaths = map[string]struct{}{
	"/healthz": {},
	"/readyz":  {},
	"/metrics": {},
}

var sensitiveKeys = []string{"password", "passwd", "pwd", "new_password", "old_password", "token", "authorization"}

// OperationLog 迁移到 observability 包
func OperationLog(p *kafka.Producer) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 跳过无需记录的路径
		rawPath := c.Request.URL.Path
		if _, ok := skipOpLogPaths[rawPath]; ok {
			c.Next()
			return
		}
		start := time.Now()
		var bodyBytes []byte
		if c.Request.Body != nil {
			b, _ := io.ReadAll(io.LimitReader(c.Request.Body, 4096))
			bodyBytes = b
			c.Request.Body = io.NopCloser(bytes.NewBuffer(b))
		}
		bw := &bodyWriter{ResponseWriter: c.Writer}
		c.Writer = bw
		c.Next()
		queryStr := c.Request.URL.RawQuery
		if len(queryStr) > 1024 {
			queryStr = queryStr[:1024]
		}
		ua := c.Request.UserAgent()
		if len(ua) > 256 {
			ua = ua[:256]
		}
		ref := c.Request.Referer()
		if len(ref) > 512 {
			ref = ref[:512]
		}
		path := c.FullPath()
		if path == "" {
			path = rawPath
		}
		if path != "" && !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		var qDecoded string
		if queryStr != "" {
			if vals, err := url.ParseQuery(queryStr); err == nil {
				pairs := make([]string, 0, len(vals))
				for k, v := range vals {
					if len(v) > 0 {
						val := v[0]
						if len(val) > 100 {
							val = val[:100]
						}
						pairs = append(pairs, k+"="+val)
					}
				}
				qDecoded = strings.Join(pairs, "&")
			}
			if len(qDecoded) > 512 {
				qDecoded = qDecoded[:512]
			}
		}
		sanitizedBody := sanitizeJSON(bodyBytes)
		respBody := truncateString(bw.buf.String(), 4096)
		actionName := deriveActionName(path, c.Request.Method)
		e := map[string]interface{}{
			"action_name": actionName,
			"path":        path,
			"method":      c.Request.Method,
			"status":      c.Writer.Status(),
			"latency_ms":  time.Since(start).Milliseconds(),
			"ip":          c.ClientIP(),
			"user_id":     c.GetInt64("user_id"),
			"time":        time.Now().Format(time.RFC3339),
			"body":        sanitizedBody,
			"query":       qDecoded,
			"ua":          ua,
			"referer":     ref,
			"resp_size":   bw.buf.Len(),
			"resp_body":   respBody,
		}
		if len(c.Errors) > 0 {
			errs := make([]string, 0, len(c.Errors))
			for _, er := range c.Errors {
				errs = append(errs, er.Error())
			}
			e["errors"] = errs
		}
		b, _ := json.Marshal(e)
		if traceID, ok := c.Get("trace_id"); ok {
			_ = p.SendWithHeaders(c.Request.Context(), nil, b, map[string]string{"trace_id": traceID.(string)})
		} else {
			_ = p.Send(c.Request.Context(), nil, b)
		}
	}
}

type bodyWriter struct {
	gin.ResponseWriter
	buf bytes.Buffer
}

func (w *bodyWriter) Write(b []byte) (int, error) {
	if w.buf.Len() < 4096 {
		remain := 4096 - w.buf.Len()
		if len(b) > remain {
			w.buf.Write(b[:remain])
		} else {
			w.buf.Write(b)
		}
	}
	return w.ResponseWriter.Write(b)
}

func sanitizeJSON(src []byte) string {
	if len(src) == 0 {
		return ""
	}
	if len(src) > 4096 {
		src = src[:4096]
	}
	var m interface{}
	if json.Unmarshal(src, &m) != nil {
		return string(src)
	}
	sanitizeValue(&m)
	b, err := json.Marshal(m)
	if err != nil {
		return string(src)
	}
	return string(b)
}

func sanitizeValue(v *interface{}) {
	switch val := (*v).(type) {
	case map[string]interface{}:
		for k, vv := range val {
			lower := strings.ToLower(k)
			for _, s := range sensitiveKeys {
				if lower == s {
					val[k] = "***"
					goto NEXT
				}
			}
			sanitizeValue(&vv)
			val[k] = vv
		NEXT:
		}
	case []interface{}:
		for i, elem := range val {
			sanitizeValue(&elem)
			val[i] = elem
		}
	}
}

func truncateString(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}

func deriveActionName(path, method string) string {
	if path == "" {
		return method
	}
	p := strings.Trim(path, "/")
	if p == "" {
		return method
	}
	p = strings.ReplaceAll(p, "/", ":")
	p = strings.ReplaceAll(p, ":", "_")
	return strings.ToLower(method + "_" + p)
}
