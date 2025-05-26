# Full Stack Platform Application

This project includes a Go backend (`metadata-service`), a PostgreSQL database, an Nginx API Gateway, and a Vue.js frontend (`platform-ui`). This README provides instructions for running and testing the entire application stack using Docker Compose.

## Prerequisites

- Docker installed and running on your system.
- Docker Compose installed (usually comes with Docker Desktop).
- A web browser.
- Ports `8000`, `8888`, and `5432` should be available on your host machine, or you can adjust the port mappings in `docker-compose.yml`.

## Directory Structure Overview

```
/
├── docker-compose.yml       # Main Docker Compose file for orchestration
├── metadata-service/        # Go backend service
│   ├── Dockerfile
│   ├── db/migrations/init.sql # SQL schema (GORM AutoMigrate is used primarily)
│   └── ... (Go source files)
├── nginx_gateway/           # Nginx API Gateway configuration
│   ├── Dockerfile
│   └── conf.d/default.conf  # Nginx config for API gateway
├── platform-ui/             # Vue.js frontend application
│   ├── Dockerfile           # Dockerfile for building and serving the Vue app with Nginx
│   ├── nginx.conf           # Nginx config for serving the Vue SPA
│   ├── .env.development     # Contains VUE_APP_API_BASE_URL for local dev
│   └── ... (Vue source files)
└── pgdata/                  # (Will be created by Docker Compose) PostgreSQL data persistence
└── README.md                # This file
```

## Running the Application

1.  **Clone the Repository (if applicable):**
    If you have this project as a Git repository, clone it first. These instructions assume you are at the root of the project where `docker-compose.yml` is located.

2.  **Build and Start Services:**
    Open a terminal at the root of the project and run:
    ```bash
    docker-compose up --build -d
    ```
    *   `--build`: Forces Docker Compose to build the images from the Dockerfiles (important for the first run or after code changes).
    *   `-d`: Runs the containers in detached mode (in the background).

    This command will:
    *   Pull the `postgres:13-alpine` image.
    *   Build Docker images for `metadata-service`, `api-gateway`, and `frontend`.
    *   Create and start containers for all four services.
    *   Create a Docker network named `app-network` (or similar, based on project directory name) for inter-service communication.
    *   Create a Docker volume named `pgdata` for PostgreSQL data persistence.

3.  **Check Container Status:**
    You can check the status of the running containers with:
    ```bash
    docker-compose ps
    ```
    All services should show as "Up" or "running". You can also view logs for a specific service:
    ```bash
    docker-compose logs -f <service_name>  # e.g., docker-compose logs -f metadata-service
    ```

## End-to-End Testing Steps

1.  **Access the Frontend UI:**
    Open your web browser and navigate to:
    [http://localhost:8000](http://localhost:8000)
    You should see the `platform-ui` homepage with a navigation bar.

2.  **Verify Database Migrations (`metadata-service`):**
    *   The `metadata-service` uses GORM's `AutoMigrate` feature. When it starts and successfully connects to PostgreSQL, it should create the `entity_definitions` and `attribute_definitions` tables if they don't exist.
    *   You can check the logs of the `metadata-service` for messages indicating database connection and migration:
        ```bash
        docker-compose logs metadata-service
        ```
        Look for lines like "Database connection established" and "Database schema migration completed."
    *   (Optional) To directly verify table creation in PostgreSQL:
        You can connect to the PostgreSQL database using a tool like `psql` or any GUI client. From your host machine (if you mapped port 5432):
        ```bash
        # Example using psql (if installed locally)
        psql -h localhost -p 5432 -U platform_user -d platform_db
        # Password will be "platform_password" (from docker-compose.yml)
        ```
        Once connected, you can list tables with `\dt` or query a specific table:
        ```sql
        SELECT * FROM entity_definitions;
        ```

3.  **Perform CRUD Operations for "Entities":**
    *   In the `platform-ui` web application ([http://localhost:8000](http://localhost:8000)), click on the "**Entities**" link in the navigation bar. This will take you to the Entity Definitions management page.
    *   **Create a New Entity:**
        *   Click the "**Add New Entity**" button.
        *   A dialog form will appear. Enter a **Name** (e.g., "Customer") and an optional **Description**.
        *   Click "**Save**".
        *   You should see a success notification, and the new entity should appear in the table below.
    *   **View the List of Entities:**
        *   The table on the Entities page should display the newly created entity and any others you add.
        *   Verify that the "Created At" and "Updated At" timestamps are populated.
    *   **Edit an Existing Entity:**
        *   In the table, find the entity you created and click the "Edit" icon (pencil) in its row.
        *   The form dialog will appear, pre-filled with the entity's data.
        *   Modify the Name or Description.
        *   Click "**Save**".
        *   You should see a success notification, and the entity's details in the table should update.
    *   **Delete an Entity:**
        *   In the table, find an entity and click the "Delete" icon (trash can) in its row.
        *   A confirmation dialog will appear. Click "**Delete**".
        *   You should see a success notification, and the entity should be removed from the table.

4.  **Confirm API Gateway and Data Persistence:**
    *   **API Gateway:**
        *   Open your browser's Developer Tools (usually F12) and go to the "Network" tab.
        *   Perform any CRUD operation (e.g., add a new entity or refresh the list).
        *   Observe the XHR/fetch requests. The "Request URL" should be to `http://localhost:8888/api/v1/entities/...`. This confirms that the frontend is sending requests to the Nginx API Gateway.
    *   **Data Persistence:**
        *   Add a few entities.
        *   Refresh the browser page (F5 or Ctrl+R/Cmd+R). The entities should still be listed, demonstrating that the data is fetched from the backend and persisted in the PostgreSQL database.
        *   You can also try stopping and restarting the application stack (see next step) and then verify that the data is still present.

## Stopping the Application

1.  **Stop and Remove Containers, Networks, and Volumes:**
    To stop all running services and remove the containers, networks, and the `pgdata` volume (which means **database data will be lost** if you remove the volume), run:
    ```bash
    docker-compose down -v
    ```
    If you want to stop the services but **keep the `pgdata` volume** (so data persists across restarts), use:
    ```bash
    docker-compose down
    ```
    You can then start it again with `docker-compose up -d` and the data will still be there.

## Further Development

-   To modify backend code (`metadata-service`), make your changes and then rebuild/restart that specific service:
    ```bash
    docker-compose up --build -d metadata-service
    ```
-   To modify frontend code (`platform-ui`), make your changes. The frontend Docker image will need to be rebuilt:
    ```bash
    docker-compose up --build -d frontend
    ```
    (Note: For a faster frontend development workflow, you might run `npm run serve` directly on your host machine during development, pointing its API calls to `http://localhost:8888`, and only use the Dockerized frontend for integration testing or production-like builds.)

