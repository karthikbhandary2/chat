ALTER TABLE users
  ALTER COLUMN id SET DATA TYPE UUID USING gen_random_uuid(),
  ALTER COLUMN id SET DEFAULT gen_random_uuid();

ALTER TABLE users RENAME COLUMN hashed_password TO password_hash;
ALTER TABLE users RENAME COLUMN create_at TO created_at;
ALTER TABLE users ALTER COLUMN created_at SET DATA TYPE TIMESTAMPTZ;