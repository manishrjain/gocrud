package main

type Update interface {
	SetSource(source string) Update // Authorship
	AddChild(kind string) Update    // New entry in table kind
	MarkDeleted() Update            // Set deleted to true
	Set(property string, value interface{}) Update
	Execute() error
}

type Query interface {
	UptoDepth(level int) Query
	Collect(kind string) Query       // select * from kind where parent = me;
	FilterOut(property string) Query // where property is NULL;
	Run() (Result, error)
}
type Result struct{}
