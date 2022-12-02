# Fossil

A small, efficient time-series database. See [Overview](./docs/overview.md) for a high-level overview on current design.

- [Fossil](#fossil)
  - [Overview](#overview)
    - [Use Cases](#use-cases)
  - [How to](#how-to)
    - [Install](#install)
    - [Usage](#usage)
    - [Config](#config)
      - [Root `fossil` config block](#root-fossil-config-block)
      - [`database` config block](#database-config-block)


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

## How to

### Install
```shell
go install github.com/dburkart/fossil
```
or
```shell
git clone https://github.com/dburkart/fossil.git
cd fossil
go run main.go
```

### Usage
```shell
> fossil server -h
Database for collecting and querying metrics

Usage:
  fossil server [flags]

Flags:
  -d, --database string   Path to store database files (default "./")
  -h, --help              help for server
  -p, --port int          Database server port for data collection (default 8001)
      --prom-http int     Set the port for /metrics is bound to (default 2112)

Global Flags:
  -c, --config string   Path to the fossil config file (default "./config.toml")
  -H, --host string     Host to send the messages (default "fossil://local/default")
      --local           Configures the logger to print readable logs (default true)
  -v, --verbose count   -v for debug logs (-vv for trace)
```

### Config
```toml
[fossil]
port = 8000
prom-port = 2112

host = "fossil://local/default"
local = true
verbose = 2

[database]
directory = "data/default"

[database.test]
directory = "data/mydata"

[database.tester]
```

#### Root `fossil` config block
| Option             | Default                    | Description                                            |
| ------------------ | -------------------------- | ------------------------------------------------------ |
| `fossil.port`      | 8000                       | Port fossil server listens on                          |
| `fossil.prom-port` | 2112                       | Port fossil server servers `/metrics` on               |
| `fossil.verbose`   | 0                          | Configures the log level [0: info, 1: debug, 2: trace] |
| `fossil.host`      | `"fossil://local/default"` | Connection string client will connect to               |
| `fossil.local`     | true                       | Configures output logs to be in plaintext              |

####  `database` config block
The first database block without a `.<name>` applies to the `default` database. 
Any database block that contains a name identifier will configure the server 
to create a database with that name and configure it with the options contained
in the block. For example, the `database.tester` block in the toml config above
will automatically have the directory set to `./data/tester` as the location to
store the data for that logical database, where as the `database.test` block
has configured its data directory as `data/mydata`.

| Option               | Default           | Description                                                       |
| -------------------- | ----------------- | ----------------------------------------------------------------- |
| `database.directory` | `"./data/<name>"` | Directory the sever uses to store the data for a logical database |
