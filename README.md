# queuectl - A CLI-based Background Job Queue System

`queuectl` is a minimal, production-grade job queue system built in Go. It manages background jobs with worker processes, handles retries using exponential backoff, and maintains a Dead Letter Queue (DLQ) for permanently failed jobs.

## Features

- **Enqueue & Manage Jobs**: Add jobs with simple commands.
- **Persistent Storage**: Job data persists across restarts using an embedded SQLite database.
- **Multiple Workers**: Process jobs in parallel with multiple worker processes.
- **Automatic Retries**: Failed jobs are automatically retried with configurable exponential backoff.
- **Dead Letter Queue (DLQ)**: Jobs that exhaust all retries are moved to a DLQ for manual inspection or retry.
- **Graceful Shutdown**: Workers finish their current job before exiting.
- **CLI Interface**: All operations are accessible through a clean and simple CLI.

---

## Architecture Overview

### Job Lifecycle

A job progresses through the following states:

1.  **`pending`**: The initial state. The job is waiting in the queue to be picked up by a worker.
2.  **`processing`**: A worker has picked up the job and is executing its command.
3.  **`completed`**: The job's command executed successfully (exit code 0).
4.  **`failed`**: The job's command failed (non-zero exit code). It will be retried.
5.  **`dead`**: The job has failed `max_retries` times and has been moved to the Dead Letter Queue.

### Data Persistence

-   **SQLite**: Job data is stored in a single file database (`~/.queuectl/jobs.db`). SQLite was chosen for its transactional integrity (ACID compliance) and ability to handle concurrent access from multiple workers without data corruption, which is a significant advantage over a simple JSON file.

### Worker Logic

-   **Concurrency**: The `worker start --count N` command launches a manager process that spawns `N` worker goroutines.
-   **Job Locking**: To prevent multiple workers from processing the same job, a worker locks a job by selecting it and updating its state to `processing` within a single database transaction. This ensures atomicity.
-   **Graceful Shutdown**: When `worker stop` is called, a `SIGTERM` signal is sent to the manager process. The manager propagates a shutdown signal to all workers, which allows them to finish their current job before exiting.

---

## Setup & Installation

### Prerequisites

-   Go 1.18 or later

### Build from Source

1.  **Clone the repository:**
    ```sh
    git clone https://github.com/your-username/queuectl.git
    cd queuectl
    ```

2.  **Build the binary:**
    ```sh
    go build -o queuectl .
    ```

3.  **Install it (optional):**
    Move the binary to a directory in your `PATH` to make it available system-wide.
    ```sh
    sudo mv queuectl /usr/local/bin/
    ```

---

## Usage Examples

### 1. Enqueue a Job

Add a new job to the queue. The `id` is optional; one will be generated if omitted.

```sh
# Enqueue a simple job
queuectl enqueue '{"command":"echo Hello World"}'
# > Successfully enqueued job with ID: 1a2b3c4d-....

# Enqueue a job that will take some time
queuectl enqueue '{"id":"job-sleep-5", "command":"sleep 5 && echo Done sleeping"}'
# > Successfully enqueued job with ID: job-sleep-5
```

### 2. Start Workers

Start worker processes in the background. The command will run as a daemon.

```sh
# Start 3 workers
queuectl worker start --count 3
# > Starting 3 workers...
# (Logs will show worker activity)
```
*You can view the logs in your terminal. For a real daemon, you would redirect output to a log file.*

### 3. Check Status

Get a summary of job states and worker status.

```sh
queuectl status
# > Job Status Summary:
# > +------------+-------+
# > |   STATE    | COUNT |
# > +------------+-------+
# > | pending    |     0 |
# > | processing |     1 |
# > | completed  |     1 |
# > | failed     |     0 |
# > | dead       |     0 |
# > +------------+-------+
# >
# > Worker Status:
# > Workers are running.
```

### 4. List Jobs

List jobs in a specific state.

```sh
queuectl list --state completed
# > +-------------+---------------------------+----------+---------------------+---------------------+
# > |     ID      |          COMMAND          | ATTEMPTS |     CREATED AT      |     UPDATED AT      |
# > +-------------+---------------------------+----------+---------------------+---------------------+
# > | 1a2b3c4d... | echo Hello World          |        1 | 2023-10-27 10:30:00 | 2023-10-27 10:30:01 |
# > +-------------+---------------------------+----------+---------------------+---------------------+
```

### 5. Handling Failures (Retry & DLQ)

Let's enqueue a job that is guaranteed to fail.

```sh
# Configure max-retries to 2 for this demo
queuectl config set max-retries 2

# Enqueue a failing job
queuectl enqueue '{"id":"failing-job", "command":"exit 1"}'
```

After starting the workers, you will see logs indicating retries with exponential backoff. After 2 attempts, it will be moved to the DLQ.

```sh
# Check the DLQ
queuectl dlq list
# > +-------------+-----------+----------+---------------------+---------------------+
# > |     ID      |  COMMAND  | ATTEMPTS |     CREATED AT      |     UPDATED AT      |
# > +-------------+-----------+----------+---------------------+---------------------+
# > | failing-job | exit 1    |        2 | 2023-10-27 10:35:00 | 2023-10-27 10:35:05 |
# > +-------------+-----------+----------+---------------------+---------------------+

# Retry the job from the DLQ
queuectl dlq retry failing-job
# > Job failing-job has been moved from DLQ back to the pending queue.
```

### 6. Stop Workers

Stop the worker manager process gracefully.

```sh
queuectl worker stop
# > Sending SIGTERM to worker process with PID 12345
# > Stop signal sent. Workers should shut down shortly.
```

### 7. Configuration

Manage settings like max retries and backoff base.

```sh
# Set the default max retries for new jobs to 5
queuectl config set max-retries 5

# Set the exponential backoff base (delay = base ^ attempts)
queuectl config set backoff-base 3
```

---

## Testing & Validation

To validate the core flows, you can run the following sequence of commands in your terminal.

1.  **Clean Slate**: Remove any previous database or config.
    ```sh
    rm -rf ~/.queuectl
    ```

2.  **Enqueue Jobs**:
    ```sh
    queuectl enqueue '{"id":"job1", "command":"echo job 1 complete"}'
    queuectl enqueue '{"id":"job2", "command":"sleep 3 && echo job 2 complete"}'
    queuectl enqueue '{"id":"job3-fail", "command":"non_existent_command"}'
    ```

3.  **Check Initial Status**:
    ```sh
    queuectl status
    # Should show 3 pending jobs and no workers.
    ```

4.  **Start Workers**:
    Open a new terminal window for this command to see the logs.
    ```sh
    queuectl worker start --count 2
    ```
    *You will see `job1` and `job2` get processed. `job3-fail` will start failing and retrying.*

5.  **Check Status During Processing**:
    In the original terminal:
    ```sh
    queuectl status
    # Should show jobs moving from pending to processing/completed/failed.
    ```

6.  **Wait for DLQ**:
    After a short while, `job3-fail` will exhaust its retries (default 3) and move to the DLQ.

7.  **Check Final State**:
    ```sh
    queuectl list --state completed
    queuectl dlq list
    ```

8.  **Stop Workers**:
    ```sh
    queuectl worker stop
    ```

This script validates successful completion, parallel processing, failure handling, retries, and the DLQ mechanism.
