# Command Line Interface

Fossil provides an easy way to interact with fossil databases via the `fossil client` command.

## Commands

### LIST

The `list` command lists available databases or topics.

Example:
```
> list
default
events

> list topics
/
/logs
/measurements
```

### USE

The `use` command switches between databases

**Syntax**

`use <database>`

Example:
```
> use events
201 database changed
```

### APPEND

The `append` command appends new data to the specified topic in the current database.

**Syntax**

`append [<topic>] <data>`

If the topic is omitted, the default, "/" topic is used.

Example:
```
> append /logs something happened
200 Ok
```

### QUERY

The `query` commands queries the current database for data.

**Syntax**

`query <user-query>`

Where `<user-query>` is a query which conforms to the [query grammar](./grammar.md). 

*Note: data-predicates are not yet supported*

Some examples:

```
> query all
+-------------------------------------+--------------+----------------------+
|                TIME                 |    TOPIC     |         DATA         |
+-------------------------------------+--------------+----------------------+
| 2022-12-04T10:38:14.282027325-08:00 | /foo/bar/baz | blah                 |
| 2022-12-04T10:41:19.106400274-08:00 | /foo/bar     | baz                  |
| 2022-12-04T10:41:39.712533708-08:00 | /foo         | bar                  |
| 2022-12-06T09:47:48.51024114-08:00  | /foo         | new day, new message |
+-------------------------------------+--------------+----------------------+

> query all in /foo/bar
+-------------------------------------+--------------+------+
|                TIME                 |    TOPIC     | DATA |
+-------------------------------------+--------------+------+
| 2022-12-04T10:38:14.282027325-08:00 | /foo/bar/baz | blah |
| 2022-12-04T10:41:19.106400274-08:00 | /foo/bar     | baz  |
+-------------------------------------+--------------+------+

> query sample(@minute)
+-------------------------------------+--------------+----------------------+
|                TIME                 |    TOPIC     |         DATA         |
+-------------------------------------+--------------+----------------------+
| 2022-12-04T10:38:14.282027325-08:00 | /foo/bar/baz | blah                 |
| 2022-12-04T10:41:19.106400274-08:00 | /foo/bar     | baz                  |
| 2022-12-06T09:47:48.51024114-08:00  | /foo         | new day, new message |
+-------------------------------------+--------------+----------------------+
```

### STATS

The `stats` command returns stats on the running server + database.

**Syntax**

`stats [<database>]`

Example:

```
> stats
Allocated Heap: 784 kB
Total Memory: 18 MB
Uptime: 5h4m59.356606988s
Segments: 1
```