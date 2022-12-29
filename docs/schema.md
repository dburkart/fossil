# Fossil Schemas

Fossil schemas are an optional feature which provide topic-level data consistency by
recording type information in the database. Additionally, defining a topic-level schema
will eventually allow additional processing within queries.

## Data Types

All data sent to fossil is stored as an opaque blob of bytes in the database in the end. By default,
the fossil client will interpret these bytes as a unicode string, which is why strings are the default
data type.

The complete list of types are:

* string
* binary
* boolean
* int8, int16, int32, int64
* float
* array
* composite

When creating a new topic, clients can choose to set a schema on the topic describing the kind of data expected to be
stored. All datum which do not conform to the schema will be rejected on append.

Some interesting notes on data types:

* There is no difference between `string` and `binary` in terms of how data is stored in the database.
  The difference is semantic; clients can choose to interpret the data differently depending on the type.
  For example, the fossil client will not display any data of type `binary`, since it could contain unprintable
  characters which have the potential to mess up a terminal.
* Arrays can only hold one type of data, and the data can only be fixed. `string`, `binary`, and `composite`
  are all variable length, so cannot be held in an array. Additionally, array length must be declared as part of
  the upfront schema.
* A composite is a combination of types that can be anything except a composite.

## Default Schema

Unless otherwise specified, all topics created will have the default schema of `string`

## Schemas in the topic hierarchy

If a given topic has a schema other than the default, all sub-topics must also conform
to that same schema. For example, suppose the following topics exist in the database:

```
/
/logs
/sensors
/sensors/temp
/sensors/temp/garage
/sensors/humidity
```

If we set a schema of `float` to the `/sensors/temp` topic, then all sub-topics will
also have that same schema enforced.

Setting a different schema on a sub-topic will result in an error.

## Schema Syntax

For string, boolean, int*, float, and array, they are simply defined as the name of the type itself:

| Type     | Syntax                    |
|----------|---------------------------|
| string   | string                    |
| binary   | binary                    |
| boolean  | boolean                   |
| int*     | int8, int16, int32, int64 |
| float    | float                     |
| array    | `[size]<fixed-type>`      |

A composite type has a syntax similar to a JSON object, except that values are types:

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

### Grammar

```abnf
schema      = type / array / shallow-map

type        = "string" / "binary" / fixed-type		  
fixed-type  = "boolean" / "int8" / "int16" / "int32" / "int64" /
              "uint8" / "uint16" / "uint32" / "uint64" / "float32" / "float64"
array       = "[" 1*DIGIT "]" fixed-type

composite   = "{" entries "}"
entries     = 1*entry
entry       = key ":" value ","
value       = type / array

key         = DQUOTE 1*( ALPHA / DIGIT / "_" / "-" ) DQUOTE
```