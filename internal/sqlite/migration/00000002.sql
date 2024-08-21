create table
  if not exists user_events_migrated (
    id integer not null primary key,
    user_id integer not null,
    event_id integer,
    created_at datetime not null,
    `type` text NOT NULL,
    UNIQUE (user_id, event_id, `type`)
  );

INSERT INTO user_events_migrated (user_id, event_id, created_at, `type`) SELECT user_id, event_id, created_at, type FROM user_events;

DROP TABLE user_events;

ALTER TABLE user_events_migrated RENAME TO user_events;
