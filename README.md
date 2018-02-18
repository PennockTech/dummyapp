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

## Setup

```
heroku apps:create pt-dummy-app
heroku labs:enable go-language-metrics
heroku labs:enable runtime-heroku-metrics

make BUILD_TAGS=heroku heroku-deploy
```

Created repo on Docker Hub through web UI: `pennocktech/dummyapp`

## To Do

We have a serving area, `poetry/` (which can be repointed).
We deliberately do not include that in the repo.
The build system for the final deployed container when constructed in CI
should integrate a poem or two into that area.
But the CI auto-build is not done yet.
