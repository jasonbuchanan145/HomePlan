create extension if not exists pgcrypto;

create table if not exists users (
  id uuid primary key default gen_random_uuid(),
  email text unique,
  display_name text,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists anonymous_sessions (
  id uuid primary key default gen_random_uuid(),
  token text not null unique,
  expires_at timestamptz not null,
  created_at timestamptz not null default now()
);

create table if not exists houses (
  id uuid primary key default gen_random_uuid(),
  owner_user_id uuid references users(id) on delete set null,
  anonymous_session_id uuid references anonymous_sessions(id) on delete set null,
  name text not null default 'Home repair plan',
  expires_at timestamptz,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  constraint houses_identity_check check (owner_user_id is not null or anonymous_session_id is not null)
);

create table if not exists house_members (
  house_id uuid not null references houses(id) on delete cascade,
  user_id uuid not null references users(id) on delete cascade,
  role text not null default 'owner',
  created_at timestamptz not null default now(),
  primary key (house_id, user_id)
);

create table if not exists house_state (
  house_id uuid primary key references houses(id) on delete cascade,
  state jsonb not null,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists house_versions (
  id uuid primary key default gen_random_uuid(),
  house_id uuid not null references houses(id) on delete cascade,
  label text,
  state jsonb not null,
  created_by_user_id uuid references users(id) on delete set null,
  created_at timestamptz not null default now()
);

create table if not exists house_events (
  id uuid primary key default gen_random_uuid(),
  house_id uuid not null references houses(id) on delete cascade,
  actor_type text not null,
  actor_user_id uuid references users(id) on delete set null,
  event_type text not null,
  payload jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now()
);

create table if not exists proposed_changes (
  id uuid primary key default gen_random_uuid(),
  house_id uuid not null references houses(id) on delete cascade,
  source text not null,
  status text not null default 'pending',
  patch jsonb not null,
  summary text,
  created_by_user_id uuid references users(id) on delete set null,
  created_at timestamptz not null default now(),
  decided_at timestamptz
);

create table if not exists ai_runs (
  id uuid primary key default gen_random_uuid(),
  house_id uuid references houses(id) on delete set null,
  user_id uuid references users(id) on delete set null,
  anonymous_session_id uuid references anonymous_sessions(id) on delete set null,
  model text,
  prompt_type text not null,
  input_tokens integer,
  output_tokens integer,
  estimated_cost_cents integer,
  status text not null default 'queued',
  metadata jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now()
);

create table if not exists api_tokens (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references users(id) on delete cascade,
  name text not null,
  token_hash text not null,
  scopes text[] not null default '{}',
  last_used_at timestamptz,
  created_at timestamptz not null default now(),
  revoked_at timestamptz
);

create index if not exists anonymous_sessions_expires_at_idx on anonymous_sessions (expires_at);
create index if not exists houses_anonymous_session_id_idx on houses (anonymous_session_id);
create index if not exists house_events_house_id_created_at_idx on house_events (house_id, created_at desc);
create index if not exists proposed_changes_house_id_status_idx on proposed_changes (house_id, status);
create index if not exists ai_runs_house_id_created_at_idx on ai_runs (house_id, created_at desc);
