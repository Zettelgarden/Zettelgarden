-- Create a table to store sections from summarization analysis
CREATE TABLE IF NOT EXISTS summary_sections (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    card_pk INTEGER REFERENCES cards(id) ON DELETE CASCADE,
    summarization_id INTEGER NOT NULL REFERENCES summarizations(id) ON DELETE CASCADE,
    section_title TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(summarization_id, section_title)
);

-- Create a table to store theses from summarization analysis
CREATE TABLE IF NOT EXISTS summary_theses (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    card_pk INTEGER REFERENCES cards(id) ON DELETE CASCADE,
    summarization_id INTEGER NOT NULL REFERENCES summarizations(id) ON DELETE CASCADE,
    section_id INTEGER NOT NULL REFERENCES summary_sections(id) ON DELETE CASCADE,
    thesis TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create a table to store arguments from summarization analysis
CREATE TABLE IF NOT EXISTS summary_arguments (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    card_pk INTEGER REFERENCES cards(id) ON DELETE CASCADE,
    summarization_id INTEGER NOT NULL REFERENCES summarizations(id) ON DELETE CASCADE,
    thesis_id INTEGER NOT NULL REFERENCES summary_theses(id) ON DELETE CASCADE,
    argument TEXT NOT NULL,
    importance INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
