# Blog Example

A full-featured blog application demonstrating [htmlc](../../README.md) in a real-world Go web server. It uses event sourcing for persistence and Vue-style single-file components for server-side rendering.

## Features

- Public blog with post listing, individual post pages, and an Atom feed
- Admin panel with session-based authentication
- Create, edit, publish, unpublish, and delete posts
- Impression (view) tracking per post
- Event-sourced persistence via an append-only JSONL log file

## Files

| File | Description |
|------|-------------|
| `main.go` | Entry point; reads config from environment variables and starts the HTTP server |
| `server.go` | HTTP routes and handlers; uses htmlc to render Vue-style templates |
| `store.go` | Event-sourced post store backed by a JSONL log file |
| `templates/` | htmlc components (`.vue` files) for every page |

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | TCP port the server listens on |
| `ADMIN_USERNAME` | `admin` | Admin login username |
| `ADMIN_PASSWORD` | `password` | Admin login password |
| `SITE_TITLE` | `My Blog` | Blog title shown in templates |
| `LOG_FILE` | `blog.jsonl` | Path to the JSONL event log file |

## Running

```bash
cd examples/blog
go run .
```

With custom settings:

```bash
PORT=9000 SITE_TITLE="My Awesome Blog" ADMIN_PASSWORD=secret go run .
```

The server logs the listening address and admin URL on startup:

```
2024/01/01 12:00:00 listening on http://localhost:8080
2024/01/01 12:00:00 admin: http://localhost:8080/admin (user: admin)
```

## Routes

### Public

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/` | Home page — lists published posts |
| `GET` | `/posts/{id}` | Individual post page |
| `GET` | `/feed.atom` | Atom feed of published posts |

### Admin

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/admin/login` | Login form |
| `POST` | `/admin/login` | Submit credentials |
| `POST` | `/admin/logout` | Log out |
| `GET` | `/admin/` | Dashboard — lists all posts |
| `GET` | `/admin/drafts` | Drafts-only view |
| `GET` | `/admin/posts/new` | New post form |
| `POST` | `/admin/posts/new` | Create a draft |
| `GET` | `/admin/posts/{id}/edit` | Edit post form |
| `POST` | `/admin/posts/{id}/edit` | Save edits |
| `POST` | `/admin/posts/{id}/publish` | Publish a draft |
| `POST` | `/admin/posts/{id}/unpublish` | Revert to draft |
| `POST` | `/admin/posts/{id}/delete` | Delete a post |

## Persistence

All state changes are appended to the JSONL log file (`blog.jsonl` by default). On startup the log is replayed from scratch to rebuild in-memory state. The log is never modified or truncated — only appended to.

Event types stored in the log:

- `PostCreated`
- `PostUpdated`
- `PostPublished`
- `PostUnpublished`
- `PostDeleted`
- `ImpressionRecorded`
