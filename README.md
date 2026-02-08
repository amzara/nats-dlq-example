smol experiment for me to learn how to implement a DLQ with nats.io 
idea: have a ticker that enters an entry to db every minute.
i will close port 5432 on local to test, and when the db conn fails (simulate real world failure)
i need the message to be entered into a retry queue to retry, and after a specific amount of failure, send to dead leter queue and do not try again

//add pgx connection pooling first, then retries
//keep the things alive indefinitely (the inserts are just events)
//every minute print out the items in the consumer in a table list

//next implementation steps
1) comment the db insert first, get the pub/sub going w/o goroutines
2) next goal, retries.
todo: fire 25 message, A-Z
