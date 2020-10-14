package main

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/concourse/concourse/go-concourse/concourse"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/thomasmitchell/prism/config"
	"golang.org/x/oauth2"
	"gopkg.in/square/go-jose.v2/jwt"
)

func main() {
	configPath := mustEnv("CONFIG")
	log("loading config at path `%s'", configPath)
	cfg, err := config.Load(configPath)
	if err != nil {
		bailWith("Error loading config: %s", err)
	}
	clientTransport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: cfg.Concourse.InsecureSkipVerify,
		},
	}

	authClient := &http.Client{Transport: clientTransport}

	concourseClient := concourse.NewClient(
		cfg.Concourse.URL,
		&http.Client{
			Transport: &oauth2.Transport{
				Source: oauth2.ReuseTokenSource(nil, &concourseAuth{
					client:   authClient,
					url:      cfg.Concourse.URL,
					username: cfg.Concourse.Auth.Username,
					password: cfg.Concourse.Auth.Password,
				}),
				Base: clientTransport,
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

	bindAddr := fmt.Sprintf(":%d", cfg.Server.Port)
	if cfg.Server.TLS.Enabled {
		log("starting server with TLS on port %d", cfg.Server.Port)
		err = http.ListenAndServeTLS(
			bindAddr,
			cfg.Server.TLS.CertificatePath,
			cfg.Server.TLS.PrivateKeyPath,
			router,
		)
	} else {
		log("starting server without TLS on port %d", cfg.Server.Port)
		err = http.ListenAndServe(bindAddr, router)
	}

	bailWith("Server exited", err)
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

type concourseAuth struct {
	client   *http.Client
	url      string
	username string
	password string
}

func (c concourseAuth) Token() (*oauth2.Token, error) {
	oauth2Config := oauth2.Config{
		ClientID:     "fly",
		ClientSecret: "Zmx5",
		Endpoint:     oauth2.Endpoint{TokenURL: c.url + "/sky/issuer/token"},
		Scopes:       []string{"openid", "profile", "email", "federated:id", "groups"},
	}

	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, c.client)

	token, err := oauth2Config.PasswordCredentialsToken(ctx, c.username, c.password)
	if err != nil {
		log("error fetching oauth2 token: %s", err)
		return nil, err
	}
	expiry, err := c.parseTokenExpiry(token.AccessToken)
	if err != nil {
		log("error parsing token expiry: %s", err)
		return nil, fmt.Errorf("error parsing token expiry: %s", err)
	}

	return &oauth2.Token{
		TokenType:   token.TokenType,
		AccessToken: token.AccessToken,
		Expiry:      expiry,
	}, nil
}

func (c concourseAuth) parseTokenExpiry(token string) (time.Time, error) {
	raw, err := base64.RawStdEncoding.DecodeString(token)
	if err != nil {
		return time.Time{}, err
	}
	if len(raw) != 28 {
		return time.Time{}, errors.New("invalid access token length")
	}
	expiry := jwt.NumericDate(binary.LittleEndian.Uint64(raw[20:]))
	return expiry.Time(), nil
}
