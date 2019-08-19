# isfabianstillalive.com

Should be a relatively simple question, but my friends and relatives keep asking for updates over SMS.

This was a website that I had up and running in May 2017 before a solo trip into the Caucuses. It's not running any more, but the code is still here and there's a blog post with lots of info at https://capnfabs.net/posts/is-fabian-still-alive/.

## Run with: 

You should already have heroku tools and go installed.

```sh
# FIRST TIME SETUP
# We don't check this file in because then I think it updates the vars on 
# heroku. But we want them set locally.
cp .env_template .env
# Install govendor
go get -u github.com/kardianos/govendor
# I think that should do it?

# TO RUN THE APP
# ldflags -s is required because a couple of the SQL packages break on OSX
# because of some application verification thing I guess; I can't really
# remember. You can ditch that if you're on linux.
govendor install -ldflags -s ./cmd/... && heroku local
```

## Deploy with

```sh
git push heroku master
```

## How is the code set up?
- [`cmd/webroot`](./cmd/webroot) - this is where the application lives. Right now, it's in a single file.
- [`static/`](./static) - static files (CSS and the like). CSS is super minimal because we just import bootstrap over the CDN and then re-theme it in a really hacky way using the heroku theme colours from their 'getting started in go' repo.
- [`templates`](./templates) - HTML templates.
- [`vendor/vendor.json`](./vendor/vendor.json) - I'm using [govendor](https://github.com/kardianos/govendor) for tracking deps. There's almost certainly some stuff in there that isn't required.

## How to run the tests

lol.
