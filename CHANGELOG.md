# v0.2.1

Fix one bug and one regression in v0.2.0.

  * Fix a panic in the type checker when creating a tuple from another tuple element.
  * Fix a regression with time expressions where the type checker is tripped needlessly.

# v0.2.0

This release adds powerful data processing capabilities to the database,
as well as many other improvements.

  * Adds data processing capabilities in the form of pipelines.
  * Adds support for data schemas on topics.
  * Adds prometheus metrics.
  * Enhanced REPL experience.
  * Improved on-disk data structure.
  * Auto-migration of database format.
  * More extensive testing.
  * Machine-readable client output.
  * Many other small improvements.

# v0.1.5

Adds a --version flag to make it easier to distinguish between different
versions of fossil.

# v0.1.4

One small enhancement, and a security fix around message size.

  * Switches the protocol to use BigEndian encoding for numbers, matching
    what we do on main.
  * Constrain individual protocol messages to be < 100 MiB in size. This
    limit will be configurable in the future.

# v0.1.3

Small bugfix release that fixes a couple of correctness issues.

  * Fixes a time-series ordering issue, since we weren't pulling our append
    timestamp after acquiring the write lock.
  * Fixes an issue where the write-ahead log would not be applied on large
    enough databases.
  * Enhances the client API to try reconnecting to a lost fossil server.

# v0.1.2

Small bugfix release with a fix for a regression, as well as a stability fix.

  * Fixes a regression in v0.1.1 that would prevent a database with more
    than 10,000 elements from being read from disk.
  * Fixes a silent server crash when spilling a database to disk during
    a high volume of concurrent appends.

# v0.1.1

Small bugfix release with reliability and quality-of-life fixes.

  * Fixes a bug where the append command with no arguments caused the
    server to panic.
  * Fixes a panic in the parser caused by an empty query.
  * Allow some non-letter, non-digit characters in topics.
  * Fixes a concurrency crash around unsafe use of a map.
  * Adds autocomplete filtering.

# v0.1

This version marks the first release of the fossil time-series database.
Some release highlights:

  * Simple time-series database, conceptually based on the classic paper
    Birrell D. et al, "A Simple and Efficient Implementation for Small Databases".
  * Query parsing language
  * Server implementation
    * Support for multiple databases
    * Protocol
  * Client implementation
    * Command history
    * Tab completion
    * Local database access (no server)
  * Basic API for go clients
