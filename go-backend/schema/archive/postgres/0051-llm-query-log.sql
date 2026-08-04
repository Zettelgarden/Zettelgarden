CREATE TABLE IF NOT EXISTS llm_query_log (
       id SERIAL PRIMARY KEY,
       user_id int,
       model text,
       prompt_tokens int,
       completion_tokens int,
       created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
       FOREIGN KEY (user_id) REFERENCES users(id)
);
