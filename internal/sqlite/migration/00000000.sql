create table
  if not exists users (
    id integer not null primary key,
    name text unique,
    created_at datetime not null
  );

create table
  if not exists events (
    id integer not null primary key,
    type text not null,
    started_at datetime not null,
    finished_at datetime
  );

create table
  if not exists user_events (
    id integer not null primary key,
    user_id integer not null,
    event_id integer,
    created_at datetime not null,
    UNIQUE (user_id, event_id)
  );
