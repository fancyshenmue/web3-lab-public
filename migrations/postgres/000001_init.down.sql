-- Rollback: drop all tables in reverse dependency order

DROP TABLE IF EXISTS account_audit_logs;
DROP TABLE IF EXISTS account_merge_history;
DROP TABLE IF EXISTS account_sessions;
DROP TABLE IF EXISTS account_identities;
DROP TABLE IF EXISTS identity_providers;
DROP TABLE IF EXISTS accounts;

DROP EXTENSION IF EXISTS "pgcrypto";
