# Query Grammar

```abnf
query = quantifier [ identifier ] [ topic-selector ] [ time-predicate ] [ data-predicate ]

; Quantifier
quantifier = "all" / sample
sample = "sample(" timespan ")"

; Identifier
identifier = 1*(ALPHA / DIGIT)

; Topic selection
topic-selector = "in" (topic / "(" topic-list ")" )
topic-list = topic ["," topic-list]
topic = "#" 1*(ALPHA / DIGIT)

; Time
time-predicate = ( "since" time-expression ) / ( "until" time-expression ) / ( "between" time-expression ".." time-expression )
time-whence = "+now" / "+begin" / "+" iso8601
time-quantity = timespan / time-quantity ( "*" / "+" / "-" ) number
timespan = "@second" / "@minute" / "@hour" / "@day" / "@week" / "@month" / "@year"
time-expression = ( time-whence ( "-" / "+" ) time-quantity ) / time-quantity

; Data
data-predicate = "->" data-expression
data-expression = term comparator term
term = number / string / data-field
data-field = identifier "." identifier
comparator = "==" / "!=" / "<" / "<=" / ">" / ">="

; Data Types
number = 1*DIGIT
string = DQUOTE *ALPHA DQUOTE
```

Examples:

```
all in #visits from @yesterday
all x in #clicks since @day * 30 -> x.target == "window"
sample(@minute) in #cpu-usage since @week
```