# End-to-End Test: Scheduled Data Ingestion

**Objective:**
To verify that a data ingestion task can be scheduled via the metadata API and is automatically triggered by the Scheduler Service at the specified time, leading to data being processed and stored correctly.

**Test Scenario Document Structure:**

## 1. Prerequisites

*   **Latest Code:** Ensure you have the latest version of the codebase pulled from the repository.
*   **Docker and Docker Compose:** Docker and Docker Compose must be installed and running on your system.
*   **Sample Data Directory:** Create a directory named `sample_data` at the root of the project (i.e., next to the `docker-compose.yml` file).
*   **API Tool:** An API interaction tool like Postman, Insomnia, or `curl` is required to make HTTP requests to the API Gateway.

## 2. Step 1: Prepare Sample CSV Data

1.  Inside the `sample_data` directory, create a file named `scheduled_products.csv`.
2.  Add the following content to `sample_data/scheduled_products.csv`:

    ```csv
    product_id,product_name,price,category
    SCHED001,Laptop Pro,1250.99,Electronics
    SCHED002,Wireless Mouse,25.50,Electronics
    SCHED003,Coffee Beans Dark Roast,18.75,Groceries
    SCHED004,Desk Lamp LED,35.20,Home Goods
    SCHED005,Notebook Classic,7.99,Stationery
    ```

## 3. Step 2: Start All Backend Services

1.  Open a terminal at the root of the project.
2.  Run the command:
    ```bash
    docker-compose up --build -d
    ```
3.  Verify that all services start correctly, including `scheduler-service`. Check the status with:
    ```bash
    docker-compose ps
    ```
    All services should show a "running" or "healthy" state.

## 4. Step 3: Set Up Metadata via API Calls

Use your chosen API tool to make the following POST requests. The `BaseURL` for the API Gateway is `http://localhost:8080/api/v1`. Replace `{{...}}` placeholders with the actual IDs returned by the API in previous steps.

*   **A. Create Entity "ScheduledProduct"**
    *   **Method:** `POST`
    *   **URL:** `{{BaseURL}}/entities/`
    *   **Body (JSON):**
        ```json
        {
            "name": "ScheduledProduct",
            "description": "Products for scheduled CSV test"
        }
        ```
    *   **Action:** Note the `id` from the response. This will be your `{{TEST_PRODUCT_ENTITY_ID}}`.

*   **B. Create Attributes for "ScheduledProduct"**
    *   For each attribute below, make a `POST` request to `{{BaseURL}}/entities/{{TEST_PRODUCT_ENTITY_ID}}/attributes/`.
    *   **Attribute 1: `product_id`**
        *   **Body (JSON):**
            ```json
            {
                "name": "product_id",
                "data_type": "string",
                "description": "Unique identifier for the product from schedule",
                "is_filterable": true,
                "is_indexed": true
            }
            ```
        *   **Action:** Note the `id` from the response as `{{ATTR_ID_PRODID}}`.
    *   **Attribute 2: `product_name`**
        *   **Body (JSON):**
            ```json
            {
                "name": "product_name",
                "data_type": "string",
                "description": "Name of the scheduled product",
                "is_filterable": true
            }
            ```
        *   **Action:** Note the `id` from the response as `{{ATTR_ID_PRODNAME}}`.
    *   **Attribute 3: `price`**
        *   **Body (JSON):**
            ```json
            {
                "name": "price",
                "data_type": "float",
                "description": "Price of the scheduled product"
            }
            ```
        *   **Action:** Note the `id` from the response as `{{ATTR_ID_PRICE}}`.
    *   **Attribute 4: `category`**
        *   **Body (JSON):**
            ```json
            {
                "name": "category",
                "data_type": "string",
                "description": "Category of the scheduled product, to be lowercased"
            }
            ```
        *   **Action:** Note the `id` from the response as `{{ATTR_ID_CATEGORY}}`.

*   **C. Create DataSourceConfig for `scheduled_products.csv`**
    *   **Method:** `POST`
    *   **URL:** `{{BaseURL}}/datasources/`
    *   **Body (JSON):**
        ```json
        {
            "name": "CSVScheduledProducts",
            "type": "csv",
            "connection_details": "{\"filepath\":\"/app/csv_data_mount/scheduled_products.csv\"}",
            "entity_id": "{{TEST_PRODUCT_ENTITY_ID}}"
        }
        ```
    *   **Action:** Note the `id` from the response as `{{CSV_PRODUCTS_SOURCE_ID}}`.

*   **D. Create Field Mappings**
    *   For each mapping below, make a `POST` request to `{{BaseURL}}/datasources/{{CSV_PRODUCTS_SOURCE_ID}}/mappings/`.
    *   **Mapping 1: `product_id`**
        *   **Body (JSON):**
            ```json
            {
                "source_field_name": "product_id",
                "entity_id": "{{TEST_PRODUCT_ENTITY_ID}}",
                "attribute_id": "{{ATTR_ID_PRODID}}"
            }
            ```
    *   **Mapping 2: `product_name`**
        *   **Body (JSON):**
            ```json
            {
                "source_field_name": "product_name",
                "entity_id": "{{TEST_PRODUCT_ENTITY_ID}}",
                "attribute_id": "{{ATTR_ID_PRODNAME}}"
            }
            ```
    *   **Mapping 3: `price`**
        *   **Body (JSON):**
            ```json
            {
                "source_field_name": "price",
                "entity_id": "{{TEST_PRODUCT_ENTITY_ID}}",
                "attribute_id": "{{ATTR_ID_PRICE}}"
            }
            ```
    *   **Mapping 4: `category`**
        *   **Body (JSON):**
            ```json
            {
                "source_field_name": "category",
                "entity_id": "{{TEST_PRODUCT_ENTITY_ID}}",
                "attribute_id": "{{ATTR_ID_CATEGORY}}",
                "transformation_rule": "lowercase"
            }
            ```

