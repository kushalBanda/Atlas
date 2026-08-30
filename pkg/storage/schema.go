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
	span_kind            VARCHAR,
	level                VARCHAR,
	llm_model            VARCHAR,
	llm_prompt_tokens    BIGINT,
	llm_completion_tokens BIGINT,
	llm_cost             DOUBLE,
	llm_temperature      DOUBLE,
	llm_top_p            DOUBLE,
	llm_max_tokens       BIGINT,
	llm_usage_details    VARCHAR,
	llm_cost_details     VARCHAR,
	llm_time_to_first_token_nano BIGINT,
	llm_prompt_id        VARCHAR,
	llm_prompt_name      VARCHAR,
	llm_prompt_version   BIGINT,
	session_id           VARCHAR,
	user_id              VARCHAR,
	agent_run_id         VARCHAR,
	agent_name           VARCHAR,
	agent_step_kind      VARCHAR,
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

// migrationDDL brings a spans table created before the agent-run columns
// existed up to the current shape. schemaDDL is CREATE TABLE IF NOT EXISTS,
// so it is a no-op against an existing atlas.duckdb file — without these
// ALTERs an upgraded binary would fail every insert on an old database.
// Each statement is idempotent, so this runs unconditionally at startup.
const migrationDDL = `
ALTER TABLE spans ADD COLUMN IF NOT EXISTS session_id VARCHAR;
ALTER TABLE spans ADD COLUMN IF NOT EXISTS user_id VARCHAR;
ALTER TABLE spans ADD COLUMN IF NOT EXISTS agent_run_id VARCHAR;
ALTER TABLE spans ADD COLUMN IF NOT EXISTS agent_name VARCHAR;
ALTER TABLE spans ADD COLUMN IF NOT EXISTS agent_step_kind VARCHAR;
`
