# prism

## why prism?

Concourse's isolation of resources keeps implementations clean, but can also
lead to pipeline paradigms where there are multiple resource specs that point
to the same Git repository, aimed at triggering off of different files within
the repo.

Having multiple resources for each repository can lead to your Concourse
pipelines sending a lot of clone requests against the remote repository. To
alleviate this, one may want to use Concourse's incoming webhook implementation
to trigger checks. Except now you have to write multiple webhook rules for
the same repository.

`prism` can take a single incoming webhook for a repository and send it to all
of the git resources in your pipelines that need to receive it.

## how does prism do that thing that it does?

`prism` authenticates to Concourse as a local auth user. It uses this
authentication to see teams and pipeline configurations. When a webhook request
comes in, it fetches the pipeline configuration from the Concourse API and
scours it for git resources. If the resource points to the same git repository
as the one specified in the request, it will forward a webhook to that resource.
The webhook token sent in the outgoing request will be the same as the one
found in the incoming webhook request.

the git uri in the incoming request and the uri in pipeline config do not have
to match exactly. `prism` does some normalization on the uris such that a uri
for https will match a uri of the same repository for ssh. Also, prism does not
care about the `.git` suffix sometimes appended to repository uris.

## how does one prism?

`prism` runs as an HTTP server program. `prism` reads in a YAML-formatted
configuration file. The path to this configuration file is given to the
program as the `CONFIG` environment variable. A minimal configuration file
looks like this:

```yaml
concourse:
  url: https://concourse.example.com
  auth:
    username: concourse
    password: mypassword
```

All other configuration options are optional. An exhaustive configuration looks
like this:

```yaml
concourse:
  # (string); the URL of the Concourse ATC server with the protocol scheme
  # included.
  url: https://concourse.example.com
  # (bool); whether to verify the server certificate when connecting to the
  # Concourse ATC server. Defaults to false.
  insecure_skip_verify: false
  auth:
    # (string); the username to authenticate to the ATC as
    username: concourse
    # (string); the password to authenticate to the ATC with
    password: mypassword

server:
  # (number); the port for the prism server to listen on. Defaults to 4580.
  port: 4580
  tls:
    # (bool); whether the prism server should listen on TLS. Defaults to false.
    enabled: true
    # (string); the certificate that the prism server should serve
    certificate: |
      -----BEGIN CERTIFICATE-----
      a certificate would go here
      -----END CERTIFICATE-----
    private_key: |
      -----BEGIN PRIVATE KEY-----
      a private key would go here
      -----END PRIVATE KEY-----
```

## how does one configure a webhook for use with prism?

Send your webhook to your prism server in the following format:

```
https://prism.example.com:4580/v1/webhook/git/[team]/[pipeline]?git_url=[git-clone-uri]&webhook_token=[token]
```

* `[team]` should be replaced with the name of the team the pipeline resides in.
* `[pipeline]` should be replaced with the name of the pipeline to send
webhooks to.
* `[git-clone-uri]` should be replaced with the uri that you would configure as
the remote for the given repository. This has to be query encoded for a URL, so
an example would look like
`https%3A%2F%2Fgithub.com%2Fthomasmitchell%2Fprism.git`. Ugly, but correct.
* `[token]` should be replaced with the webhook token configured for the
pipeline`s git resources. This also needs to be query encoded for a URL. Perhaps
just use characters for your sufficiently long token that don't need to be
handled specially by URL encoding.
