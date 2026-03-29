-- Adding a column is_notified to existing table saved_listings
ALTER TABLE saved_listings 
ADD COLUMN IF NOT EXISTS is_notified BOOLEAN DEFAULT FALSE;

-- Creating index for fast search of not notified listings
CREATE INDEX IF NOT EXISTS idx_saved_listings_is_notified 
ON saved_listings(is_notified);

-- Updating existing records
UPDATE saved_listings 
SET is_notified = TRUE 
WHERE is_notified IS NULL;