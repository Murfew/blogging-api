# blogging-api

A simple REST API for managing blog posts, built with Go's standard library and PostgreSQL.

## Features

- Create, read, update, and delete blog posts
- Search posts by keyword across title, content, and category
- PostgreSQL-backed with connection pooling via `pgx`

## Tech Stack

- **Go** — standard library `net/http` (no framework)
- **PostgreSQL** — via [pgx/v5](https://github.com/jackc/pgx)

## Prerequisites

- Go 1.25+
- PostgreSQL instance with a `posts` table

### Database schema

```sql
CREATE TABLE posts (
    id         SERIAL PRIMARY KEY,
    title      TEXT        NOT NULL,
    content    TEXT        NOT NULL,
    category   TEXT        NOT NULL,
    tags       TEXT[]      NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

## Setup

1. Clone the repo and install dependencies:

```bash
git clone https://github.com/murfew/blogging-api.git
cd blogging-api
go mod download
```

2. Create a `.env` file (or export the variable directly):

```env
DATABASE_URL=postgres://user:password@localhost:5432/dbname
```

3. Run the server:

```bash
go run main.go
```

The server starts on `http://localhost:8080`.

## API

### Posts

| Method | Path          | Description                          |
|--------|---------------|--------------------------------------|
| GET    | `/posts`      | List all posts                       |
| GET    | `/posts?term=` | Search posts by keyword             |
| GET    | `/posts/{id}` | Get a post by ID                     |
| POST   | `/posts`      | Create a new post                    |
| PUT    | `/posts/{id}` | Update an existing post              |
| DELETE | `/posts/{id}` | Delete a post                        |

### Post schema

```json
{
  "id": 1,
  "title": "Hello World",
  "content": "My first post.",
  "category": "general",
  "tags": ["go", "api"],
  "created_at": "2026-06-06T12:00:00Z",
  "updated_at": "2026-06-06T12:00:00Z"
}
```

`title`, `content`, and `category` are required on create and update. `tags` is optional.

## License

[MIT](LICENSE)
