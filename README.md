
Project Structure
![img_1.png](img_1.png)

# How to Run

### Setup Dependencies:

### `go mod init job-scheduler-service`

### `go mod tidy`

### Start Infrastructure:

### `docker-compose up -d`

### Run the System:

#### go run . # Main scheduler + worker
#### OR
#### go run . -mode=worker # Worker only

Use the API:

### Schedule a job
`'curl -X POST http://localhost:8080/api/v1/jobs \
-H "Content-Type: application/json" \
-d '{"job_type":"email","payload":{"to":"test@example.com","subject":"Test","body":"Hello","from":"noreply@example.com"}}'
`
### Check queue stats
curl http://localhost:8080/api/v1/queues/stats


#### I've designed a comprehensive distributed job scheduling system using Hexagonal Architecture that addresses the requirements.
### Here are the key highlights:
## Architecture Highlights

* Separation of Concerns: Clear separation between scheduler, workers, and queue management
* Fault Tolerance: Workers can fail without affecting the system
* Scalability: Add more workers by simply running additional processes
* Observability: Complete job lifecycle tracking and metrics
* Business Flexibility: Easy to add new job types and handlers
* Tech Flexibility : Easy to swap or integrate new port for storage (Postgres, MongoDB)  and queue (Redis,RabbitMQ,SQS, KAKFA)
### Core Architecture Benefits
1. Clean Domain Logic: The Job, RetryPolicy, and Priority entities contain pure business logic with no infrastructure dependencies.
2. Flexible Adapters: Easy to swap Redis for SQS, memory storage for Postgres, or add new job handlers without touching core logic.
3. Production Ready: Includes proper error handling, metrics collection, graceful shutdown, and configurable retry policies with exponential backoff.
### Key Features Implemented
* Priority Queues: Critical → High → Medium → Low processing order
* Delayed Jobs: Schedule jobs for future execution with automatic promotion
* Retry Logic: Configurable retry policies with exponential backoff
* Dead Letter Queue: Failed jobs after exhausting retries
* Worker Management: Horizontal scaling with heartbeat monitoring
* Comprehensive Monitoring: Metrics and structured logging hooks

### Production Deployment
#### The system is designed to scale horizontally:

* Multiple scheduler instances for high availability
* Worker pools that can be scaled independently
* Queue-based job distribution for load balancing
* Database persistence for job state and history

🧪 Development Friendly
* Memory implementations for local development
* Clear interfaces make it easy to add new job types
* Comprehensive test examples included

#### The architecture follows Go idioms while maintaining clean separation between business logic and infrastructure concerns. 
#### We can start with the in-memory adapters for development and seamlessly switch to Redis/Postgres for production.

## High Level Architecture Diagram
![img.png](img.png)

## FLowchart Diagram
![img_2.png](img_2.png)

### Retry logic flow

![img_3.png](img_3.png)

### Database Schema

````-- PostgreSQL Schema (for production implementation)
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE jobs (
id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
type VARCHAR(100) NOT NULL,
priority INTEGER NOT NULL DEFAULT 2,
payload JSONB NOT NULL,
metadata JSONB DEFAULT '{}',
status VARCHAR(20) NOT NULL DEFAULT 'pending',
max_retries INTEGER NOT NULL DEFAULT 3,
current_retries INTEGER NOT NULL DEFAULT 0,
scheduled_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
executed_at TIMESTAMPTZ,
completed_at TIMESTAMPTZ,
worker_id VARCHAR(100),
error_message TEXT,
version INTEGER NOT NULL DEFAULT 1,
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX idx_jobs_status_priority ON jobs(status, priority);
CREATE INDEX idx_jobs_scheduled_at ON jobs(scheduled_at) WHERE status = 'pending';
CREATE INDEX idx_jobs_type ON jobs(type);
CREATE INDEX idx_jobs_worker_id ON jobs(worker_id) WHERE worker_id IS NOT NULL;

-- Job execution history
CREATE TABLE job_history (
id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
attempt_number INTEGER NOT NULL,
worker_id VARCHAR(100) NOT NULL,
started_at TIMESTAMPTZ NOT NULL,
completed_at TIMESTAMPTZ,
status VARCHAR(20) NOT NULL,
error_message TEXT,
execution_time_ms INTEGER,
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_job_history_job_id ON job_history(job_id);
CREATE INDEX idx_job_history_started_at ON job_history(started_at);

-- Workers table
CREATE TABLE workers (
id VARCHAR(100) PRIMARY KEY,
hostname VARCHAR(255) NOT NULL,
supported_job_types TEXT[] NOT NULL,
max_concurrent_jobs INTEGER NOT NULL DEFAULT 10,
current_jobs INTEGER NOT NULL DEFAULT 0,
status VARCHAR(20) NOT NULL DEFAULT 'active',
last_heartbeat TIMESTAMPTZ NOT NULL DEFAULT NOW(),
metadata JSONB DEFAULT '{}',
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_workers_status ON workers(status);
CREATE INDEX idx_workers_last_heartbeat ON workers(last_heartbeat);````