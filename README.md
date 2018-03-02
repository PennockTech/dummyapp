dummyapp
========

This is a simple app for working out the flows for automated Docker image
creation and deployment from within CI.

No warranty.  You get to keep all the pieces and shards if it breaks.
There's a 2-clause BSD license as a formality.

There should be little enough here in terms of "traditional code", but the
infrastructure of how pieces fit together may be useful to you, after reading
and analysis.  If you use any of it, then a word of attribution might be nice
(and will absolve you of the need to honor the formal copyright notice
propagation for build framework).

This Git repo is setup so that pushes automatically trigger builds within
Circle CI, which creates a from-scratch Docker image (using a multi-stage
Dockerfile) and deploys it to both Docker Hub and, for master branch,
to Heroku.

## Terminology

* *DinD*: Docker-in-Docker
* *Controller Image*: the image launched by Circle CI to handle the steps from
  `.circleci/config.yml`; this is not invoked _by_ `docker build`, but is the
  image where a shell command of `docker build` will be run, via make.
  The Controller Image is not specified outside of the `.circleci/` directory.
* *Builder Image*: the multi-stage `Dockerfile` creates two images; the first
  is the Builder Image and has a userland, normal tools, a compiler and more.
* *Deploy Image*: the much smaller image created later in the multi-stage
  `Dockerfile`, which copies files made in the earlier stages.  This is the
  "product" and is what is pushed out.

---

## Setup

We create a Heroku app, enable Go language metrics manually (because using
Docker deploy, not a buildpack), disable git push to the remote (but leave
the remote in place so that the Heroku CLI can auto-determine the deployed
app name), and do a build and deploy with the Heroku tag set.

The build-tag affects both the Docker image tag-name and the content which
is built; for Heroku, it ensures that we compile their metrics push code.

```
heroku apps:create pt-dummy-app
heroku labs:enable go-language-metrics
heroku labs:enable runtime-heroku-metrics

git config --local --unset remote.heroku.fetch
git config --local remote.heroku.pushurl no_push_because_we_deploy_docker_images
git config --local --bool remote.heroku.skipFetchAll true

make BUILD_TAGS=heroku heroku-deploy
```

Created repo on Docker Hub through web UI: `pennocktech/dummyapp`

For a second project, `philpennock/poetry` (as an example of depending upon
external data) I set up an automated Docker Hub build.  That project creates
a data-only Docker image, which we now depend upon at build time.  There's
one `COPY --from` line in our `Dockerfile` to edit to remove that.

Created Circle CI project; pushed on branch circle, aborted first build on
master.


### Authentication

Created a Circle CI org-level Context, `heroku-and-dockerhub`, added
credentials there for `HEROKU_TOKEN`, `DOCKERHUB_USER`, and
`DOCKERHUB_PASS`.

As to the values:

1. Heroku is bad enough in only having one auth token at a time, unscoped, so
   no ability to trace leaks or constrain actions of the token.
   Run `heroku auth:token` while signed in, that's the token to use.
2. Docker Hub ... defaults to storing your usercode and master password in
   `~/.docker/config.json`; you probably want to install
   `docker-credential-helper` if you haven't already done so.
3. Docker Hub on v2 has an API for getting tokens, but the `offline_token=true`
   part of a request is not honored (that I can tell), so you don't get
   anything usable for passing into another service.
4. So: create a bot account for Docker Hub via normal sign-up; this bot can
   create arbitrary repos of its own, but only under its own account.
   Then in the fully privileged admin account, click `Organizations` in the
   header, then your organization, then near the top `Teams` and create a new
   team, `dummyapppushers` and add the new account to it.
   Then go to the repository page, `Collaborators`, and add the
   `dummyapppushers` team with Write access.
   + You can now use the usercode/password for the bot account in
     `DOCKERHUB_USER` and `DOCKERHUB_PASS`.

Now update the `.circleci/config.yml` to reference the context; yes, any build
within the org can request any context, you can't have admins defining
restricted contexts with some credentials.  If you want that, then you'll need
multiple Circle CI orgs (each with their own billing?).

### Build Dependencies

* We're using a `domainr/ci` image for building in Circle, simply because it's
  Go with a few tools added; we could work without it, but would need to
  install `dep`.
  <a href="https://github.com/domainr/ci">GitHub</a>,
  <a href="https://hub.docker.com/r/domainr/ci/">Docker Hub</a>.
* We merge in contents from a data image; at the size we're at, it's silly,
  but that's necessarily true for larger data sets.  I'm using
  `philpennock/poetry` which is just a couple of Rudyard Kipling poems.  
  <a href="https://github.com/philpennock/poetry">GitHub</a>,
  <a href="https://hub.docker.com/r/philpennock/poetry/">Docker Hub</a>.
* A Docker-official `golang` image, for the Builder Image.  
  <a href="https://github.com/docker-library/golang/">GitHub</a>,
  <a href="https://hub.docker.com/_/golang/">Docker Hub</a>.
  + But in Circle, we override this to be the same as the Controller Image.

All are automated Docker Hub builds as public images from public GitHub repos.
The `golang` image is from the `docker-library` GitHub organization, while the
others are from GitHub repos which have names matching the Docker Hub repo
names.

---

## Build Tricks

#### Use an HTTP proxy during build, and switch the base image:

Create:

```sh
DOCKER_http_proxy=http://192.0.2.1:3128/ DOCKER_RUNTIME_BASE_IMAGE=alpine \
  make build-image`
```

Run:

```sh
docker run --entrypoint=/bin/sh -it --rm ${imageid}
```

Note that because we default to Heroku bug-compatibility, we have to override
`ENTRYPOINT`, so when using Alpine you'll have to force it back.  Or change
the `build/Dockerfile`.
