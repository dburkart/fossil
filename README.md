# Fossil

A small, efficient time-series database. See [Overview](./docs/overview.md) for a high-level overview on current design.

- [Fossil](#fossil)
  - [Overview](#overview)
    - [Use Cases](#use-cases)
  - [How to](#how-to)
    - [Install](#install)
    - [Connect](#connecting-to-a-fossil-server)
      - [CLI](#using-the-cli)
      - [Programmatically](#programmatically)
    - [Server](#running-the-server)
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

### Connecting to a fossil server

#### Using the CLI

The below example connects to a local fossil server:

```shell
> fossil client -H fossil://localhost:8001
```

The fossil client supports sending commands to the server. For example queries, see [docs/cli.md](./docs/cli.md).

#### Programmatically

The main use-case for connecting to a fossil server programmatically is for appending data. This can
be done in only a few lines:

```go
import fossil "github.com/dburkart/fossil/api"

client, err := fossil.NewClient("fossil://localhost:8001")
if err != nil {
	panic(err)
}
// Append the data "Data" to the default topic "/"
client.Append("/", []byte("Data"))
```

### Running the server

```shell
> fossil server -h
Database for collecting and querying metrics

Usage:
  fossil server [flags]

Flags:
  -d, --database string   Path to store database files (default "./")
  -h, --help              help for server
  -p, --port int          Database server port for data collection (default 8001)
      --prom-port int     Set the port for /metrics (default 2112)

Global Flags:
  -c, --config string   Path to the fossil config file (default "./config.toml")
  -H, --host string     Host to send the messages (default "fossil://local/default")
      --local           Configures the logger to print readable logs (default true)
  -v, --verbose count   -v for debug logs (-vv for trace)
```

For documentation on deploying Fossil, see [deployment.md](./docs/deployment.md).

### Client / Server Config

```toml
[fossil]
port = 8000
prom-port = 2112

host = "fossil://localhost:8001/default"
local = true
verbose = 2

[database]
directory = "/mnt/e/data"

[database.test]
directory = "./data"

[database.tester]
```

#### Root `fossil` config block
| Option             | Default       | Description                                            |
| ------------------ |---------------| ------------------------------------------------------ |
| `fossil.port`      | 8001          | Port fossil server listens on                          |
| `fossil.prom-port` | 2112          | Port fossil server servers `/metrics` on               |
| `fossil.verbose`   | 0             | Configures the log level [0: info, 1: debug, 2: trace] |
| `fossil.host`      | `"./default"` | Connection string client will connect to               |
| `fossil.local`     | true          | Configures output logs to be in plaintext              |

####  `database` config block
The first database block without a `.<name>` applies to the `default` database. 
Any database block that contains a name identifier will configure the server 
to create a database with that name and configure it with the options contained
in the block. For example, the `database.tester` block in the toml config above
will automatically have the directory set to `./data/tester` as the location to
store the data for that logical database, where as the `database.test` block
has configured its data directory as `data/mydata`.

**Note:** If the only database directory set in the config file is on the default block, all databases will be created in that directory.

| Option               | Default | Description                                                                                   |
| -------------------- | ------- | --------------------------------------------------------------------------------------------- |
| `database.directory` | `"./"`  | Directory the sever uses to store the data for a logical database. This directory must exist. |
