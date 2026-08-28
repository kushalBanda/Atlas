package storage

// schemaDDL creates the core tables Atlas owns directly. Plugin modules
// register additional tables through SchemaRegistrar at startup.
const schemaDDL = `
CREATE TABLE IF NOT EXISTS spans (
	trace_id             VARCHAR NOT NULL,
	span_id              VARCHAR NOT NULL,
	parent_span_id       VARCHAR,
	service_name         VARCHAR NOT NULL,
	name                 VARCHAR NOT NULL,
	start_time           TIMESTAMP NOT NULL,
	end_time             TIMESTAMP NOT NULL,
	status_code          VARCHAR NOT NULL,
	attributes           VARCHAR,
	resource_attributes  VARCHAR,
	llm_model            VARCHAR,
	llm_prompt_tokens    BIGINT,
	llm_completion_tokens BIGINT,
	llm_cost             DOUBLE,
	PRIMARY KEY (trace_id, span_id)
);

CREATE TABLE IF NOT EXISTS traces (
	trace_id                  VARCHAR PRIMARY KEY,
	first_seen                TIMESTAMP NOT NULL,
	last_seen                 TIMESTAMP NOT NULL,
	closed_at                 TIMESTAMP,
	likely_root_cause_span_id VARCHAR,
	reason                    VARCHAR,
	self_time_pct             DOUBLE
);
`
