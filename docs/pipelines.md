# Data Pipelines

The goal of Fossil's data processing facilities is to allow essentially arbitrary data manipulation. 
It does this through it's own version of MapReduce, via the respective functions `filter`, `map`, and `reduce`. 
Each step of the pipeline can be chained together to outsource computation to the database.

## Processing Functions

A processing function in Fossil takes the form:

```
<name> <args> -> <expression>
```

Where the result of everything to the right of `->` is returned.

## Chaining Operator

Functions can be chained together using the pipe (`|`) operator. Each stage of the pipeline that is staged together 
"feeds" the next in the pipeline. Under the hood Fossil uses a Go `chan` to connect each stage to the next, 
allowing functions to operate in parallel.

## Filter

A filter function takes each input, and returns a boolean value of whether it should be accepted or rejected. 
An example that rejects all values < 50:

```
filter x -> x > 50
```

## Map

A map function takes each input, and maps it onto a new input. The output of this function need not have the 
same dimensions as the input. For example, a map can take a value, `x`, and map it into a "count" and the 
value "x" itself:

```
map x -> 1, x
```

We could later use this map to compute an average (more on that in the next section). Or, we could use a map to 
retrieve temperature data, but convert it to Celsius (assuming it's stored in Fahrenheit):

```
all in /sensors/temp since ~now - @day * 7 | map F -> 5/9 * (F-32)
```


## Reduce

A reduce function takes two values as input, and returns a single value as a result. This is a powerful abstraction 
that can be used to do all kinds of things. The constraint for a reduce function is that the outputs must be of the 
same type as the input.

In our example above, we mapped a value `x` onto a count (1), and the value itself. We could then use a reduce 
function to add everything up and then compute an average:

```
all in /latency | map x -> 1, x | reduce a, b -> a[0] + b[0], a[1] + b[1] | ⏎
                    map  count, sum -> sum / count
```

Number of log bytes written:

```
all in /logs | map log -> len(log) | reduce s_a, s_b -> s_a + s_b
```

Number of events in the last day:

```
all in /events since ~now - @day | map event -> 1 | reduce x, y -> x + y
```

