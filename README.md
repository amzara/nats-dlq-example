smol experiment for me to learn how to implement a DLQ with nats.io 
idea: have a ticker that enters an entry to db every minute.
i will close port 5432 on local to test, and when the db conn fails (simulate real world failure)
i need the message to be entered into a retry queue to retry, and after a specific amount of failure, send to dead leter queue and do not try again

