# Overview

This document tries to detail the current thinking / design behind fossil. Most sections are WIP and are most likely
unimplemented and still in the design phase.

## Architecture

### Database

A fossil database essentially behaves as an unbounded array. It is divided up into *segments*, each of which is of a 
fixed width and is made up of *datum*. Because Fossil is a time-series database, all datum are ordered sequentially. 
For this reason, timestamps are only recorded in each segment data structure, and each datum only stores an offset in 
nanoseconds from the segment timestamp.

This means that the overhead for storing data in fossil is quite low, which is useful depending on what kind of data is 
being stored in the database.

In addition to the time and data that is being stored in each segment, datum can belong to a particular *topic*. The 
topic is the primary way of indexing into the data (besides time, of course). Because a topic id is stored in each 
datum, it is trivial to filter on a particular topic, or group of topics.

### Server

Fossil server uses a TCP socket to listen for client connections. Clients can act in two modes: 
- **Fire and Forget** for data collection
- **Active** for data collection and querying

Connections are handled using goroutines and a message stream channel. Once the wire messages are parsed and sent over the stream the server beings to processed queued messages against the database.

```mermaid
graph TB;
	server
	processMessageStream[go processMessageStream]
	listen --> accept
	accept --> accept
	accept -. collector.New .-> handle[go handle];
	server --> listen
	server --> processMessageStream
	processMessageStream --> processMessageStream
	handle -.msgStream .-> server

	processMessageStream --> database
```

#### Fire and Forget
Clients in Fire and Forget mode connected to the server are limited to write only commands from the command list to ensure performant writes.

#### Active
Clients connected in active mode are open to use all commands, read and write, on the database.

## Data Types (WIP)

By default, data sent to fossil is stored as an opaque blob of bytes in the database. This unstructured data can not 
be introspected by the database, and thus is not indexable or searchable.

In addition to unstructured data, Fossil supports the following data types:

* string
* boolean
* int8, int16, int32, int64
* float
* array
* shallow map

When creating a new topic, clients can choose to set a schema on the topic describing the kind of data expected to be 
stored. All datum which do not conform to the schema will have their topic overriden with the special "nonconformant" 
topic.

Of the above data types, array and shallow map merits some discussion. The array data type must be declared to contain 
only a single type of data, and be of a particular size, i.e. `[12]string`, `[3]int`, etc.

The shallow map is a special type of map which can contain any data type except a shallow map (hence the name).  Each 
field in a shallow map has a label, defined in the schema.

### Schema Syntax (WIP)

For string, boolean, int*, float, and array, they are simply defined as the name of the type itself:

| Type    | Syntax                    |
| ------- | ------------------------- |
| string  | string                    |
| boolean | boolean                   |
| int*    | int8, int16, int32, int64 |
| float   | float                     |
| array   | `[size]<type>`            |

A shallow map has a syntax similar to a JSON object, except that values are types:

```
{
	"coordinates": [2]int32,
	"action": string,
}
```

When a topic is created with a particular schema, the schema is added to the topic map. All incoming data is then 
validated against the schema, and then packed into a datum object. Due to the overhead of creating and maintaining 
topics with schemas, they should only be used if absolutely necessary; i.e. the data itself needs to be introspected 
in some way.

## Queries (WIP)

Queries take the form of a predicate:

```
QUANTIFIER [IDENTIFIER] [in TOPIC] [(from|since|between) PREDICATE] [-> DATA PREDICATE]

Examples:

all in #visits from @yesterday
all x in #clicks since @day * 30 -> x.target == "window"
sample(@minute) in #cpu-usage since @week
```
