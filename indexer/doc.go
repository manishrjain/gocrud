// Package indexer provides two different methods to run real time incremental
// indexing for entities.
//
// This package asks you to implement an Indexer interface, which takes care
// of dependency generation (for e.g. update parent entity if child entity
// is modified), and document regeneration and reindexing (regenerate
// search document for both parent and child entities, and update them
// in search index).
//
// This methodology allows for real time incremental indexing systems. They
// can be utilized as such:
//
// Method 1:
// As demonstrated in social.go, you can call indexer.Run(ctx, numRoutines)
// in your backend server directly, so as entities get updated via calls to
// store, indexer would figure out
// entity dependencies, regenerate all the corresponding documents and index
// them in the search engine. Thus, this method provides an automatic real
// time updating index.
//
// Method 2:
// Automatic dependency generation generally isn't complete. Some indexed
// documents might get stale, or might never be generated if their
// entities were never touched. This can be fixed by running a standalone
// indexing server, which would continuously loop over all the entities
// in the store, and run regeneration of docs and reindexing for all those
// docs in the search index. This provides a fool proof mechanism to keep
// store and search data in-sync.
//
// I recommend using both the methods. Method 1 ensures real time updates
// and method 2 ensures eventual consistency.
package indexer
