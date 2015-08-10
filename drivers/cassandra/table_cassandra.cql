create keyspace crudtest with replication = {'class': 'SimpleStrategy', 'replication_factor': 1} and durable_writes = true;
use crudtest;

create table instructions (
	subject_id text,
	ts timeuuid,
	subject_type text,
	predicate text,
	object blob,
	object_id text,
	nano_ts bigint,
	source text,
	PRIMARY KEY (subject_id, ts)
	) with compaction = {'class': 'LeveledCompactionStrategy'};
