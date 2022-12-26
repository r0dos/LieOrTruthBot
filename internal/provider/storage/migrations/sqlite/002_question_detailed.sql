-- +goose Up

alter table questions
    add detailed TEXT default '';

-- +goose Down
begin;

create table questions_dg_tmp
(
    id       integer
        constraint questions_pk
            primary key autoincrement,
    question TEXT    not null,
    answer   integer not null,
    user_id  integer not null
);

insert into questions_dg_tmp(id, question, answer, user_id)
select id, question, answer, user_id
from questions;

drop table questions;

alter table questions_dg_tmp
    rename to questions;

create unique index questions_question_uindex
    on questions (question);

commit;
