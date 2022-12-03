# Query Grammar

```abnf
query           = quantifier [ identifier ] [ topic-selector ] [ time-predicate ] [ data-predicate ]

; Quantifier
quantifier      = "all" / sample
sample          = "sample(" timespan ")"

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
time-term       = time-atom *( ( "/" / "*" ) time-atom )
time-atom       = number / timespan
timespan        = "@second" / "@minute" / "@hour" / "@day" / "@week" / "@month" / "@year"

; Data
data-predicate  = "->" data-expression
data-expression = term comparator term
term            = number / string / data-field
data-field      = identifier "." identifier
comparator      = "==" / "!=" / "<" / "<=" / ">" / ">="

; Data Types
number          = 1*DIGIT
string          = DQUOTE *ALPHA DQUOTE
```

Examples:

```
all in /visits from ~yesterday
all x in /clicks since @day * 30 -> x.target == "window"
sample(@minute) in /cpu-usage since @week
```

## Semantic rules not covered by the grammar

* A data-predicate is only valid as long as ALL of the following is true:
  1. A topic-selector has been specified
  2. The topic(s) in question have schemas specified
