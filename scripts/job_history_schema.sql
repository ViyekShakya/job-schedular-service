-- auto-generated definition
create table job_history
(
    id             varchar(100)             not null
        primary key,
    job_id         uuid                     not null
        constraint fk_jobs_history
            references ??? ()
        on delete cascade,
    worker_id      varchar(100)             not null,
    attempt_number bigint                   not null,
    started_at     timestamp with time zone not null,
    completed_at   timestamp with time zone,
    status         varchar(20)              not null,
    error_message  text,
    duration       bigint default 0         not null
);

alter table job_history
    owner to admin;

create index idx_job_history_job_id
    on job_history (job_id);

