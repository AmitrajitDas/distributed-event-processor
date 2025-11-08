-- Initialize Event Processor Database
-- This script runs when PostgreSQL container starts for the first time

-- Create database if not exists (this is already done by POSTGRES_DB env var)
-- CREATE DATABASE IF NOT EXISTS event_processor;

-- Create event_processor user (this is already done by POSTGRES_USER env var)
-- CREATE USER IF NOT EXISTS eventuser WITH PASSWORD 'eventpass';

-- Switch to event_processor database
\c event_processor;

-- Grant privileges
GRANT ALL PRIVILEGES ON DATABASE event_processor TO eventuser;
GRANT ALL PRIVILEGES ON SCHEMA public TO eventuser;

-- Create tables for event storage
CREATE TABLE IF NOT EXISTS events (
    id BIGSERIAL PRIMARY KEY,
    event_id VARCHAR(255) UNIQUE NOT NULL,
    tenant_id VARCHAR(255),
    event_type VARCHAR(255) NOT NULL,
    event_data JSONB NOT NULL,
    event_timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    processing_timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    partition_key VARCHAR(255),
    offset_info JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create index for performance
CREATE INDEX IF NOT EXISTS idx_events_tenant_id ON events(tenant_id);
CREATE INDEX IF NOT EXISTS idx_events_type ON events(event_type);
CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events(event_timestamp);
CREATE INDEX IF NOT EXISTS idx_events_processing_timestamp ON events(processing_timestamp);
CREATE INDEX IF NOT EXISTS idx_events_partition_key ON events(partition_key);

-- Create table for processed results
CREATE TABLE IF NOT EXISTS processed_events (
    id BIGSERIAL PRIMARY KEY,
    original_event_id VARCHAR(255) NOT NULL,
    tenant_id VARCHAR(255),
    rule_id VARCHAR(255),
    result_type VARCHAR(255) NOT NULL,
    result_data JSONB NOT NULL,
    processing_timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for processed events
CREATE INDEX IF NOT EXISTS idx_processed_events_original_id ON processed_events(original_event_id);
CREATE INDEX IF NOT EXISTS idx_processed_events_tenant_id ON processed_events(tenant_id);
CREATE INDEX IF NOT EXISTS idx_processed_events_rule_id ON processed_events(rule_id);
CREATE INDEX IF NOT EXISTS idx_processed_events_type ON processed_events(result_type);
CREATE INDEX IF NOT EXISTS idx_processed_events_timestamp ON processed_events(processing_timestamp);

-- Create table for schemas (used by Schema Registry)
CREATE TABLE IF NOT EXISTS schemas (
    id BIGSERIAL PRIMARY KEY,
    schema_name VARCHAR(255) NOT NULL,
    schema_version INTEGER NOT NULL,
    schema_type VARCHAR(50) NOT NULL DEFAULT 'AVRO',
    schema_definition TEXT NOT NULL,
    compatibility_level VARCHAR(50) DEFAULT 'BACKWARD',
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(schema_name, schema_version)
);

-- Create indexes for schemas
CREATE INDEX IF NOT EXISTS idx_schemas_name ON schemas(schema_name);
CREATE INDEX IF NOT EXISTS idx_schemas_name_version ON schemas(schema_name, schema_version);
CREATE INDEX IF NOT EXISTS idx_schemas_active ON schemas(is_active);

-- Create table for processing rules (used by Rule Engine)
CREATE TABLE IF NOT EXISTS processing_rules (
    id BIGSERIAL PRIMARY KEY,
    rule_name VARCHAR(255) UNIQUE NOT NULL,
    rule_version INTEGER NOT NULL DEFAULT 1,
    rule_type VARCHAR(50) NOT NULL,
    rule_definition TEXT NOT NULL,
    rule_config JSONB,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for processing rules
CREATE INDEX IF NOT EXISTS idx_rules_name ON processing_rules(rule_name);
CREATE INDEX IF NOT EXISTS idx_rules_type ON processing_rules(rule_type);
CREATE INDEX IF NOT EXISTS idx_rules_active ON processing_rules(is_active);

-- Create table for output configurations (used by Output Manager)
CREATE TABLE IF NOT EXISTS output_configs (
    id BIGSERIAL PRIMARY KEY,
    config_name VARCHAR(255) UNIQUE NOT NULL,
    sink_type VARCHAR(100) NOT NULL,
    sink_config JSONB NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for output configs
CREATE INDEX IF NOT EXISTS idx_output_configs_name ON output_configs(config_name);
CREATE INDEX IF NOT EXISTS idx_output_configs_type ON output_configs(sink_type);
CREATE INDEX IF NOT EXISTS idx_output_configs_active ON output_configs(is_active);

-- Create table for dead letter queue
CREATE TABLE IF NOT EXISTS dead_letter_queue (
    id BIGSERIAL PRIMARY KEY,
    original_event_id VARCHAR(255),
    tenant_id VARCHAR(255),
    error_type VARCHAR(255) NOT NULL,
    error_message TEXT,
    error_details JSONB,
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    next_retry_at TIMESTAMP WITH TIME ZONE,
    original_event_data JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for dead letter queue
CREATE INDEX IF NOT EXISTS idx_dlq_tenant_id ON dead_letter_queue(tenant_id);
CREATE INDEX IF NOT EXISTS idx_dlq_error_type ON dead_letter_queue(error_type);
CREATE INDEX IF NOT EXISTS idx_dlq_retry_count ON dead_letter_queue(retry_count);
CREATE INDEX IF NOT EXISTS idx_dlq_next_retry ON dead_letter_queue(next_retry_at);

-- Create partitions for events table (by month)
-- This will help with performance for time-series data
CREATE TABLE IF NOT EXISTS events_y2025m09 PARTITION OF events
FOR VALUES FROM ('2025-09-01') TO ('2025-10-01');

CREATE TABLE IF NOT EXISTS events_y2024m10 PARTITION OF events
FOR VALUES FROM ('2025-10-01') TO ('2025-11-01');

CREATE TABLE IF NOT EXISTS events_y2024m11 PARTITION OF events
FOR VALUES FROM ('2025-11-01') TO ('2025-12-01');

-- Add more partitions as needed...

-- Insert sample data for testing
INSERT INTO schemas (schema_name, schema_version, schema_type, schema_definition) VALUES
('user_event', 1, 'AVRO', '{"type":"record","name":"UserEvent","fields":[{"name":"userId","type":"string"},{"name":"action","type":"string"},{"name":"timestamp","type":"long"}]}'),
('order_event', 1, 'AVRO', '{"type":"record","name":"OrderEvent","fields":[{"name":"orderId","type":"string"},{"name":"userId","type":"string"},{"name":"amount","type":"double"},{"name":"timestamp","type":"long"}]}');

INSERT INTO processing_rules (rule_name, rule_type, rule_definition, rule_config) VALUES
('fraud_detection', 'CEP', 'SELECT * FROM user_events WHERE amount > 1000', '{"threshold": 1000, "window": "5m"}'),
('user_activity_aggregation', 'AGGREGATION', 'SELECT userId, COUNT(*) as activity_count FROM user_events GROUP BY userId', '{"window": "1h", "output": "user_activity_hourly"}');

INSERT INTO output_configs (config_name, sink_type, sink_config) VALUES
('elasticsearch_sink', 'ELASTICSEARCH', '{"hosts": ["http://elasticsearch:9200"], "index": "events"}'),
('webhook_sink', 'WEBHOOK', '{"url": "http://external-api/webhook", "method": "POST", "headers": {"Content-Type": "application/json"}}');

-- Grant permissions on all tables
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO eventuser;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO eventuser;
