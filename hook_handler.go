package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/concourse/concourse/atc"
	"github.com/concourse/concourse/go-concourse/concourse"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type HookHandler struct {
	Client concourse.Client
}

func (h *HookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	reqID := uuid.New()

	vars := mux.Vars(r)
	var (
		teamIn     = vars["team"]
		pipelineIn = vars["pipeline"]
		tokenIn    = r.FormValue("webhook_token")
		gitURLIn   = h.canonizeGitURL(r.FormValue("git_url"))
	)

	logReq(
		reqID,
		"received hook for team `%s', pipeline `%s', gitURL `%s'\n",
		teamIn, pipelineIn, gitURLIn,
	)

	if tokenIn == "" {
		logReq(reqID, "no webhook_token provided")
		w.WriteHeader(http.StatusForbidden)
		return
	}

	if gitURLIn == "" {
		logReq(reqID, "no git_url provided")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	teamClient, err := h.Client.FindTeam(teamIn)
	if err != nil {
		logReq(
			reqID,
			"unable to find team `%s': %s",
			teamIn, err,
		)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	pipeInRef := atc.PipelineRef{Name: pipelineIn}

	cfg, _, found, err := teamClient.PipelineConfig(pipeInRef)
	if err != nil {
		logReq(
			reqID,
			"error when looking up pipeline `%s' against Concourse: %s",
			pipelineIn, err,
		)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if !found {
		logReq(
			reqID,
			"pipeline `%s' not found: %s",
			pipelineIn,
		)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	errCode := http.StatusNotFound

	for _, resource := range cfg.Resources {
		if !h.resourceMatches(reqID, gitURLIn, resource) {
			continue
		}

		respCode := h.doWebhook(reqID, teamIn, pipelineIn, resource.Name, tokenIn)
		if respCode/100 != 2 {
			w.WriteHeader(respCode)
			return
		}

		errCode = http.StatusOK
	}

	w.WriteHeader(errCode)
}

func (h *HookHandler) resourceMatches(reqID uuid.UUID, gitURL string, resource atc.ResourceConfig) bool {
	if resource.Type != "git" || resource.WebhookToken == "" {
		return false
	}

	gitResourceURIInterface, found := resource.Source["uri"]
	if !found {
		logReq(
			reqID,
			"git resource with name `%s' had no git URI",
			resource.Name,
		)
		return false
	}

	gitResourceURI, isString := gitResourceURIInterface.(string)
	if !isString {
		logReq(
			reqID,
			"git resource with name `%s' had non-string URI",
			resource.Name,
		)
		return false
	}

	return h.canonizeGitURL(gitResourceURI) == gitURL
}

func (h *HookHandler) doWebhook(reqID uuid.UUID, team, pipeline, resource, token string) int {
	webhookURL := h.genWebhookURL(
		h.Client.URL(), team, pipeline, resource, token,
	)

	logReq(
		reqID,
		"sending webhook request to `%s'",
		h.genRedactedWebhookURL(
			h.Client.URL(), team, pipeline, resource,
		),
	)

	resp, err := h.Client.HTTPClient().Get(webhookURL)
	if err != nil {
		logReq(
			reqID,
			"error when communicating with Concourse server: %s",
			err,
		)
		return http.StatusInternalServerError
	}

	defer func() {
		_, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			logReq(
				reqID,
				"failed to clear response body: %s",
				err,
			)
		}

		err = resp.Body.Close()
		if err != nil {
			logReq(
				reqID,
				"failed to close response body: %s",
				err,
			)
		}
	}()

	if resp.StatusCode/100 != 2 {
		logReq(
			reqID,
			"received non-2xx response from webhook: %s",
			resp.Status,
		)
	}

	return resp.StatusCode
}

var urlRegex = regexp.MustCompile(`^(?:.*@|.*:\/\/)(.*)(:?\.git)$`)

func (HookHandler) canonizeGitURL(u string) string {
	matches := urlRegex.FindStringSubmatch(u)
	if len(matches) < 2 {
		return ""
	}

	return strings.ReplaceAll(matches[1], ":", "/")
}

func (HookHandler) genWebhookURL(base, team, pipeline, resource, token string) string {
	return fmt.Sprintf("%s/api/v1/teams/%s/pipelines/%s/resources/%s/check/webhook?webhook_token=%s",
		base,
		url.PathEscape(team),
		url.PathEscape(pipeline),
		url.PathEscape(resource),
		url.QueryEscape(token))
}

func (h HookHandler) genRedactedWebhookURL(base, team, pipeline, resource string) string {
	return h.genWebhookURL(base, team, pipeline, resource, "REDACTED")
}
