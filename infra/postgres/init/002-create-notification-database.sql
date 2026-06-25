SELECT 'CREATE DATABASE notification_service'
WHERE NOT EXISTS (
    SELECT FROM pg_database WHERE datname = 'notification_service'
)\gexec
