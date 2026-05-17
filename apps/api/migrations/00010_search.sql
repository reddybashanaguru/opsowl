-- Feature 11: Full-text search via PostgreSQL tsvector
-- Adds search_vector columns with trigger-based maintenance + GIN indexes

-- ─── Cases ────────────────────────────────────────────────────────────────────
ALTER TABLE cases ADD COLUMN search_vector tsvector;

CREATE OR REPLACE FUNCTION update_cases_search() RETURNS trigger AS $$
BEGIN
  NEW.search_vector :=
    setweight(to_tsvector('english', coalesce(NEW.case_number, '')), 'A') ||
    setweight(to_tsvector('english', coalesce(NEW.title, '')), 'A') ||
    setweight(to_tsvector('english', coalesce(NEW.description, '')), 'B') ||
    setweight(to_tsvector('english', array_to_string(NEW.tags, ' ')), 'C');
  RETURN NEW;
END
$$ LANGUAGE plpgsql;

CREATE TRIGGER cases_search_vector_update
  BEFORE INSERT OR UPDATE ON cases
  FOR EACH ROW EXECUTE FUNCTION update_cases_search();

UPDATE cases SET search_vector =
  setweight(to_tsvector('english', coalesce(case_number, '')), 'A') ||
  setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
  setweight(to_tsvector('english', coalesce(description, '')), 'B') ||
  setweight(to_tsvector('english', array_to_string(tags, ' ')), 'C');

CREATE INDEX idx_cases_search ON cases USING GIN (search_vector) WHERE deleted_at IS NULL;

-- ─── Notes ────────────────────────────────────────────────────────────────────
ALTER TABLE notes ADD COLUMN search_vector tsvector;

CREATE OR REPLACE FUNCTION update_notes_search() RETURNS trigger AS $$
BEGIN
  NEW.search_vector :=
    setweight(to_tsvector('english', coalesce(NEW.title, '')), 'A') ||
    setweight(to_tsvector('english', coalesce(NEW.content_text, '')), 'B');
  RETURN NEW;
END
$$ LANGUAGE plpgsql;

CREATE TRIGGER notes_search_vector_update
  BEFORE INSERT OR UPDATE ON notes
  FOR EACH ROW EXECUTE FUNCTION update_notes_search();

UPDATE notes SET search_vector =
  setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
  setweight(to_tsvector('english', coalesce(content_text, '')), 'B');

CREATE INDEX idx_notes_search ON notes USING GIN (search_vector) WHERE deleted_at IS NULL;

-- ─── Tasks ────────────────────────────────────────────────────────────────────
ALTER TABLE tasks ADD COLUMN search_vector tsvector;

CREATE OR REPLACE FUNCTION update_tasks_search() RETURNS trigger AS $$
BEGIN
  NEW.search_vector :=
    setweight(to_tsvector('english', coalesce(NEW.title, '')), 'A') ||
    setweight(to_tsvector('english', coalesce(NEW.description, '')), 'B');
  RETURN NEW;
END
$$ LANGUAGE plpgsql;

CREATE TRIGGER tasks_search_vector_update
  BEFORE INSERT OR UPDATE ON tasks
  FOR EACH ROW EXECUTE FUNCTION update_tasks_search();

UPDATE tasks SET search_vector =
  setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
  setweight(to_tsvector('english', coalesce(description, '')), 'B');

CREATE INDEX idx_tasks_search ON tasks USING GIN (search_vector) WHERE deleted_at IS NULL;

-- ─── Documents ────────────────────────────────────────────────────────────────
ALTER TABLE documents ADD COLUMN search_vector tsvector;

CREATE OR REPLACE FUNCTION update_documents_search() RETURNS trigger AS $$
BEGIN
  -- Replace underscores, hyphens, dots with spaces so "invoice_2024.pdf" hits "invoice"
  NEW.search_vector :=
    setweight(to_tsvector('english',
      regexp_replace(coalesce(NEW.original_filename, ''), '[_\-.]', ' ', 'g')
    ), 'A');
  RETURN NEW;
END
$$ LANGUAGE plpgsql;

CREATE TRIGGER documents_search_vector_update
  BEFORE INSERT OR UPDATE ON documents
  FOR EACH ROW EXECUTE FUNCTION update_documents_search();

UPDATE documents SET search_vector =
  setweight(to_tsvector('english',
    regexp_replace(coalesce(original_filename, ''), '[_\-.]', ' ', 'g')
  ), 'A');

CREATE INDEX idx_documents_search ON documents USING GIN (search_vector) WHERE deleted_at IS NULL;
