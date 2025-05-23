# Data Ingestion Service

This service is responsible for handling data ingestion pipelines.

## Current Status

This is currently a stub implementation (Phase 2). It provides a basic HTTP server with a placeholder endpoint to trigger ingestion for a given data source ID.

The actual ingestion logic (connecting to data sources, fetching data, transforming, and storing it) is not yet implemented.

## Endpoints

*   **`POST /api/v1/ingest/trigger/{source_id}`**:
    *   Accepts a `source_id` path parameter.
    *   Currently returns a `501 Not Implemented` status, indicating that the ingestion logic is pending.
    *   Logs the received trigger request.

## Future Development

*   Implementation of actual data fetching logic for different data source types (e.g., PostgreSQL, CSV, API).
*   Integration with the Metadata Service to retrieve data source configurations and field mappings.
*   Data transformation capabilities based on mapping rules.
*   Storing ingested data into a designated data store (details TBD).
*   Error handling and reporting for ingestion jobs.
*   Potentially, a system for managing and monitoring ingestion jobs.
