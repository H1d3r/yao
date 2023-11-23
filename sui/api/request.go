package api

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/yao/sui/core"
)

// Request is the request for the page API.
type Request struct {
	File string
	*core.Request
}

// NewRequestContext is the constructor for Request.
func NewRequestContext(c *gin.Context) (*Request, int, error) {

	file, params, err := parserPath(c)
	if err != nil {
		return nil, 404, err
	}

	payload, body, err := payload(c)
	if err != nil {
		return nil, 500, err
	}

	return &Request{
		File: file,
		Request: &core.Request{
			Method:  c.Request.Method,
			Query:   c.Request.URL.Query(),
			Body:    body,
			Payload: payload,
			Referer: c.Request.Referer(),
			Headers: c.Request.Header,
			Params:  params,
		},
	}, 200, nil
}

// Render is the response for the page API.
func (r *Request) Render() (string, int, error) {
	return r.File, 200, nil
}

func parserPath(c *gin.Context) (string, map[string]string, error) {

	params := map[string]string{}

	parts := strings.Split(strings.TrimSuffix(c.Request.URL.Path, ".sui"), "/")[1:]
	if len(parts) < 1 {
		return "", nil, fmt.Errorf("no route matchers")
	}

	fileParts := []string{application.App.Root(), "public"}

	// Match the sui
	matchers := core.RouteExactMatchers[parts[0]]
	if matchers == nil {
		for matcher, reMatchers := range core.RouteMatchers {
			matched := matcher.FindStringSubmatch(parts[0])
			if len(matched) > 0 {
				matchers = reMatchers
				fileParts = append(fileParts, matched[0])
				break
			}
		}
	}

	if matchers == nil {
		return "", nil, fmt.Errorf("no route matchers")
	}

	// Match the page parts
	for i, part := range parts[1:] {
		if len(matchers) < i+1 {
			return "", nil, fmt.Errorf("no route matchers")
		}

		parent := ""
		if i > 0 {
			parent = parts[i]
		}
		matched := false
		for _, matcher := range matchers[i] {

			// Filter the parent
			if matcher.Parent != "" && matcher.Parent != parent {
				continue
			}

			if matcher.Exact == part {
				fileParts = append(fileParts, matcher.Exact)
				matched = true
				break

			} else if matcher.Regex != nil {
				if matcher.Regex.MatchString(part) {
					file := matcher.Ref.(string)
					key := strings.TrimRight(strings.TrimLeft(file, "["), "]")
					params[key] = part
					fileParts = append(fileParts, file)
					matched = true
					break
				}
			}
		}

		if !matched {
			return "", nil, fmt.Errorf("no route matchers")
		}
	}
	return filepath.Join(fileParts...) + ".sui", params, nil
}

func params(c *gin.Context) map[string]string {
	return nil
}

func payload(c *gin.Context) (map[string]interface{}, interface{}, error) {
	contentType := c.Request.Header.Get("Content-Type")
	var payload map[string]interface{}
	var body interface{}

	switch contentType {
	case "application/x-www-form-urlencoded":
		c.Request.ParseForm()
		payload = make(map[string]interface{})
		for key, value := range c.Request.Form {
			payload[key] = value
		}
		body = nil
		break

	case "multipart/form-data":
		c.Request.ParseMultipartForm(32 << 20)
		payload = make(map[string]interface{})
		for key, value := range c.Request.MultipartForm.Value {
			payload[key] = value
		}
		body = nil
		break

	case "application/json":
		if c.Request.Body == nil {
			return nil, nil, nil
		}

		c.Bind(&payload)
		body = nil
		break

	default:
		if c.Request.Body == nil {
			return nil, nil, nil
		}

		var data []byte
		_, err := c.Request.Body.Read(data)
		if err != nil && err.Error() != "EOF" {
			return nil, nil, err
		}
		body = data
	}

	return payload, body, nil
}
