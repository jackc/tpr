\set primary_test_database_name `echo $TEST_DATABASE`
\set copied_test_database_count `echo $TEST_DATABASE_COUNT`

drop database if exists :primary_test_database_name;
create database :primary_test_database_name;

\c :primary_test_database_name

-- Do setup here...
\setenv PGDATABASE :primary_test_database_name
\! tern migrate -c postgresql/tern.conf -m postgresql/migrations

\o /dev/null
\i test/testdata/pgundolog.sql
select pgundolog.create_trigger_for_all_tables_in_schema('public');
\o

create schema testdb;
create table testdb.databases (name text primary key, acquirer_pid int);

insert into testdb.databases (name)
select :'primary_test_database_name' || '_' || n
from generate_series(1, :copied_test_database_count) n;

select format('drop database if exists %I with (force)', name),
	format('create database %I template = %I', name, :'primary_test_database_name')
from testdb.databases
\gexec
