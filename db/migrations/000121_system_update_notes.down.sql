DELETE FROM admin_activities
WHERE method = 'RELEASE' AND route LIKE '/system_updates/%';
