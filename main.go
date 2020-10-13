package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/concourse/concourse/go-concourse/concourse"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/thomasmitchell/prism/config"
)

func main() {
	configPath := mustEnv("CONFIG")
	log("loading config at path `%s'", configPath)
	cfg, err := config.Load(configPath)
	if err != nil {
		bailWith("Error loading config: %s", err)
	}

	concourseClient := concourse.NewClient(
		cfg.Concourse.URL,
		&http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: cfg.Concourse.InsecureSkipVerify,
				},
			},
		},
		false,
	)

	router := mux.NewRouter()
	router.Handle(
		"/v1/webhook/git/{team}/{pipeline}",
		&HookHandler{
			Client: concourseClient,
		},
	).Methods("GET", "POST", "PUT")

	if cfg.Server.TLS.Enabled {
		log("starting server with TLS on port %d", cfg.Server.Port)
		listenAndServeTLS(&cfg.Server, router)
	} else {
		log("starting server without TLS on port %d", cfg.Server.Port)
		http.ListenAndServe(fmt.Sprintf(":%d", cfg.Server.Port), router)
	}
}

func listenAndServeTLS(conf *config.Server, handler http.Handler) error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", conf.Port))
	if err != nil {
		return err
	}

	defer ln.Close()

	cert, err := tls.X509KeyPair([]byte(conf.TLS.Certificate), []byte(conf.TLS.PrivateKey))
	if err != nil {
		return err
	}

	tlsListener := tls.NewListener(ln, &tls.Config{
		NextProtos:   []string{"http/1.1"},
		Certificates: []tls.Certificate{cert},
	})

	return http.Serve(tlsListener, handler)
}

func mustEnv(envvar string) string {
	v := os.Getenv(envvar)
	if v == "" {
		bailWith("Required envvar %s not found", envvar)
	}

	return v
}

func bailWith(f string, args ...interface{}) {
	fLog(os.Stderr, f, args...)
	os.Exit(1)
}

func log(f string, args ...interface{}) {
	fLog(os.Stdout, f, args...)
}

func logReq(uuid uuid.UUID, f string, args ...interface{}) {
	log("%s: "+f, append([]interface{}{uuid.String()}, args...)...)
}

func fLog(w io.Writer, f string, args ...interface{}) {
	fmt.Printf("%s: "+f+"\n", append([]interface{}{time.Now().Format(time.RFC3339)}, args...)...)
}
