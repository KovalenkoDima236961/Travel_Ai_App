Architecture
============

The service keeps the ASGI entry point, application wiring, HTTP routes, core
infrastructure helpers, schemas, and domain services separated.

Application Wiring
------------------

``app.main`` exposes the ASGI application. ``app.application`` creates the
FastAPI app, configures middleware, registers routes, and stores service
dependencies in ``app.state``.

HTTP Layer
----------

``app.api`` modules contain route handlers and FastAPI dependency accessors.
They translate HTTP requests into schema objects and delegate generation,
search, adaptation, and readiness work to lower layers.

Core Helpers
------------

``app.core`` contains shared infrastructure that is not tied to a specific
route, including exception handling, service-relative path resolution, and
readiness checks for configured dependencies.

Service Layer
-------------

``app.services`` contains generator orchestration, prompt construction,
response parsing, validation, local knowledge search, and template adaptation.
External systems such as Ollama and ChromaDB are accessed from this layer.

Schemas
-------

``app.schemas`` contains the request and response contracts used by Trip
Service and by the local development routes. Pydantic aliases preserve the
public JSON shape while Python code uses snake_case field names internally.
