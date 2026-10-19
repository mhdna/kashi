-- Off by default: when on, a daily job deletes unknown-item pictures whose
-- units were all matched 30+ days ago, and pictures of checkouts that never
-- became a sale 2+ days after their link expired.
ALTER TABLE app_settings ADD COLUMN unknown_images_auto_delete boolean NOT NULL DEFAULT false;
