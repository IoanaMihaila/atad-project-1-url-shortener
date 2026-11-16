# URL Shortener with Analytics

A lightweight HTTP web service written in Go that turns long `http`/`https` URLs into short codes, redirects visitors to the original address, and tracks visit counts and timestamps per link. It is built for anyone who needs shareable short links with basic usage analytics and persistent storage across service restarts.

- **Student:** Mihaila Ioana, group DMWD
- **Project:** Project 1 - URL Shortener with Analytics
- **Language:** Go

## What it does

The service exposes a REST-like JSON API for creating, listing, inspecting, and deleting short links, along with a redirect endpoint (`GET /{code}`) that forwards visitors via HTTP `302 Found`. Users can either request a custom short code (4 to 10 characters) or let the server generate a unique 7-character code automatically. Every redirect increments the link's visit counter and updates `last_visit_at`, while reading the statistics endpoint does not affect the count.

To keep the service focused, it does not implement user authentication, link expiration dates, custom domains, or a frontend dashboard—it acts purely as a containerized backend service behind HTTP on port `8080`.

At the moment, the repository structure, metadata, CI pipeline, and architecture documentation are set up, while the URL shortener handlers, disk persistence, and tests are in active development.

## How to run it

    docker run --rm -p 8080:8080 ghcr.io/ioanamihaila/atad-project-1-url-shortener:latest

To build and run locally without Docker:

    go run ./cmd/app
    go run ./cmd/app --version

## What you should see

Once the service is running on `http://localhost:8080`, checking `/health` returns HTTP `200 OK`:

    curl.exe -i http://localhost:8080/health

    HTTP/1.1 200 OK
    Content-Type: application/json

    {"status":"ok","uptime_seconds":5}

Creating a short link and following its redirect looks like this:

    curl.exe -i -H "Content-Type: application/json" -d '{"url":"https://go.dev/doc/effective_go","code":"gofmt"}' http://localhost:8080/links

    HTTP/1.1 201 Created
    Content-Type: application/json

    {"id":1,"code":"gofmt","url":"https://go.dev/doc/effective_go","created_at":"2026-11-16T21:00:00Z"}

Visiting `http://localhost:8080/gofmt` responds with `302 Found` and `Location: https://go.dev/doc/effective_go`.