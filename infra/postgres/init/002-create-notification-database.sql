SELECT 'CREATE DATABASE notification_service'
WHERE NOT EXISTS (
    SELECT FROM pg_database WHERE datname = 'notification_service'
)\gexec

SELECT 'CREATE DATABASE external_integrations_service'
WHERE NOT EXISTS (
    SELECT FROM pg_database WHERE datname = 'external_integrations_service'
)\gexec
