# Protocol

## Wire Message

```
+--------+------------------+----------+
|   4    |        8         |  N - 12  |
+--------+-----------------------------+
|  len   |     Command      |   Data   |
+--------+-----------------------------+
```

The data portion is what the command handlers work on.

### Generic Response Messages

#### OK
```
200 ok
```
Ok can be replaced by a message

#### ERR
```
code string
```
code is an integer number for the given code. A code of 0 means this is a custom error message.

### USE
#### UseRequest
```
dbName
```
The name of the database to switch too.

#### UseResponse
Generic Ok

### QUERY
#### QueryRequest
```
string
```
Query is just a string extracted from the data segment.

#### QueryResponse
TODO

### APPEND
#### AppendRequest
```
topic data
```
Append is sent in two parts. Topic is the path for this data item.

#### AppendResponse 
See generic Ok

### STATS
#### StatsRequest
```
dbfilter
```
Filter for db name. Defaults to the current database.

#### StatsResponse
TODO