*   **E. Create ScheduleDefinition**
    *   **Method:** `POST`
    *   **URL:** `{{BaseURL}}/schedules/`
    *   **Body (JSON):**
        ```json
        {
            "name": "Scheduled Product Ingestion Test",
            "description": "Ingests scheduled_products.csv every minute",
            "cron_expression": "*/1 * * * *",
            "task_type": "ingest_data_source",
            "task_parameters": "{\"source_id\":\"{{CSV_PRODUCTS_SOURCE_ID}}\"}",
            "is_enabled": true
        }
        ```
    *   **Action:** Note the `id` from the response as `{{SCHEDULE_ID}}`.

## 5. Step 4: Monitor and Observe

*   **Scheduler Service Logs:**
    *   Open a terminal and run: `docker-compose logs -f scheduler-service`
    *   Look for logs indicating that the service started and loaded the new schedule (e.g., "Loading schedules...", "Successfully added job for schedule ID {{SCHEDULE_ID}}...").
    *   Wait for the next minute mark. You should see logs similar to:
        `Executing scheduled task from Schedule ID: {{SCHEDULE_ID}} (Scheduled Product Ingestion Test) - Triggering ingestion for Source ID: {{CSV_PRODUCTS_SOURCE_ID}}`
        `Successfully triggered ingestion for Source ID {{CSV_PRODUCTS_SOURCE_ID}} (scheduled by {{SCHEDULE_ID}} ...)`

*   **Ingestion Service Logs:**
    *   Open another terminal and run: `docker-compose logs -f ingestion-service`
    *   Around the time the scheduler triggers the task, look for logs indicating it received an API call to trigger ingestion for `{{CSV_PRODUCTS_SOURCE_ID}}`.
    *   Expect logs like:
        `Starting ingestion for source ID: {{CSV_PRODUCTS_SOURCE_ID}}`
        `Starting CSV ingestion for source ID: {{CSV_PRODUCTS_SOURCE_ID}}. ConnectionDetails: {"filepath":"/app/csv_data_mount/scheduled_products.csv"}`
        `Successfully ingested 5 records for source ID: {{CSV_PRODUCTS_SOURCE_ID}} from CSV file /app/csv_data_mount/scheduled_products.csv.`
        `Sending 5 ingested records for SourceID {{CSV_PRODUCTS_SOURCE_ID}} (EntityID from DSConfig: '{{TEST_PRODUCT_ENTITY_ID}}') to processing service.`
        `Successfully sent data for source ID {{CSV_PRODUCTS_SOURCE_ID}} to processing service.`

*   **Processing Service Logs:**
    *   Open another terminal and run: `docker-compose logs -f processing-service`
    *   Look for logs indicating it received data from the ingestion service and processed it.
        `Processing data for sourceID: {{CSV_PRODUCTS_SOURCE_ID}}, entityTypeName: ScheduledProduct. Records received: 5` (or the name of your entity)
        `Successfully processed and stored 5 records for sourceID: {{CSV_PRODUCTS_SOURCE_ID}}, entityTypeName: ScheduledProduct`

## 6. Step 5: Verify Data in PostgreSQL

1.  Wait for at least a minute or two after the schedule was supposed to run to ensure data processing is complete.
2.  Connect to the PostgreSQL database:
    ```bash
    docker-compose exec postgres psql -U admin -d metadata_db
    ```
3.  Execute the following SQL query (replace `{{CSV_PRODUCTS_SOURCE_ID}}` with the actual ID):
    ```sql
    SELECT entity_definition_id, entity_type_name, source_id, attributes FROM processed_entities WHERE source_id = '{{CSV_PRODUCTS_SOURCE_ID}}';
    ```
4.  **Expected Results:**
    *   You should see 5 rows returned.
    *   `entity_definition_id` should match `{{TEST_PRODUCT_ENTITY_ID}}`.
    *   `entity_type_name` should be "ScheduledProduct" (or whatever name was used for the entity associated with `TEST_PRODUCT_ENTITY_ID` if the `DataSourceConfig.EntityID` was used directly as `EntityTypeName` by Ingestion/Processing services, which it is).
    *   `source_id` should match `{{CSV_PRODUCTS_SOURCE_ID}}`.
    *   The `attributes` JSONB column should contain the data from the CSV:
        *   For `SCHED001`: `{"price": 1250.99, "category": "electronics", "product_id": "SCHED001", "product_name": "Laptop Pro"}` (Note: `price` as a number, `category` as lowercase. The order of keys in JSON might vary).
        *   Similar correct JSON structures for the other 4 records.

## 7. Step 6: Disable Schedule (Optional Cleanup)

1.  To prevent the schedule from running every minute, disable it via the API.
    *   **Method:** `PUT`
    *   **URL:** `{{BaseURL}}/schedules/{{SCHEDULE_ID}}`
    *   **Body (JSON):**
        ```json
        {
            "name": "Scheduled Product Ingestion Test",
            "description": "Ingests scheduled_products.csv every minute",
            "cron_expression": "*/1 * * * *", 
            "task_type": "ingest_data_source",
            "task_parameters": "{\"source_id\":\"{{CSV_PRODUCTS_SOURCE_ID}}\"}",
            "is_enabled": false 
        }
        ```
2.  **Verification (Optional):**
    *   Observe the `scheduler-service` logs for the next few minutes. It should no longer log the execution of this specific task.
    *   (More advanced) If the scheduler reloads schedules periodically or on update, you might see a log indicating the schedule is now disabled or no longer being added.

This completes the manual end-to-end test scenario.
```
