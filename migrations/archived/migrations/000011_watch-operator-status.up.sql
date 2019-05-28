drop table worker_watch_state;

alter table watch add column desired_operator_status text null;
alter table watch add column current_operator_status text null;
