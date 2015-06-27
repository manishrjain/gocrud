create table instructions (
	subject_id varchar(32),
	subject_type varchar(32),
	predicate varchar(255),
	object blob,
	object_id varchar(32),
	nano_ts bigint,
	source text,

	id integer auto_increment,
	primary key (id)
);
