# v0.1

This version marks the first release of the fossil time-series database.
Some release highlights:

  * Simple time-series database, conceptually based on the classic paper
    Birrell D. et al, "A Simple and Efficient Implementation for Small Databases".
  * Query parsing language
  * Server implementation
    * Support for multiple databases
    * Protocol
  * Client implementation
    * Command history
    * Tab completion
    * Local database access (no server)
  * Basic API for go clients