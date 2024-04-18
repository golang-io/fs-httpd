package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang-io/requests"
	"io"
	"log"
	"net/http"
	"os"
)

func Token(token string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if token == "" || r.Header.Get("Token") != token {
				http.Error(w, "token header is must!", http.StatusBadRequest)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func Method(method string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if method != "" && r.Method != method {
				http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func Echo(w http.ResponseWriter, r *http.Request) { _, _ = io.Copy(w, r.Body) }

func RequestLog(output string) func(http.Handler) http.Handler {
	out := os.Stdout
	if output != "stdout" {
		var err error
		if out, err = os.OpenFile(output, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644); err != nil {
			out = os.Stdout
			fmt.Println(err)
		}
	}
	return middleware.RequestLogger(&middleware.DefaultLogFormatter{Logger: log.New(out, "", log.LstdFlags)})
}

func H(r *requests.ServeMux, path string) {
	r.Route(path, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(r.URL.Path))
	})
}

// curl http://127.0.0.1:8080 -F '123=@xxx.json' -F '456=@jjj.json' -vvvv
func main() {
	var token = flag.String("token", "byFjRL3cr4v656AojKjW", "上传使用的TOKEN")
	var prefix = flag.String("prefix", "/tmp", "上传文件的前缀路径")
	var listen = flag.String("listen", "0.0.0.0:8080", "监听端口")
	var output = flag.String("output", "stdout", "日志输出")
	flag.Parse()
	r := requests.NewServeMux(requests.URL(*listen), requests.Use(RequestLog(*output), middleware.Recoverer))
	r.Route("/echo", Echo)
	r.Route("/ping", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("pong")) })

	r.Route("/_upload", requests.ServeUpload(*prefix), requests.Use(Token(*token), Method("POST")))
	r.Route("/backup", func(w http.ResponseWriter, r *http.Request) {
		http.StripPrefix("/backup", http.FileServer(http.Dir(*prefix))).ServeHTTP(w, r)
	}, requests.Use(Method("GET")))

	r.Route("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/backup/", http.StatusFound)
	})
	r.Pprof()
	err := requests.ListenAndServe(context.Background(), r)
	fmt.Println(err)
}
