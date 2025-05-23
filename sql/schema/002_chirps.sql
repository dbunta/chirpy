-- +goose Up
create table chirps (
    id uuid primary key,
    created_at timestamp not null,
    updated_at timestamp not null,
    body text not null, 
    user_id uuid not null,
    constraint fk_users foreign key (user_id) references users(id) on delete cascade
);

-- +goose Down
drop table chirps;