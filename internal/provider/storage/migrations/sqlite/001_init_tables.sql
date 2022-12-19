-- +goose Up
create table chat_user
(
    id integer not null
        constraint chat_user_pk
            primary key autoincrement,
    chat_id integer not null,
    user_id integer not null,
    value integer default 1 not null,
    created_at datetime default current_timestamp not null,
    updated_at datetime default current_timestamp not null
);

create unique index chat_user_chat_id_user_id_uindex
    on chat_user (chat_id, user_id);

create index chat_id_index
    on chat_user (chat_id);

create table admins
(
    user_id integer not null
);

create unique index admins_user_id_uindex
    on admins (user_id);

create table questions
(
    id       integer
        constraint questions_pk
            primary key autoincrement,
    question TEXT    not null,
    answer   integer not null,
    user_id  TEXT    not null
);

create unique index questions_question_uindex
    on questions (question);

-- +goose Down
DROP TABLE chat_user;

DROP TABLE admins;

DROP TABLE questions;
