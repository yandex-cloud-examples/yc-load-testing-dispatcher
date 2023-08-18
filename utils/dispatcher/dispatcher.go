package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
)

var (
	mutex       sync.Mutex
	port        = flag.Int("port", 8888, "port to listen")
	ssl         = flag.Bool("ssl", false, "use ssl")
	noproxy     = flag.Bool("noproxy", false, "save request without proxing")
	saveall     = flag.Bool("saveall", false, "save all requests without checking them for the return code")
	nostatic    = flag.Bool("nostatic", false, "exclude static fiiles like css, jpeg, etc")
	staticFiles = [...]string{"css", "js", "jpeg", "jpg", "png", "gif", "ico", "svg", "woff", "woff2"}
	target      = flag.String("target", "", "target host")
)

type HttpRequest struct {
	Host     string      `json:"host"`
	URI      string      `json:"uri"`
	Method   string      `json:"method"`
	Protocol string      `json:"-"`
	CLength  int64       `json:"-"`
	Headers  http.Header `json:"-"`
	Body     []byte      `json:"-"`
}

type HttpJson struct {
	HttpRequest
	Tag     string            `json:"tag"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body,omitempty"`
}

func main() {
	flag.Parse()

	http.HandleFunc("/", RootHandler)
	fmt.Println("Dispatcher listens to your requests")
	fmt.Println(runServer())
}

func runServer() error {
	return http.ListenAndServe(fmt.Sprintf(":%d", *port), nil)
}

func RootHandler(w http.ResponseWriter, r *http.Request) {

	h := &HttpRequest{}
	if err := h.ParseRequest(r); err != nil {
		fmt.Println(err)
	}

	if *noproxy {
		if err := h.SaveRequest(); err != nil {
			fmt.Fprint(w, err)
		} else {
			fmt.Fprint(w, "request is saved")
		}
	} else {

		proxyReq, err := h.getProxyRequest()
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
		}

		client := &http.Client{}
		resp, err := client.Do(proxyReq)
		if err != nil {
			fmt.Println(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode < http.StatusMultipleChoices || *saveall {
			if err := h.SaveRequest(); err != nil {
				fmt.Println(err)
			}
		}

		copyHeader(w.Header(), resp.Header)
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	}
}

// Request Proxying Section
func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func (h *HttpRequest) getProxyRequest() (*http.Request, error) {

	schema := "http"
	if *ssl {
		schema = "https"
	}

	req, err := http.NewRequest(
		h.Method,
		fmt.Sprintf("%s://%s%s", schema, h.Host, h.URI),
		bytes.NewBuffer(h.Body),
	)

	if err != nil {
		return nil, err
	}

	req.Header = h.Headers

	return req, nil
}

// Payload File Recording Section
func (h *HttpRequest) ParseRequest(r *http.Request) error {
	h.URI = r.RequestURI
	h.Method = r.Method
	h.Protocol = r.Proto
	h.CLength = r.ContentLength

	h.Headers = r.Header
	h.Headers.Del("Referer")

	if *target != "" {
		h.Host = *target
	} else {
		host, ok := r.Header["Host"]
		if ok {
			h.Host = host[0]
		} else {
			h.Host = r.Host
		}
	}

	if r.ContentLength > 0 {
		buf, err := io.ReadAll(r.Body)
		if err != nil {
			return err
		}
		h.Body = buf
	}

	return nil
}

func (h *HttpRequest) SaveRequest() error {
	if *nostatic && checkUriForStatic(h.URI) {
		return nil
	}

	if len(h.Body) == int(h.CLength) {
		if err := WriteToFile("raw.payload", h.FormatRaw()); err != nil {
			return err
		}
		if err := WriteToFile("httpjson.payload", h.FormatHttpJson()); err != nil {
			return err
		}
		if h.Method == "POST" {
			if err := WriteToFile("uripost.payload", h.FormatUriPost()); err != nil {
				return err
			}
		}
		if h.Method == "GET" {
			if err := WriteToFile("uri.payload", h.FormatUri()); err != nil {
				return err
			}
		}
	}

	return nil
}

func (h *HttpRequest) FormatUri() []byte {
	if h.Method == "GET" {
		return []byte(
			fmt.Sprintf(
				"%s\r\n%s %s\r\n",
				h.getHeaders(),
				h.URI,
				h.getTag()))
	} else {
		return nil
	}
}

func (h *HttpRequest) FormatUriPost() []byte {
	if h.Method == "POST" {
		return []byte(
			fmt.Sprintf(
				"%s\r\n%d %s\r\n%s\r\n",
				h.getHeaders(),
				h.CLength,
				h.URI,
				string(h.Body)))
	} else {
		return nil
	}
}

func (h *HttpRequest) FormatRaw() []byte {
	raw := fmt.Sprintf(
		"%s %s %s\r\n%s\r\n\r\n%s",
		h.Method,
		h.URI,
		h.Protocol,
		regexp.MustCompile(`\[*\]*`).ReplaceAllString(h.getHeaders(), ""),
		string(h.Body))

	return []byte(
		fmt.Sprintf(
			"%d %s\n%s\r\n",
			len(raw),
			h.getTag(),
			raw))
}

func (h *HttpRequest) FormatHttpJson() []byte {
	hj := &HttpJson{
		*h,
		h.getTag(),
		make(map[string]string),
		string(h.Body),
	}
	for k, v := range h.Headers {
		hj.Headers[k] = strings.Join(v, ", ")
	}

	data, err := json.Marshal(hj)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	return data
}

func (h *HttpRequest) getTag() string {
	tag := "root"
	if h.URI != "/" {
		tag = fmt.Sprintf(
			"%s%s",
			strings.ToLower(h.Method),
			strings.ReplaceAll(strings.Split(h.URI, "?")[0], "/", "_"))
	}

	return tag
}

func (h *HttpRequest) getHeaders() string {
	headers := make([]string, 0, len(h.Headers)+1)
	if _, ok := h.Headers["Host"]; !ok {
		headers = append(headers, fmt.Sprintf("[Host: %v]", h.Host))
	}
	for k, v := range h.Headers {
		headers = append(headers, fmt.Sprintf("[%v: %v]", k, strings.Join(v, ", ")))
	}

	return strings.Join(headers, "\n")
}

func WriteToFile(name string, data []byte) error {
	mutex.Lock()
	defer mutex.Unlock()

	file, err := os.OpenFile(name, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	n, err := file.Write(data)
	if err != nil {
		return err
	}
	if n != len(data) {
		return fmt.Errorf("Not full payload was written")
	}
	file.Write([]byte("\r\n"))
	fmt.Printf("The payload is written to a %s file\n", name)

	return nil
}

func checkUriForStatic(uri string) bool {
	split := strings.Split(strings.Split(uri, "?")[0], ".")
	ext := split[len(split)-1]
	for _, v := range staticFiles {
		if ext == v {
			return true
		}
	}

	return false
}
