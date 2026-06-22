SELECT 'CREATE DATABASE auth_service'
WHERE NOT EXISTS (
    SELECT FROM pg_database WHERE datname = 'auth_service'
)\gexec
