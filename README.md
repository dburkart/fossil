# Fossil

A small, efficient time-series database. See [Overview](./docs/overview.md) for a high-level overview on current design.

## Overview

Fossil is intended to be simple enough to use as a local time-series database (similar to a SQLite database) and robust
enough to perform in a distributed environment. If what you need to track is small pieces of data, ordered by time, then
this is the right database for you!

For more detailed documentation of the internals see the [docs](/docs) folder.

### Use Cases

Fossil solves the following use cases:

- Local Time Series Database that stores data in a local DB file
- Collect data over a TCP connection in fire and forget mode
- Query data over a TCP connection
