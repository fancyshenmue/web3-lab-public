-- Remove seed identity providers
DELETE FROM identity_providers WHERE provider_id IN ('google', 'github', 'facebook', 'apple', 'email', 'eoa');
