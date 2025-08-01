-- auto-generated definition
create table workers
(
    id              varchar(100)             not null
        primary key,
    hostname        varchar(255)             not null,
    supported_types text                     not null,
    max_concurrent  bigint                   not null,
    current_jobs    bigint default 0         not null,
    status          varchar(20)              not null,
    last_heartbeat  timestamp with time zone not null,
    registered_at   timestamp with time zone not null
);

alter table workers
    owner to admin;

create index idx_workers_last_heartbeat
    on workers (last_heartbeat);

create index idx_workers_status
    on workers (status);

