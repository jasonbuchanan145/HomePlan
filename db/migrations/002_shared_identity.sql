alter table users
  add column if not exists primary_email text,
  add column if not exists avatar_url text,
  add column if not exists last_login_at timestamptz;

update users
set primary_email = lower(email)
where primary_email is null and email is not null;

create unique index if not exists users_primary_email_unique_idx
  on users (lower(primary_email))
  where primary_email is not null;

create table if not exists user_identities (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references users(id) on delete cascade,
  provider text not null,
  provider_subject text not null,
  email text,
  email_verified boolean not null default false,
  display_name text,
  avatar_url text,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique (provider, provider_subject)
);

create index if not exists user_identities_user_id_idx on user_identities (user_id);

create table if not exists user_sessions (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references users(id) on delete cascade,
  token_hash text not null unique,
  expires_at timestamptz not null,
  created_at timestamptz not null default now(),
  last_seen_at timestamptz not null default now(),
  revoked_at timestamptz
);

create index if not exists user_sessions_user_id_idx on user_sessions (user_id);
create index if not exists user_sessions_expires_at_idx on user_sessions (expires_at);

create table if not exists app_entitlements (
  user_id uuid not null references users(id) on delete cascade,
  app_key text not null,
  can_access boolean not null default false,
  can_use_ai boolean not null default false,
  metadata jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  primary key (user_id, app_key)
);

create index if not exists app_entitlements_app_key_idx on app_entitlements (app_key);
