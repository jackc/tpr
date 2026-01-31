-- Create user 'tpr' if it doesn't exist
DO
$$
BEGIN
  IF NOT EXISTS (
    SELECT FROM pg_catalog.pg_roles
    WHERE rolname = 'tpr'
  ) THEN
    CREATE ROLE tpr WITH LOGIN PASSWORD 'password';
  END IF;
END
$$;

-- Create development database if it doesn't exist
SELECT 'CREATE DATABASE tpr_dev'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'tpr_dev')\gexec

-- Create test database if it doesn't exist
SELECT 'CREATE DATABASE tpr_test'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'tpr_test')\gexec
