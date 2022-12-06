# Protocol

## Wire Message

```
+--------+------------------+----------+
|   4    |        8         |  N - 12  |
+--------+------------------+----------+
|  len   |     Command      |   Data   |
+--------+------------------+----------+
```

The data portion is what the command handlers work on.

### Generic Response Messages

#### OkResponse
```
200 ok
+--------+----------------+
|   4    |       N        |
+--------+----------------+
| uint32 |    Message     |
+--------+----------------+
```
Ok can be replaced by a message

#### ERR
```
code string
500 error
+--------+----------------+
|   4    |       N        |
+--------+----------------+
| uint32 |  Err message   |
+--------+----------------+
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
```
Response
+--------+----------------+-----+----------------+
|   4    |       0        |     |       N        |
+--------+----------------+ ... +----------------+
| count  |     Entry      |     |     Entry      |
+--------+----------------+-----+----------------+

Entry
+--------+----------------+
|   4    |       N        |
+--------+----------------+
|  len   |      data      |
+--------+----------------+
```

### APPEND
#### AppendRequest
```
topic data
+--------+----------------+--------------+
|   4    |       N        |  4 + N -> M  |
+--------+----------------+--------------+
|  len   |     topic      |   Data...    |
+--------+----------------+--------------+
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
