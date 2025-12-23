# Messaging App Frontend (SvelteKit)

This directory contains the SvelteKit frontend application for the Messaging App. It provides the user interface for interacting with the backend API.

## Technologies Used

*   **Framework:** SvelteKit
*   **Styling:** Tailwind CSS
*   **UI Components:** Custom components built with Tailwind CSS (and `bits-ui` for some primitives)
*   **Icons:** Lucide Svelte

## Local Development

To run the frontend application locally, you should typically start the entire Messaging App stack using the root `Makefile`.

1.  **Ensure Docker and Docker Compose are installed.**
2.  **Navigate to the project root directory (`messaging-app/`).**
3.  **Start all services (backend, database, Kafka, Redis, etc.):**
    ```sh
    make up
    ```
    This will also build and run the frontend development server.

Alternatively, if you only need to run the frontend development server (assuming the backend is already running elsewhere):

1.  **Navigate to this directory (`messaging-app/client/`).**
2.  **Install dependencies:**
    ```sh
    pnpm install
    ```
    (or `npm install` or `yarn install` if you prefer)
3.  **Start the development server:**
    ```sh
    pnpm run dev
    ```
    The frontend will typically be accessible at `http://localhost:5173`.

## Building for Production

To create a production-ready build of the frontend application:

1.  **Navigate to this directory (`messaging-app/client/`).**
2.  **Build the application:**
    ```sh
    pnpm run build
    ```
    The build output will be in the `build/` directory.

You can preview the production build with:
```sh
pnpm run preview
```

## Linting and Formatting

*   **Lint code:**
    ```sh
    pnpm run lint
    ```
*   **Format code:**
    ```sh
    pnpm run format
    ```

## Project Structure

(Optional: Add a brief overview of the `src/` directory structure if it adds significant value beyond what's obvious from the file system.)