# Query Grammar

```abnf
query           = quantifier [ identifier ] [ topic-selector ] [ time-predicate ] [ data-pipeline ]

; Quantifier
quantifier      = "all" / sample
sample          = "sample(" time-quantity ")"

; Identifier
identifier      = 1*(ALPHA / DIGIT)

; Topic selection
topic-selector  = "in" topic
topic           = "/" 1*(ALPHA / DIGIT / "/")

; Time
time-predicate  = ( "since" time-expression ) / ( "before" time-expression ) / 
                  ( "between" time-expression "," time-expression )
time-expression = ( time-whence ( "-" / "+" ) time-quantity ) / time-whence
time-whence     = "~now" / "~(" RFC3339 ")"
time-quantity   = time-term *( ( "-" / "+" ) time-term )
time-term       = time-atom *( "*" time-atom )
time-atom       = number / timespan
timespan        = "@second" / "@minute" / "@hour" / "@day" / "@week" / "@month" / "@year"

; Data Pipeline
data-pipeline   = 1*data-stage
data-stage      = ":" data-function
data-function   = ( "filter" / "map" / "reduce" ) data-args "->" expression
data-args       = identifier [ "," data-args ]

; Expressions
expression      = equality
equality        = comparison *( ( "!=" / "==" ) comparison )
comparison      = term *( ( ">" / ">=" / "<" / "<=" ) term )
term            = term_md *( ( "-" / "+" ) term_md )
term_md         = unary *( ( "/" / "*" ) unary )
unary           = ( "-" / "+" ) unary / primary
primary         = identifier / number / string / tuple / builtin

; Built in functions
builtin         = identifier "(" expression  ")"

; Data Types
number          = 1*DIGIT
string          = DQUOTE *ALPHA DQUOTE
tuple           = expression 1*( "," expression )
```

Simple Query Examples:

```
all in /visits since ~now - @day
sample(@minute) in /cpu-usage since @week
```

For more information on Data pipelines, see [data pipelines](./pipelines.md)
