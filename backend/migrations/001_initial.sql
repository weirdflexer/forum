CREATE TABLE sections (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), slug text NOT NULL UNIQUE CHECK (slug ~ '^[a-z0-9-]{2,40}$'),
 title text NOT NULL CHECK (char_length(title) BETWEEN 2 AND 80), description text NOT NULL DEFAULT '' CHECK (char_length(description)<=240),
 is_archived boolean NOT NULL DEFAULT false, created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE anonymous_sessions (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), token_hash text NOT NULL UNIQUE,
 created_at timestamptz NOT NULL DEFAULT now(), expires_at timestamptz NOT NULL, revoked_at timestamptz
);
CREATE TABLE staff_accounts (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), login text NOT NULL UNIQUE,
 password_hash text NOT NULL, role text NOT NULL CHECK(role IN ('admin','moderator')), is_active boolean NOT NULL DEFAULT true
);
CREATE TABLE staff_sessions (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), staff_id uuid NOT NULL REFERENCES staff_accounts ON DELETE CASCADE,
 token_hash text NOT NULL UNIQUE, expires_at timestamptz NOT NULL
);
CREATE TABLE topics (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), section_id uuid NOT NULL REFERENCES sections,
 author_session_id uuid REFERENCES anonymous_sessions ON DELETE SET NULL,
 title text NOT NULL CHECK (char_length(title) BETWEEN 5 AND 120), status text NOT NULL DEFAULT 'open' CHECK(status IN ('open','closed')),
 created_at timestamptz NOT NULL DEFAULT now(), last_activity_at timestamptz NOT NULL DEFAULT now(),
 search_vector tsvector GENERATED ALWAYS AS (to_tsvector('russian', title)) STORED
);
CREATE TABLE posts (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), topic_id uuid NOT NULL REFERENCES topics ON DELETE CASCADE,
 author_session_id uuid REFERENCES anonymous_sessions ON DELETE SET NULL, number integer NOT NULL CHECK(number>0),
 body text NOT NULL CHECK (char_length(body)<=5000), status text NOT NULL DEFAULT 'visible' CHECK(status IN ('visible','deleted','hidden')),
 created_at timestamptz NOT NULL DEFAULT now(),
 search_vector tsvector GENERATED ALWAYS AS (to_tsvector('russian', body)) STORED,
 UNIQUE(topic_id,number)
);
CREATE TABLE reports (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), post_id uuid NOT NULL REFERENCES posts,
 reporter_session_id uuid REFERENCES anonymous_sessions ON DELETE SET NULL,
 reason text NOT NULL CHECK(reason IN ('spam','abuse','personal_data','other')), comment text NOT NULL DEFAULT '' CHECK(char_length(comment)<=1000),
 status text NOT NULL DEFAULT 'open' CHECK(status IN ('open','dismissed','resolved')),
 decision_reason text NOT NULL DEFAULT '', created_at timestamptz NOT NULL DEFAULT now(), resolved_at timestamptz
);
CREATE UNIQUE INDEX reports_open_unique ON reports(reporter_session_id,post_id) WHERE status='open';
CREATE TABLE moderation_actions (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), staff_id uuid NOT NULL REFERENCES staff_accounts,
 post_id uuid REFERENCES posts, topic_id uuid REFERENCES topics, action text NOT NULL, reason text NOT NULL CHECK(char_length(reason) BETWEEN 1 AND 1000),
 created_at timestamptz NOT NULL DEFAULT now(), CHECK(num_nonnulls(post_id,topic_id)=1)
);
CREATE TABLE idempotency_keys (
 session_id uuid NOT NULL REFERENCES anonymous_sessions ON DELETE CASCADE, key text NOT NULL,
 request_hash text NOT NULL, response jsonb NOT NULL, status integer NOT NULL, expires_at timestamptz NOT NULL,
 PRIMARY KEY(session_id,key)
);
CREATE TABLE rate_events (
 session_id uuid NOT NULL REFERENCES anonymous_sessions ON DELETE CASCADE, action text NOT NULL, created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX topics_recent ON topics(last_activity_at DESC,id DESC);
CREATE INDEX topics_section ON topics(section_id,last_activity_at DESC,id DESC);
CREATE INDEX topics_search ON topics USING gin(search_vector);
CREATE INDEX posts_search ON posts USING gin(search_vector) WHERE status='visible';
CREATE INDEX posts_author ON posts(author_session_id);
CREATE INDEX reports_queue ON reports(status,created_at);
CREATE INDEX rate_events_lookup ON rate_events(session_id,action,created_at);
CREATE INDEX sessions_expiry ON anonymous_sessions(expires_at);
