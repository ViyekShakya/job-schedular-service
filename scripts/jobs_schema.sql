-- auto-generated definition
create table jobs
(
    id           uuid                     not null
        primary key,
    type         varchar(50)              not null,
    priority     integer                  not null,
    payload      jsonb                    not null,
    metadata     jsonb,
    status       varchar(20)              not null,
    retry_policy jsonb                    not null,
    created_at   timestamp with time zone not null,
    updated_at   timestamp with time zone not null,
    scheduled_at timestamp with time zone not null,
    processed_at timestamp with time zone,
    completed_at timestamp with time zone,
    failed_at    timestamp with time zone,
    last_error   text,
    worker_id    varchar(100)
);

alter table jobs
    owner to admin;

create index idx_jobs_scheduled_at
    on jobs (scheduled_at);

create index idx_jobs_status
    on jobs (status);

create index idx_jobs_priority
    on jobs (priority);

create index idx_jobs_type
    on jobs (type);



