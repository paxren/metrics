--
CREATE TABLE metrics (
    id VARCHAR(255) PRIMARY KEY,
    mtype VARCHAR(20) NOT NULL CHECK (mtype IN ('counter', 'gauge')),
    delta BIGINT,
    value DOUBLE PRECISION,
    hash VARCHAR(64)
);