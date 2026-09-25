# Start with one modular application and a PostgreSQL outbox

Each server initially runs one modular Go application backed by PostgreSQL, with a delivery worker in the same application. A domain change and its outgoing delivery records commit in one transaction, allowing delivery to resume after process failure without a separate message broker. This keeps solo operation manageable while retaining durable work, at the cost of implementing retry, deduplication, and worker recovery correctly and sharing process resources between HTTP requests and background work.
