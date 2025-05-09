CREATE TABLE IF NOT EXISTS images (
    id SERIAL PRIMARY KEY,
    prompt TEXT NOT NULL,
    uuid TEXT,
    status TEXT NOT NULL DEFAULT 'ReadyToGenerate',
    base64 TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_images_status ON images(status);
CREATE INDEX IF NOT EXISTS idx_images_uuid ON images(uuid); 