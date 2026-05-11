# Database migrations

Run migrations in lexical order against a Postgres database before starting the API.

The first migration creates the relational shell for identity, anonymous sessions,
house ownership, JSONB house state, audit events, AI runs, proposal drafts, and
future MCP/API tokens.
