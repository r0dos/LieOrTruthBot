-- +goose Up
begin;

create table chat_user_dg_tmp
(
    chat_id    integer                            not null,
    user_id    integer                            not null,
    value      integer  default 1                 not null,
    created_at datetime default current_timestamp not null,
    updated_at datetime default current_timestamp not null
);

insert into chat_user_dg_tmp(chat_id, user_id, value, created_at, updated_at)
select chat_id, user_id, value, created_at, updated_at
from chat_user;

drop table chat_user;

alter table chat_user_dg_tmp
    rename to chat_user;

create index chat_id_index
    on chat_user (chat_id);

create unique index chat_user_chat_id_user_id_uindex
    on chat_user (chat_id, user_id);

commit;

-- +goose Down
begin;

create table chat_user_dg_tmp
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

insert into chat_user_dg_tmp(chat_id, user_id, value, created_at, updated_at)
select chat_id, user_id, value, created_at, updated_at
from chat_user;

drop table chat_user;

alter table chat_user_dg_tmp
    rename to chat_user;

create unique index chat_user_chat_id_user_id_uindex
    on chat_user (chat_id, user_id);

create index chat_id_index
    on chat_user (chat_id);

commit;
