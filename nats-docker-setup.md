## NATS Setup for Development with Docker

To run a NATS server with JetStream enabled for development, you can use the following Docker command or `docker-compose.yml` snippet.

### Using `docker-compose.yml`

Create a `docker-compose.yml` file in your project root (or a dedicated directory for infrastructure) with the following content:

```yaml
version: '3.7'
services:
  nats:
    image: nats:2.10-alpine # Use a version that includes JetStream, e.g., 2.10 or later
    ports:
      - "4222:4222" # Client port
      - "6222:6222" # Optional: NATS clustering port
      - "8222:8222" # HTTP management port (for monitoring, etc.)
    command: "-js"  # Enable JetStream
    # To persist JetStream data, you can mount a volume:
    # volumes:
    #   - ./nats-data:/data # Mounts a local directory 'nats-data' into the container's /data
    # command: "-js -sd /data" # Enable JetStream and specify storage directory
```

Then run `docker-compose up -d nats` from the directory containing the `docker-compose.yml` file.

### Using Docker Run Command

Alternatively, you can run NATS directly using Docker:

```bash
docker run -d --name nats-server \
  -p 4222:4222 \
  -p 6222:6222 \
  -p 8222:8222 \
  nats:2.10-alpine -js
```

To persist JetStream data with `docker run`:

```bash
docker run -d --name nats-server \
  -p 4222:4222 \
  -p 6222:6222 \
  -p 8222:8222 \
  -v $(pwd)/nats-data:/data \
  nats:2.10-alpine -js -sd /data
```
This will create a `nats-data` directory in your current working directory on the host.

### Streams and Subjects

NATS JetStream streams and subjects can be:
1.  **Dynamically Created**: Applications (publishers or subscribers) can create streams and consumers on-demand if they have the appropriate permissions and the NATS server configuration allows it.
2.  **Pre-configured**: Streams can be defined administratively using the NATS CLI or other management tools before applications connect.

For this project, we will assume that the Orchestration Service (as a publisher) will dynamically create streams if they don't exist, or publish to pre-existing streams. Action Executor services (as subscribers) will typically create durable consumers on these streams.

**Key NATS concepts used:**
*   **Subjects**: The topic name to which messages are published (e.g., `actions.webhook`, `actions.email`).
*   **Streams**: Persistent storage for messages published to subjects. JetStream streams capture messages from one or more subjects.
*   **Consumers**: How subscribers read messages from a stream. Durable consumers allow subscribers to pick up where they left off.

The Orchestration Service will publish tasks to subjects like `actions.<ActionType>`. Action Executor services will subscribe to these subjects, usually via a stream that captures these subjects.
The stream name could be, for example, `ACTION_TASKS` and it could capture subjects `actions.>`.
Each Action Executor would then create a durable consumer on this stream for its specific action type (e.g., a "webhook-executor" consumer for subject `actions.webhook`).
This setup ensures that even if no executor is running when a task is published, the task is persisted by JetStream and will be processed once an executor comes online.
