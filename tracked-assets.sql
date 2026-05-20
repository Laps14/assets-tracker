-- create user tracker password 'AY0&#yX8';

-- create database assets_tracker owner tracker;
-- drop database if exists assets_tracker;
-- create database assets_tracker;

create table tracked_assets (
  id serial primary key,
  title varchar(7),
  description varchar(1024),
  stock_val numeric(4,2),
  target_vals numeric(4,2)[],
  company_logo bpchar
);

-- grant select,insert,update,delete on tracked_assets to tracker;
