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