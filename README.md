# sqs-consumer-go

Disclaimer: at this time this project is mostly a case study for me to explore different concurrency patterns in golang while solving a common use case when running services in AWS.  This has not been tested in a production setting yet so use at your own risk.  Also please excuse any typos, writing README's and spelling are not skills I have mastered yet.

This is a project that came up when realizing that consuming messages off an SQS queue or any queue for that matter tends to be a lot of boiler plate just to get to a single "handle" function that does some business logic.  Similar to how AWS Lambda integrates with SQS, I wanted to mimic that behavior such that you can load this module in and with a few configurable options and a handler function and start consuming messages off SQS.  As part of this project, I also wanted to explore different golang patterns to handle concurrency while making a modular and extensible framework.

## Consumer
At the root of the packages is a "Consumer" which takes a handler function and some options.  Most options are available to fine tune how you want to process and consumer messages i.e., concurrency, heartbeat, etc 

A consumer internally needs a runner which is an interface that defines the root "process".  In this case we use an SqsRunner.  As the name implies the SqsRunner encapsulates the logic needed to poll an SQS queue. The Consumer.Start() func will basically start the runner until the Consumer.Stop() func is called to shut down the whole system.

## Runner
Conceptually the SqsRunner runs an infinite for loop to poll SQS for new messages. When it receives new message is calles to the "worker" which will handle the scheduling of the "handler" func.  The worker is responsible for maintaining the required level of concurrency and handling backpressure.  I've provided two types of workers: a semaphore worker and a worker pool.  The SqsRunner will also "listen" for the result of the worker and in this case will acknowledge the messages that have finished processing.

## Worker
The SqsWorker's main role is to take the batch of SQS messages we have received, and dispatches them to the worker pool.  As mentioned, the two provider worker pools are a semaphore implementation and a worker pool implementation.  Once the individual message has been processed, the worker will check if it returned an error and if it needs to be retried.  In the case it will not be retried, it sends the response to a haverster such that batches of messages can be acknowledged.  I'm not 100% convinced yet that batching really helps all that much in this case as SQS limites batches to sizes of 10 max, and doesn't really rate limit deleting messages.  So it might actually be simpler to implement a "haverstor" that doesn't do any batching and just processes the result of the message as it finishes.

### Semaphore Worker
The idea behind a semaphore worker pool is to use a weighted semaphore (almost like a ticketing system).  Every time a piece of work is dispatched, we try to aquired the semaphore while internally decrementing the semaphore count.  If the semaphore reaches its limit, the dispatch function effectively has to wait until the semaphore frees up.

NOTE: this particular implementation will spin up a new goroutine for every dispatch which can theoretically be less efficient than the worker pool.  A worker pool has the advantage of being able to reuse objects without always releasing them to the Garbage Collector.

### Worker Pool
The implementation of the worker pool was heavily inspired by this: https://github.com/alitto/pond.  The implementation holds a set amount of "workers" in memory that are all listening to a common channel ready to "do work".  However, it dynamically allocates workers as they are needed up to the max concurrency.  Once the max concurrency is met, it will actually start queuing the work in what I call a "waiting room".  This way under heavy load, we can store the backpressure in memory.  The main loop inside the worker pool implementation first tries to process anything in the waiting room before looking for new work.  Once the waiting room is cleared, it will start processing new work.  This repeats forever, until it is stopped.

NOTE: the waiting room is currently implemented using a slice/array, but when I get more time, I want to implement a ring buffer here as this will be a more efficient data structure.

NOTE: also a feature in alitto/pond I found interesting was the idea of shrinking the worker pool if the process is sitting idle.  Also baked into the main loop is idle-checking logic such that one by one workers are released, releasing resources, etc.  Once the process ramps back up as load increases, the workers will be respawned as needed. 
