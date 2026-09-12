-- Tracks each byte range accepted for an in-progress chunked upload, so
-- chunks can be sent (and retried) out of order — a handful in flight at
-- once — instead of the server enforcing strict sequential arrival.
CREATE TABLE IF NOT EXISTS upload_session_chunks (
    session_id   UUID NOT NULL REFERENCES upload_sessions(id) ON DELETE CASCADE,
    start_offset BIGINT NOT NULL,
    end_offset   BIGINT NOT NULL,
    PRIMARY KEY (session_id, start_offset)
);
