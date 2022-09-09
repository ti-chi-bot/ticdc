drop database if exists `gbk`;
create database `gbk` character set gbk;
use `gbk`;

create table t1 (id int, name varchar(20), primary key(`id`)) character set gbk;
insert into t1 (id, name) values (0, '你好0');
insert into t1 (id, name) values (1, '你好1'), (2, '你好2');

create table t3 (id int, name varchar(20) character set gbk, primary key(`id`)) character set utf8;
insert into t3 (id, name) values (0, '你好0');
insert into t3 (id, name) values (1, '你好1'), (2, '你好2');

create table t5 (id int, name varchar(20), primary key(`id`)) character set latin1;
insert into t5 (id, name) values (0, 'Müller');
